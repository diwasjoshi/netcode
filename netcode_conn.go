package netcode

import (
	"log"
	"net"

	"inet.af/netaddr"
)

type NetcodeData struct {
	data []byte
	from *netaddr.IPPort
}

const (
	SOCKET_RCVBUF_SIZE = 2048 * 2048
	SOCKET_SNDBUF_SIZE = 2048 * 2048
)

type NetcodeRecvHandler func(data *NetcodeData)

type NetcodeConn struct {
	conn      *net.UDPConn
	reuseAddr *net.UDPAddr
	closeCh   chan struct{}
	isClosed  bool

	recvSize int
	sendSize int
	maxBytes int

	recvHandlerFn NetcodeRecvHandler
}

func NewNetcodeConn() *NetcodeConn {
	c := &NetcodeConn{}

	c.closeCh = make(chan struct{})
	c.isClosed = true
	c.maxBytes = MAX_PACKET_BYTES
	c.recvSize = SOCKET_RCVBUF_SIZE
	c.sendSize = SOCKET_SNDBUF_SIZE
	c.reuseAddr = &net.UDPAddr{}
	return c
}

func (c *NetcodeConn) SetRecvHandler(recvHandlerFn NetcodeRecvHandler) {
	c.recvHandlerFn = recvHandlerFn
}

func (c *NetcodeConn) Write(b []byte) (int, error) {
	if c.isClosed {
		return -1, ErrWriteClosedSocket
	}
	return c.conn.Write(b)
}

func (c *NetcodeConn) WriteTo(b []byte, to *netaddr.IPPort) (int, error) {
	if c.isClosed {
		return -1, ErrWriteClosedSocket
	}
	return c.conn.WriteTo(b, to.UDPAddrAt(c.reuseAddr))
}

func (c *NetcodeConn) Close() error {
	if !c.isClosed {
		close(c.closeCh)
	}
	c.isClosed = true

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *NetcodeConn) SetReadBuffer(bytes int) {
	c.recvSize = bytes
}

func (c *NetcodeConn) SetWriteBuffer(bytes int) {
	c.sendSize = bytes
}

// LocalAddr returns the local network address.
func (c *NetcodeConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *NetcodeConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *NetcodeConn) Dial(address *netaddr.IPPort) error {
	var err error

	if c.recvHandlerFn == nil {
		return ErrPacketHandlerBeforeListen
	}

	c.closeCh = make(chan struct{})
	c.conn, err = net.DialUDP("udp", nil, address.UDPAddrAt(c.reuseAddr))
	if err != nil {
		return err
	}
	return c.create()
}

func (c *NetcodeConn) Listen(address *netaddr.IPPort) error {
	var err error

	if c.recvHandlerFn == nil {
		return ErrPacketHandlerBeforeListen
	}

	c.conn, err = net.ListenUDP("udp", address.UDPAddrAt(c.reuseAddr))
	if err != nil {
		return err
	}

	c.create()
	return err
}

func (c *NetcodeConn) create() error {
	c.isClosed = false
	c.conn.SetReadBuffer(c.recvSize)
	c.conn.SetWriteBuffer(c.sendSize)
	go c.readLoop()
	return nil
}

func (c *NetcodeConn) receiver(ch chan *NetcodeData) {
	for {

		if err := c.read(); err == nil {
			select {
			case <-c.closeCh:
				return
			default:
				continue
			}
		} else {
			if c.isClosed {
				return
			}
			log.Printf("error reading data from socket: %s\n", err)
		}

	}
}

// read does the actual connection read call, verifies we have a
// buffer > 0 and < maxBytes and is of a valid packet type before
// we bother to attempt to actually dispatch it to the recvHandlerFn.
func (c *NetcodeConn) read() error {
	var n int
	var from *net.UDPAddr
	var err error
	netData := &NetcodeData{}
	netData.data = make([]byte, c.maxBytes)

	n, from, err = c.conn.ReadFromUDP(netData.data)
	if err != nil {
		return err
	}

	if n == 0 {
		return ErrSocketZeroRecv
	}

	if n > c.maxBytes {
		return ErrPacketSizeMax
	}

	// check if it's a valid packet
	if NewPacket(netData.data) == nil {
		return ErrInvalidPacket
	}

	netData.data = netData.data[:n]
	ip, ok := netaddr.FromStdIP(from.IP)
	if !ok {
		return ErrInvalidIPAddress
	}

	ip = ip.WithZone(from.Zone)
	netData.from = &netaddr.IPPort{IP: ip, Port: uint16(from.Port)}

	c.recvHandlerFn(netData)
	return nil
}

// dispatch the NetcodeData to the bound recvHandler function.
func (c *NetcodeConn) readLoop() {
	dataCh := make(chan *NetcodeData)
	go c.receiver(dataCh)
}
