package utils

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

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

// GetPublicIP attempts to get the public IPv4 address using HTTP requests
func GetPublicIP() (string, error) {
	// Try multiple IP lookup services
	services := []string{
		"http://ipv4.icanhazip.com",
		"http://api.ipify.org",
		"http://ifconfig.me/ip",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				DualStack: false, // Disable IPv6
			}).DialContext,
		},
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		ip := strings.TrimSpace(string(body))
		// Validate that we got an IPv4 address
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil && parsedIP.To4() != nil {
			return ip, nil
		}
	}

	// Fallback to outbound IP if public IP lookup fails
	return GetOutboundIP()
}
