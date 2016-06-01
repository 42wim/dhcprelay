package main

import (
	dhcp "github.com/krolaw/dhcp4"

	"flag"
	"fmt"
	"net"
	"strings"
)

var dhcpServers []net.IP
var dhcpGIAddr net.IP

type DHCPHandler struct {
	m map[string]bool
}

func (h *DHCPHandler) ServeDHCP(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (d dhcp.Packet) {
	switch msgType {

	case dhcp.Discover:
		fmt.Println("discover ", p.YIAddr(), "from", p.CHAddr())
		h.m[string(p.XId())] = true
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetGIAddr(dhcpGIAddr)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.Offer:
		if !h.m[string(p.XId())] {
			return nil
		}
		var sip net.IP
		for k, v := range p.ParseOptions() {
			if k == dhcp.OptionServerIdentifier {
				sip = v
			}
		}
		fmt.Println("offering from", sip.String(), p.YIAddr(), "to", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.Request:
		h.m[string(p.XId())] = true
		fmt.Println("request ", p.YIAddr(), "from", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(dhcpGIAddr)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.ACK:
		if !h.m[string(p.XId())] {
			return nil
		}
		var sip net.IP
		for k, v := range p.ParseOptions() {
			if k == dhcp.OptionServerIdentifier {
				sip = v
			}
		}
		fmt.Println("ACK from", sip.String(), p.YIAddr(), "to", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.NAK:
		if !h.m[string(p.XId())] {
			return nil
		}
		fmt.Println("NAK from", p.SIAddr(), p.YIAddr(), "to", p.CHAddr())
		p2 := dhcp.NewPacket(dhcp.BootReply)
		p2.SetXId(p.XId())
		p2.SetFile(p.File())
		p2.SetFlags(p.Flags())
		p2.SetSIAddr(p.SIAddr())
		p2.SetYIAddr(p.YIAddr())
		p2.SetGIAddr(p.GIAddr())
		p2.SetCHAddr(p.CHAddr())
		p2.SetSecs(p.Secs())
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2

	case dhcp.Release, dhcp.Decline:
		p2 := dhcp.NewPacket(dhcp.BootRequest)
		p2.SetCHAddr(p.CHAddr())
		p2.SetFile(p.File())
		p2.SetCIAddr(p.CIAddr())
		p2.SetSIAddr(p.SIAddr())
		p2.SetGIAddr(dhcpGIAddr)
		p2.SetXId(p.XId())
		p2.SetBroadcast(false)
		for k, v := range p.ParseOptions() {
			p2.AddOption(k, v)
		}
		return p2
	}
	return nil
}

func createRelay(in, out string) {
	handler := &DHCPHandler{m: make(map[string]bool)}
	go ListenAndServeIf(in, out, 67, handler)
	ListenAndServeIf(out, in, 68, handler)
}

func main() {
	flagInInt := flag.String("in", "eth0", "Interface to listen for DHCPv4/BOOTP queries on")
	flagOutInt := flag.String("out", "eth1", "Outgoing interface (to relay to dhcpservers)")
	flagServers := flag.String("destination", "", "ip1 ip2 (ip addresses of the dhcpservers")
	flagBindIP := flag.String("giaddr", "", "required ip address (of outgoing interface and to be used as GIADDR")
	flag.Parse()
	servers := strings.Fields(*flagServers)
	for _, s := range servers {
		dhcpServers = append(dhcpServers, net.ParseIP(s))
	}
	dhcpGIAddr = net.ParseIP(*flagBindIP)
	if dhcpGIAddr == nil {
		panic("giaddr needed")
	}
	createRelay(*flagInInt, *flagOutInt)
}
