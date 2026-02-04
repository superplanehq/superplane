package registry

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

var blockedHosts = []string{
	// Cloud metadata endpoints
	"metadata.google.internal",
	"metadata.goog",
	"metadata.azure.com",
	"169.254.169.254",
	"fd00:ec2::254",
	// Kubernetes API
	"kubernetes.default",
	"kubernetes.default.svc",
	"kubernetes.default.svc.cluster.local",
	// Localhost variations
	"localhost",
	"127.0.0.1",
	"::1",
	"0.0.0.0",
}

var privateIPRanges = []string{
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
}

var parsedPrivateRanges []*net.IPNet

func init() {
	for _, cidr := range privateIPRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			parsedPrivateRanges = append(parsedPrivateRanges, ipNet)
		}
	}
}

// ValidateURLForSSRF performs URL-level SSRF checks (scheme, blocked hostnames).
// IP-level checks are performed at connection time by the dialer's Control function
// to prevent DNS rebinding attacks.
func ValidateURLForSSRF(targetURL string) error {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}

	host := parsedURL.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a host")
	}

	// Check blocked hostnames
	hostLower := strings.ToLower(host)
	for _, blocked := range blockedHosts {
		if hostLower == blocked || strings.HasSuffix(hostLower, "."+blocked) {
			return fmt.Errorf("access to %s is not allowed for security reasons", host)
		}
	}

	// If host is an IP address, validate it immediately
	if ip := net.ParseIP(host); ip != nil {
		if err := validateIP(ip); err != nil {
			return err
		}
	}
	// For hostnames, IP validation happens at connection time via the dialer

	return nil
}

func validateIP(ip net.IP) error {
	// Handle IPv4-mapped IPv6 addresses (e.g., ::ffff:127.0.0.1)
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	for _, ipNet := range parsedPrivateRanges {
		if ipNet.Contains(ip) {
			return fmt.Errorf("access to private IP addresses is not allowed for security reasons")
		}
	}
	return nil
}

// ssrfSafeDialer creates a dialer that validates IP addresses at connection time.
// This prevents DNS rebinding attacks by checking the resolved IP just before connecting.
func ssrfSafeDialer() *net.Dialer {
	return &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("invalid address: %w", err)
			}

			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("invalid IP address: %s", host)
			}

			if err := validateIP(ip); err != nil {
				return fmt.Errorf("connection blocked: %w", err)
			}

			return nil
		},
	}
}

func NewSSRFSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := ssrfSafeDialer()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}

			// Validate URL-level checks (scheme, blocked hostnames)
			if err := ValidateURLForSSRF(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}

			return nil
		},
	}
}
