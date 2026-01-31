// Package security provides security utilities for the namazu application.
// This includes SSRF prevention, URL validation, and rate limiting helpers.
package security

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// IsPrivateIP checks if the given IP address is a private, localhost, or link-local address.
// Returns true for:
// - Private ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
// - Localhost: 127.0.0.0/8, ::1
// - Link-local: 169.254.0.0/16
// Returns false for public IPs and invalid IP strings.
func IsPrivateIP(ipStr string) bool {
	if ipStr == "" {
		return false
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check for localhost (127.0.0.0/8)
	if ip.IsLoopback() {
		return true
	}

	// Check for private IPs (10.x.x.x, 172.16-31.x.x, 192.168.x.x)
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local addresses (169.254.x.x for IPv4, fe80::/10 for IPv6)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return false
}

// IsLocalhost checks if the given host is localhost.
// Accepts: "localhost", "127.0.0.1", "::1", "[::1]", "0.0.0.0"
func IsLocalhost(host string) bool {
	// Remove brackets from IPv6
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")

	switch host {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	}

	// Check if it's a 127.x.x.x address
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}

	return false
}

// ValidateWebhookURL validates a webhook URL for SSRF vulnerabilities.
// It checks:
// - URL is valid and has http/https scheme
// - HTTPS is required (unless allowLocal is true and host is localhost)
// - Host is not a private IP address
// - Host is not localhost (unless allowLocal is true)
//
// Parameters:
// - urlStr: The URL to validate
// - allowLocal: If true, allows localhost URLs with HTTP (for development)
//
// Returns nil if the URL is safe, or an error describing the issue.
func ValidateWebhookURL(urlStr string, allowLocal bool) error {
	if urlStr == "" {
		return fmt.Errorf("URL is empty")
	}

	// Parse the URL
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Validate scheme
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %q (only http and https are allowed)", parsed.Scheme)
	}

	// Extract host without port
	host := ExtractHostWithoutPort(parsed.Host)
	if host == "" {
		return fmt.Errorf("invalid URL: missing host")
	}

	isLocal := IsLocalhost(host)

	// Check HTTPS requirement
	if scheme == "http" {
		if !allowLocal || !isLocal {
			if isLocal {
				return fmt.Errorf("localhost URLs are not allowed in production")
			}
			return fmt.Errorf("HTTPS is required for webhook URLs")
		}
	}

	// Block localhost unless allowed
	if isLocal && !allowLocal {
		return fmt.Errorf("localhost URLs are not allowed")
	}

	// Check for private IP addresses
	ip := net.ParseIP(host)
	if ip != nil && IsPrivateIP(host) {
		// Allow localhost if explicitly permitted
		if isLocal && allowLocal {
			return nil
		}
		return fmt.Errorf("private IP addresses are not allowed")
	}

	return nil
}
