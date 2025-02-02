package discovery

import (
	"encoding/json"
	"fmt"
	"net"
)

// Result represents a single discovery result
type Result struct {
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Host      string `json:"host"`
	Url       string `json:"url"`
	Raw       []byte `json:"-"`
	Error     error  `json:"-"`
}

// IpPort returns the IP and Port in a "ip:port" format
func (result *Result) IpPort() string {
	return net.JoinHostPort(result.IP, fmt.Sprint(result.Port))
}

// HostPort returns the Host and Port in a "host:port" format
func (result *Result) HostPort() string {
	return net.JoinHostPort(result.Host, fmt.Sprint(result.Port))
}

// RawData returns the raw data as a string
func (result *Result) RawData() string {
	return string(result.Raw)
}

// JSON returns the JSON representation of the result
func (result *Result) JSON() string {
	data, _ := json.Marshal(result)
	return string(data)
}
