package xnetwork

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

type ConnAddrType struct {
	Transport string
	Addr      string
	Port      string
}

func (c *ConnAddrType) String() string {
	return fmt.Sprintf("%s://%s:%s", c.Transport, c.Addr, c.Port)
}

type Response struct {
	Conn net.Conn

	Request     *Request
	RawResponse []byte
	Length      int
	ReadAt      time.Time

	LocalAddr  string
	RemoteAddr string
	Network    string
}

func (r *Response) SourceAddr() *ConnAddrType {
	return r.makeConnAddr(r.LocalAddr)
}

func (r *Response) DestinationAddr() *ConnAddrType {
	return r.makeConnAddr(r.RemoteAddr)
}

func (r *Response) GetLatency() time.Duration {
	return r.ReadAt.Sub(r.Request.sendAt)
}

func (r *Response) GetRaw() []byte {
	return r.RawResponse
}

func (r *Response) makeConnAddr(addrString string) *ConnAddrType {
	network := r.Network
	switch network {
	case "tcp", "tcp4", "tcp6":
		tcpAddr, err := net.ResolveTCPAddr(network, addrString)
		if err != nil {
			return nil
		}
		return &ConnAddrType{
			Transport: network,
			Addr:      string(tcpAddr.IP),
			Port:      strconv.Itoa(tcpAddr.Port),
		}
	case "udp", "udp4", "udp6":
		udpAddr, err := net.ResolveUDPAddr(network, addrString)
		if err != nil {
			return nil
		}
		return &ConnAddrType{
			Transport: network,
			Addr:      string(udpAddr.IP),
			Port:      strconv.Itoa(udpAddr.Port),
		}
	default:
		return nil
	}
}
