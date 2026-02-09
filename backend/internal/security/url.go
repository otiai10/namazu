package security

import (
	"fmt"
	"net/url"
	"strings"
)

// RequireHTTPS validates that a URL uses HTTPS scheme.
// If allowLocal is true, HTTP is allowed for localhost addresses only.
//
// Parameters:
// - urlStr: The URL to validate
// - allowLocal: If true, allows HTTP for localhost (development mode)
//
// Returns nil if the URL scheme is valid, or an error describing the issue.
func RequireHTTPS(urlStr string, allowLocal bool) error {
	if urlStr == "" {
		return fmt.Errorf("URL is empty")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)

	// Only allow http and https schemes
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %q (only http and https are allowed)", parsed.Scheme)
	}

	// HTTPS is always allowed
	if scheme == "https" {
		return nil
	}

	// HTTP is only allowed for localhost when allowLocal is true
	if scheme == "http" {
		if !allowLocal {
			return fmt.Errorf("HTTPS is required for webhook URLs")
		}

		host := ExtractHostWithoutPort(parsed.Host)
		if !IsLocalhost(host) {
			return fmt.Errorf("HTTP is only allowed for localhost URLs")
		}
	}

	return nil
}

// ParseAndValidateURL parses a URL string and validates it has required components.
// Returns an error if the URL is empty, malformed, or missing scheme/host.
func ParseAndValidateURL(urlStr string) (*url.URL, error) {
	if urlStr == "" {
		return nil, fmt.Errorf("URL is empty")
	}

	parsed, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid URL: missing scheme")
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL: missing host")
	}

	return parsed, nil
}

// ExtractHostWithoutPort extracts the hostname from a host:port string.
// Handles IPv6 addresses with brackets correctly.
//
// Examples:
//   - "example.com:8443" -> "example.com"
//   - "example.com" -> "example.com"
//   - "[::1]:8080" -> "::1"
//   - "[::1]" -> "::1"
//   - "192.168.1.1:8080" -> "192.168.1.1"
func ExtractHostWithoutPort(host string) string {
	if host == "" {
		return ""
	}

	// Handle IPv6 addresses with brackets
	if strings.HasPrefix(host, "[") {
		// Find the closing bracket
		endBracket := strings.Index(host, "]")
		if endBracket > 0 {
			// Return the address without brackets
			return host[1:endBracket]
		}
	}

	// Split by colon to remove port (for IPv4 and hostnames)
	// Note: This only works correctly for IPv4 and hostnames, not IPv6 without brackets
	colonIndex := strings.LastIndex(host, ":")
	if colonIndex > 0 {
		// Check if this looks like a port (all digits after colon)
		possiblePort := host[colonIndex+1:]
		isPort := true
		for _, c := range possiblePort {
			if c < '0' || c > '9' {
				isPort = false
				break
			}
		}
		if isPort && len(possiblePort) > 0 {
			return host[:colonIndex]
		}
	}

	return host
}
