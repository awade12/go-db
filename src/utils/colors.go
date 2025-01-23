package utils

import (
	"crypto/rand"
	"encoding/base64"
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
	ResetColor = color.New(color.Reset).SprintFunc()
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

// GenerateSecurePassword generates a cryptographically secure password
func GenerateSecurePassword() string {
	// Define character sets
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers := "0123456789"
	symbols := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	// Ensure at least one character from each set
	password := make([]byte, 20) // 20 characters total
	password[0] = lowercase[secureRandomInt(len(lowercase))]
	password[1] = uppercase[secureRandomInt(len(uppercase))]
	password[2] = numbers[secureRandomInt(len(numbers))]
	password[3] = symbols[secureRandomInt(len(symbols))]

	// All possible characters for remaining positions
	allChars := lowercase + uppercase + numbers + symbols

	// Fill remaining positions with random characters
	for i := 4; i < 20; i++ {
		password[i] = allChars[secureRandomInt(len(allChars))]
	}

	// Shuffle the password to avoid predictable character positions
	shuffleBytes(password)

	return string(password)
}

// secureRandomInt generates a cryptographically secure random integer in range [0, max)
func secureRandomInt(max int) int {
	var b [4]byte
	_, err := rand.Read(b[:])
	if err != nil {
		panic("failed to generate random number: " + err.Error())
	}
	return int(uint32(b[0])|uint32(b[1])<<8|uint32(b[2])<<16|uint32(b[3])<<24) % max
}

// shuffleBytes randomly shuffles a byte slice using Fisher-Yates algorithm
func shuffleBytes(b []byte) {
	for i := len(b) - 1; i > 0; i-- {
		j := secureRandomInt(i + 1)
		b[i], b[j] = b[j], b[i]
	}
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic("failed to generate random string: " + err.Error())
	}
	return base64.URLEncoding.EncodeToString(b)[:length]
}
