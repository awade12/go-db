package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/fatih/color"
)

var (
	Success  = color.New(color.FgGreen, color.Bold).SprintFunc()
	Info     = color.New(color.FgCyan).SprintFunc()
	Warn     = color.New(color.FgYellow).SprintFunc()
	ErrColor = color.New(color.FgRed, color.Bold).SprintFunc()
)

// GetOutboundIP gets the preferred outbound IP of this machine
func GetOutboundIP() (string, error) {
	// Dial a UDP connection to a reliable IP (Google's DNS)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", fmt.Errorf("failed to get outbound IP: %v", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetPublicIP attempts to get the public IP address
func GetPublicIP() (string, error) {
	// Try multiple IP lookup services
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me",
		"https://icanhazip.com",
	}

	for _, service := range services {
		resp, err := net.Dial("tcp", strings.TrimPrefix(service, "https://")+":443")
		if err != nil {
			continue
		}
		defer resp.Close()

		localAddr := resp.LocalAddr().(*net.TCPAddr)
		return localAddr.IP.String(), nil
	}

	// Fallback to outbound IP if public IP lookup fails
	return GetOutboundIP()
}
