package xnetwork

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/kataras/golog"
	"github.com/totem-project/xtls"
	"net"
	"sync"
	"time"
)

const (
	ReuseConn = iota
	ReturnConn
	CloseConn
)

const connPoolTimeout = DefaultIdleTimeout * 2

type ResponseMiddleware func(*Response, *Client) error

type Client struct {
	Debug            bool
	options          *ClientOptions
	afterResponse    []ResponseMiddleware
	connPools        map[string]*ConnPool
	connPoolsAccTime map[string]time.Time
	sync.Mutex
}

var defaultClient *Client

func Init() {
	defaultClient = NewClient(GetNetOptions())
}

func NewClient(options *ClientOptions) *Client {
	client := &Client{
		options:          options,
		Debug:            options.Debug,
		afterResponse:    []ResponseMiddleware{responseLogger},
		connPools:        make(map[string]*ConnPool),
		connPoolsAccTime: make(map[string]time.Time),
	}
	go client.recycle()

	return client
}

func Do(ctx context.Context, request *Request) (*Response, error) {
	return defaultClient.Do(ctx, request, nil, ReuseConn, false)
}

func DoExplicit(ctx context.Context, request *Request, conn net.Conn, connOp int) (*Response, error) {
	return defaultClient.Do(ctx, request, conn, connOp, false)
}

func Write(ctx context.Context, request *Request) error {
	_, err := defaultClient.Do(ctx, request, nil, ReuseConn, true)
	return err
}

func WriteExplicit(ctx context.Context, request *Request, conn net.Conn, connOp int) error {
	_, err := defaultClient.Do(ctx, request, conn, connOp, true)
	return err
}

func PutConn(request *Request, conn net.Conn) bool {
	return defaultClient.PutConn(request, conn)
}

func (c *Client) Do(ctx context.Context, request *Request, conn net.Conn, connOp int, onlyWrite bool) (resp *Response, err error) {
	defer func() {
		if err != nil {
			golog.Debugf("xnetwork.Client.Do addr=%s@%s err=%v", request.Network, request.Address, err)
		}
	}()

	// step1. 检查参数
	if request.Network == "" || request.Address == "" {
		return nil, errors.New("invalid network/address")
	}

	// step2. 以network@address为单位，获取/新建连接池
	poolKey := request.Network + "@" + request.Address
	c.Lock()
	if _, ok := c.connPools[poolKey]; !ok {
		factory := func() (net.Conn, error) { return c.connect(request.IsTLS, request.Network, request.Address) }
		c.connPools[poolKey] = NewConnPool(poolKey, DefaultPoolCap, time.Second*time.Duration(c.options.ReadTimeout), DefaultIdleTimeout, factory)
	}
	connPool := c.connPools[poolKey]
	c.connPoolsAccTime[poolKey] = time.Now()
	c.Unlock()

	// step3. 以address为单位，limit qps
	limiter := ExtractQPSLimiter(ctx)
	err = limiter.Wait(ctx, request.Address)
	if err != nil {
		return nil, err
	}

	// step4. 从连接池中获取一个连接
	if conn == nil {
		conn, err = connPool.Get()
		if err != nil {
			return nil, err
		}
	}

	// step5. 回收/关闭/返回连接
	switch connOp {
	case ReuseConn:
		if err != nil {
			conn.Close()
		} else {
			defer connPool.Put(conn)
		}
	case CloseConn:
		defer conn.Close()
	case ReturnConn:
		defer func() {
			if err == nil && resp != nil {
				resp.Conn = conn
			} else {
				conn.Close()
			}
		}()
	}

	// step6. 发起请求(write), 读取响应(read)
	request.sendAt = time.Now()
	_, err = conn.Write(request.Raw)
	if err != nil || onlyWrite {
		return nil, err
	}

	readBuf := make([]byte, 20480)
	n, err := conn.Read(readBuf)
	if err != nil {
		return nil, err
	}

	// step7. 组织Response, 调用afterResponse插件
	resp = &Response{
		Request:     request,
		RawResponse: readBuf[:n],
		Length:      n,
		ReadAt:      time.Now(),
		LocalAddr:   conn.LocalAddr().String(),
		RemoteAddr:  conn.RemoteAddr().String(),
		Network:     conn.LocalAddr().Network(),
	}
	for _, f := range defaultClient.afterResponse {
		f(resp, defaultClient)
	}
	return resp, nil
}

func (c *Client) PutConn(request *Request, conn net.Conn) bool {
	c.Lock()
	defer c.Unlock()

	poolKey := request.Network + "@" + request.Address
	if connPool, ok := c.connPools[poolKey]; ok {
		c.connPoolsAccTime[poolKey] = time.Now()
		if err := connPool.Put(conn); err == nil {
			return true
		}
	}
	return false
}

func (c *Client) connect(isTLS bool, network string, address string) (net.Conn, error) {
	var (
		conn    net.Conn
		errDial error
	)
	dialer := &net.Dialer{
		Timeout:   time.Duration(c.options.DialTimeout) * time.Second,
		KeepAlive: time.Duration(c.options.KeepAlive) * time.Second,
	}

	attempt := 0
	for {
		if isTLS {
			tlsConfig, err := xtls.NewTLSConfig(c.options.TlsOptions)
			if err != nil {
				return nil, err
			}
			conn, errDial = tls.DialWithDialer(dialer, network, address, tlsConfig)
		} else {
			conn, errDial = dialer.Dial(network, address)
		}
		if errDial == nil {
			break
		}
		attempt++
		if c.options.FailRetries-attempt < 0 {
			break
		}
	}

	if errDial != nil {
		return nil, fmt.Errorf("giving up connect to %s %s after %d attempt(s) err=%v", network, address, attempt, errDial)
	}
	return conn, nil
}

func ConnpoolStatus() map[string]interface{} {
	return defaultClient.Status()
}

func (c *Client) Status() map[string]interface{} {
	c.Lock()
	defer c.Unlock()

	result := make(map[string]interface{})
	for key, t := range c.connPoolsAccTime {
		if c.connPools[key] != nil {
			result[key] = fmt.Sprintf("%d [%s]", c.connPools[key].Size(), t.Format("2006-01-02 15:04:05"))
		}
	}
	return result
}

func (c *Client) recycle() {
	ticker := time.NewTicker(connPoolTimeout * 2)
	for {
		<-ticker.C
		c.Lock()
		for key, t := range c.connPoolsAccTime {
			if t.Add(connPoolTimeout).Before(time.Now()) {
				connPool := c.connPools[key]
				if connPool != nil {
					connPool.Recycle()
				}
				delete(c.connPools, key)
				delete(c.connPoolsAccTime, key)
			}
		}
		c.Unlock()
	}
}
