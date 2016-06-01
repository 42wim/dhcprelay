package main

import (
	"fmt"
	dhcp "github.com/krolaw/dhcp4"
	"golang.org/x/net/ipv4"
	"net"
	"strconv"
)

type serveIfConn struct {
	ifIndex    int
	otherIndex int
	conn       *ipv4.PacketConn
	cm         *ipv4.ControlMessage
}

func (s *serveIfConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, s.cm, addr, err = s.conn.ReadFrom(b)
	// filter other interfaces
	if s.cm != nil && s.cm.IfIndex != s.ifIndex && s.cm.IfIndex != s.otherIndex {
		fmt.Println("dropping packet from ", s.cm.IfIndex)
		n = 0 // Packets < 240 are filtered in Serve().
	}
	return
}

func (s *serveIfConn) WriteTo(b []byte, addr net.Addr, src net.IP, index int) (n int, err error) {
	fmt.Println("writing with source", src, index)
	s.cm.Src = src
	s.cm.IfIndex = index
	return s.conn.WriteTo(b, s.cm, addr)
}

func ListenAndServeIf(interfaceName string, otherName string, port int, handler dhcp.Handler) error {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return err
	}
	fmt.Println("listen on ", interfaceName, iface.Index, port)
	other, err := net.InterfaceByName(otherName)
	if err != nil {
		return err
	}
	p := strconv.Itoa(port)
	l, err := net.ListenPacket("udp4", ":"+p)
	if err != nil {
		return err
	}
	defer l.Close()
	return ServeIf(iface.Index, other.Index, l, handler)
}

func ServeIf(ifIndex int, otherIndex int, conn net.PacketConn, handler dhcp.Handler) error {
	p := ipv4.NewPacketConn(conn)
	if err := p.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		return err
	}
	return Serve(&serveIfConn{ifIndex: ifIndex, otherIndex: otherIndex, conn: p}, handler)
}

func Serve(conn *serveIfConn, handler dhcp.Handler) error {
	buffer := make([]byte, 1500)
	for {
		n, _, err := conn.ReadFrom(buffer)
		if err != nil {
			return err
		}
		if n < 240 { // Packet too small to be DHCP
			continue
		}
		req := dhcp.Packet(buffer[:n])
		if req.HLen() > 16 { // Invalid size
			continue
		}
		options := req.ParseOptions()
		var reqType dhcp.MessageType
		if t := options[dhcp.OptionDHCPMessageType]; len(t) != 1 {
			continue
		} else {
			reqType = dhcp.MessageType(t[0])
			if reqType < dhcp.Discover || reqType > dhcp.Inform {
				continue
			}
		}
		if res := handler.ServeDHCP(req, reqType, options); res != nil {
			// request send upstream
			if res.OpCode() == 1 {
				fmt.Println("writing to dhcp server as client (using source port 68 (bootpc))")
				for _, ip := range dhcpServers {
					conn.WriteTo(res, &net.UDPAddr{IP: ip, Port: 67}, dhcpGIAddr, conn.otherIndex)
				}
			} else {
				fmt.Println("writing to client as server (using source port 67 (bootps))")
				conn.WriteTo(res, &net.UDPAddr{IP: net.IPv4bcast, Port: 68}, dhcpGIAddr, conn.ifIndex)
			}
		}
	}
}
