package util

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// IsValidIP validates if a given string is a valid IP address.
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidCIDR validates if a given string is a valid CIDR notation.
func IsValidCIDR(cidr string) bool {
	_, _, err := net.ParseCIDR(cidr)
	return err == nil
}

// ResolveHost attempts to resolve a hostname to an IP address.
func ResolveHost(hostname string) (string, error) {
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", err
	}
	return ips[0].String(), nil
}

// ScanPorts scans a given host for open ports within the specified range.
func ScanPorts(host string, startPort, endPort int) ([]int, error) {
	var openPorts []int

	if startPort < 1 || endPort > 65535 || startPort > endPort {
		return nil, errors.New("invalid port range")
	}

	for port := startPort; port <= endPort; port++ {
		address := fmt.Sprintf("%s:%d", host, port)
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			openPorts = append(openPorts, port)
			_ = conn.Close()
		}
	}

	return openPorts, nil
}

// DetectOS identifies the operating system of the destination host (basic).
func DetectOS(host string) (string, error) {
	cmd := exec.Command("nmap", "-O", host)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	outputStr := string(output)
	if strings.Contains(outputStr, "Windows") {
		return "Windows", nil
	} else if strings.Contains(outputStr, "Linux") {
		return "Linux", nil
	} else if strings.Contains(outputStr, "Mac OS") {
		return "Mac OS", nil
	}
	return "Unknown", nil
}

// SanitizeInput removes potentially harmful characters from user input.
func SanitizeInput(input string) string {
	cleanInput := strings.ReplaceAll(input, "|", "")
	cleanInput = strings.ReplaceAll(cleanInput, "&", "")
	cleanInput = strings.ReplaceAll(cleanInput, "`", "")
	cleanInput = strings.ReplaceAll(cleanInput, ";", "")
	return cleanInput
}

// GetLocalIP retrieves the local machine's IP address.
func GetLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}
	return "", errors.New("no valid IP address found")
}

// GetSystemInfo retrieves system information of the local machine.
func GetSystemInfo() string {
	return fmt.Sprintf("OS: %s, ARCH: %s", runtime.GOOS, runtime.GOARCH)
}

// WriteToFile writes data to a file.
func WriteToFile(filePath, content string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
