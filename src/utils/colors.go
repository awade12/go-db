package utils

import (
	"fmt"
	"net"

	"github.com/fatih/color"
)

var (
	Success  = color.New(color.FgGreen, color.Bold).SprintFunc()
	Info     = color.New(color.FgCyan).SprintFunc()
	Warn     = color.New(color.FgYellow).SprintFunc()
	ErrColor = color.New(color.FgRed, color.Bold).SprintFunc()
)

// GetOutboundIP gets the preferred outbound IPv4 address of this machine
func GetOutboundIP() (string, error) {
	// Get all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range ifaces {
		// Skip loopback and inactive interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			// Check if it's an IP network address
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6 addresses
			if ipNet.IP.To4() == nil {
				continue
			}

			// Skip loopback and link-local addresses
			if ipNet.IP.IsLoopback() || ipNet.IP.IsLinkLocalUnicast() {
				continue
			}

			return ipNet.IP.String(), nil
		}
	}

	// Fallback to the old method if no suitable interface is found
	conn, err := net.Dial("udp4", "8.8.8.8:80") // Use udp4 to force IPv4
	if err != nil {
		return "", fmt.Errorf("failed to get outbound IP: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetPublicIP attempts to get the public IPv4 address
func GetPublicIP() (string, error) {
	// Try multiple IP lookup services
	services := []string{
		"ipv4.icanhazip.com", // Explicitly IPv4
		"ipv4.whatismyip.akamai.com",
		"v4.ident.me",
	}

	for _, service := range services {
		dialer := &net.Dialer{
			DualStack: false, // Disable IPv6
		}

		conn, err := dialer.Dial("tcp4", service+":80") // Use tcp4 to force IPv4
		if err != nil {
			continue
		}
		defer conn.Close()

		localAddr := conn.LocalAddr().(*net.TCPAddr)
		ip := localAddr.IP.To4()
		if ip != nil {
			return ip.String(), nil
		}
	}

	// Fallback to outbound IP if public IP lookup fails
	return GetOutboundIP()
}
