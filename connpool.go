package xnetwork

import (
	"errors"
	"net"
	"time"

	"github.com/sony/gobreaker"
)

const (
	DefaultPoolCap     = 100
	DefaultIdleTimeout = time.Minute
)

type ConnPool struct {
	name        string
	conns       chan *idleConn
	idleTimeout time.Duration
	rwTimeout   time.Duration
	factory     func() (net.Conn, error)
	cb          *gobreaker.CircuitBreaker
}

type idleConn struct {
	conn net.Conn
	t    time.Time
}

func NewConnPool(name string, cap int, rwTimeout time.Duration, idleTimeout time.Duration, factory func() (net.Conn, error)) *ConnPool {
	return &ConnPool{
		name:        name,
		conns:       make(chan *idleConn, cap),
		idleTimeout: idleTimeout,
		rwTimeout:   rwTimeout,
		factory:     factory,
		cb:          gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: name}),
	}
}

func (c *ConnPool) Get() (net.Conn, error) {
	conn, err := c.get()
	if err != nil {
		return nil, err
	}

	err = conn.SetDeadline(time.Now().Add(c.rwTimeout))
	if err != nil {
		c.closeConn(conn)
		return nil, err
	}
	return conn, nil
}

func (c *ConnPool) get() (net.Conn, error) {
	for {
		select {
		case wrapConn := <-c.conns:
			if wrapConn == nil {
				return nil, errors.New("pool is closed")
			}
			if c.isTimeout(wrapConn) {
				c.closeConn(wrapConn.conn)
				continue
			}
			return wrapConn.conn, nil
		default:
			conn, err := c.cb.Execute(func() (interface{}, error) { return c.factory() })
			if err != nil {
				return nil, err
			}
			return conn.(net.Conn), nil
		}
	}
}

func (c *ConnPool) Put(conn net.Conn) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	select {
	case c.conns <- &idleConn{conn: conn, t: time.Now()}:
		return nil
	default:
		//fixme: 应该关闭最早的连接
		return c.closeConn(conn)
	}
}

func (c *ConnPool) Size() int {
	return len(c.conns)
}

func (c *ConnPool) Recycle() {
	for {
		select {
		case wrapConn := <-c.conns:
			if wrapConn == nil {
				return
			}
			if !c.isTimeout(wrapConn) {
				c.Put(wrapConn.conn) //put back
				return
			}
			c.closeConn(wrapConn.conn)
		default:
			return
		}
	}
}

func (c *ConnPool) closeConn(v net.Conn) error {
	return v.Close()
}

func (c *ConnPool) isTimeout(v *idleConn) bool {
	return c.idleTimeout > 0 && v.t.Add(c.idleTimeout).Before(time.Now())
}
