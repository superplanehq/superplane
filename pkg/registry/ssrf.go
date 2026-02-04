package registry

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
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

	hostLower := strings.ToLower(host)
	for _, blocked := range blockedHosts {
		if hostLower == blocked || strings.HasSuffix(hostLower, "."+blocked) {
			return fmt.Errorf("access to %s is not allowed for security reasons", host)
		}
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if err := validateIP(ip); err != nil {
			return err
		}
	} else {
		ips, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("DNS lookup failed for %s: %w", host, err)
		}

		for _, resolvedIP := range ips {
			if err := validateIP(resolvedIP); err != nil {
				return fmt.Errorf("access to %s is not allowed: %w", host, err)
			}
		}
	}

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

func NewSSRFSafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}

			if err := ValidateURLForSSRF(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}

			return nil
		},
	}
}
