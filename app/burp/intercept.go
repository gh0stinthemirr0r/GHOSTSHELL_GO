package main

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const (
	burpBIIDPattern   = "burpresults?biid="
	snaplen           = 1600
	defaultPort       = 80
	defaultBPF        = "tcp and dst port 80"
	externalProbeAddr = "8.8.8.8"
)

// InterceptBIID intercepts network packets to extract a BIID
func InterceptBIID(timeout time.Duration) (string, error) {
	stop := make(chan struct{})
	result := make(chan string)

	go func() {
		result <- interceptPackets(stop)
	}()

	select {
	case biid := <-result:
		return biid, nil
	case <-time.After(timeout):
		close(stop)
		return "", fmt.Errorf("timeout reached while intercepting BIID")
	}
}

func interceptPackets(stop chan struct{}) string {
	var biid string

	sourceIP, err := getSourceIP(net.ParseIP(externalProbeAddr))
	if err != nil {
		return ""
	}

	interfaceName, err := getInterfaceName(sourceIP)
	if err != nil {
		return ""
	}

	handle, err := pcap.OpenLive(interfaceName, snaplen, true, pcap.BlockForever)
	if err != nil {
		return ""
	}
	defer handle.Close()

	if err := handle.SetBPFFilter(defaultBPF); err != nil {
		return ""
	}

	packets := gopacket.NewPacketSource(handle, handle.LinkType()).Packets()
	for {
		select {
		case packet := <-packets:
			if !isValidPacket(packet) {
				continue
			}
			appLayer := packet.ApplicationLayer()
			if appLayer != nil {
				biid = extractBIID(appLayer.Payload())
				if biid != "" {
					return biid
				}
			}
		case <-stop:
			return ""
		}
	}
}

func isValidPacket(packet gopacket.Packet) bool {
	if packet == nil || packet.NetworkLayer() == nil || packet.TransportLayer() == nil {
		return false
	}

	tcp, ok := packet.TransportLayer().(*layers.TCP)
	if !ok || tcp.DstPort != layers.TCPPort(defaultPort) {
		return false
	}

	return packet.ApplicationLayer() != nil
}

func extractBIID(data []byte) string {
	start := strings.Index(string(data), burpBIIDPattern)
	if start == -1 {
		return ""
	}

	start += len(burpBIIDPattern)
	end := strings.Index(string(data[start:]), " ")
	if end == -1 {
		return string(data[start:])
	}

	return string(data[start : start+end])
}

func getSourceIP(dstIP net.IP) (net.IP, error) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: dstIP, Port: 53})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

func getInterfaceName(ip net.IP) (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), ip.String()+"/") {
				return iface.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no interface found for IP: %s", ip.String())
}
