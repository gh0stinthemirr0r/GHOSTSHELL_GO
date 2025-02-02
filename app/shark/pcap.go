package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/sirupsen/logrus"
)

type PacketData struct {
	Timestamp     time.Time `json:"timestamp"`
	SourceIP      string    `json:"source_ip"`
	DestinationIP string    `json:"destination_ip"`
	Protocol      string    `json:"protocol"`
	Length        int       `json:"length"`
}

// CaptureOptions defines the options for packet capture.
type CaptureOptions struct {
	InterfaceName string
	Duration      int
	Filter        string // e.g., "tcp and port 80"
	ExportFile    string // Path to save captured packets
	Logger        *logrus.Logger
}

// Capture starts a packet capture session with advanced features.
func Capture(options CaptureOptions) {
	options.Logger.Infof("Starting PCAP capture on interface: %s for %d seconds", options.InterfaceName, options.Duration)

	// Open device
	handle, err := pcap.OpenLive(options.InterfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		options.Logger.Errorf("Failed to open device %s: %v", options.InterfaceName, err)
		return
	}
	defer handle.Close()

	// Apply filter if specified
	if options.Filter != "" {
		if err := handle.SetBPFFilter(options.Filter); err != nil {
			options.Logger.Errorf("Failed to set BPF filter: %v", err)
			return
		}
		options.Logger.Infof("Applied filter: %s", options.Filter)
	}

	// Create a pcap file to save packets if specified
	filename := options.ExportFile
	if filename == "" {
		filename = fmt.Sprintf("capture_%s_%d.pcap", options.InterfaceName, time.Now().Unix())
	}
	f, err := os.Create(filename)
	if err != nil {
		options.Logger.Errorf("Failed to create pcap file: %v", err)
		return
	}
	defer f.Close()

	writer := pcapgo.NewWriter(f)
	writer.WriteFileHeader(1600, handle.LinkType())

	// Set up a timeout for the capture
	timeout := time.Duration(options.Duration) * time.Second
	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	// Start capturing packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()

	// Real-time packet inspection and logging
	for {
		select {
		case packet, ok := <-packets:
			if !ok {
				options.Logger.Info("Packet capture stopped.")
				return
			}

			inspectPacket(packet, options.Logger)

			if err := writer.WritePacket(packet.Metadata().CaptureInfo, packet.Data()); err != nil {
				options.Logger.Errorf("Failed to write packet: %v", err)
			}

		case <-timeoutTimer.C:
			options.Logger.Infof("PCAP capture on %s completed. Saved to %s", options.InterfaceName, filename)
			return
		}
	}
}

// inspectPacket performs real-time inspection of packets and logs key details.
func inspectPacket(packet gopacket.Packet, logger *logrus.Logger) {
	if packet == nil {
		return
	}

	networkLayer := packet.NetworkLayer()
	if networkLayer == nil {
		return
	}

	transportLayer := packet.TransportLayer()
	if transportLayer == nil {
		return
	}

	applicationLayer := packet.ApplicationLayer()

	src, dst := networkLayer.NetworkFlow().Endpoints()
	protocol := "unknown"
	if tcp, ok := transportLayer.(*layers.TCP); ok {
		protocol = fmt.Sprintf("TCP/%d", tcp.DstPort)
	} else if udp, ok := transportLayer.(*layers.UDP); ok {
		protocol = fmt.Sprintf("UDP/%d", udp.DstPort)
	}

	packetData := PacketData{
		Timestamp:     packet.Metadata().Timestamp,
		SourceIP:      src.String(),
		DestinationIP: dst.String(),
		Protocol:      protocol,
		Length:        len(packet.Data()),
	}

	if applicationLayer != nil {
		logger.Debugf("Application payload: %s", string(applicationLayer.Payload()))
	}

	jsonData, _ := json.MarshalIndent(packetData, "", "  ")
	logger.Infof("Packet: %s", string(jsonData))
}

// ListInterfaces lists all available network interfaces for packet capture.
func ListInterfaces(logger *logrus.Logger) ([]pcap.Interface, error) {
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		logger.Errorf("Failed to list interfaces: %v", err)
		return nil, err
	}

	for _, iface := range ifs {
		logger.Infof("Interface: %s (%s)", iface.Name, iface.Description)
	}

	return ifs, nil
}
