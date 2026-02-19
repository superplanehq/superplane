package registry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

type HTTPContext struct {
	client           *http.Client
	dialer           *net.Dialer
	blockedHosts     []string
	privateIPRanges  []*net.IPNet
	maxResponseBytes int64
}

type HTTPOptions struct {
	BlockedHosts     []string
	PrivateIPRanges  []string
	MaxResponseBytes int64
}

func NewHTTPContext(options HTTPOptions) (*HTTPContext, error) {
	httpCtx := &HTTPContext{
		blockedHosts:     options.BlockedHosts,
		privateIPRanges:  make([]*net.IPNet, 0),
		maxResponseBytes: options.MaxResponseBytes,
	}

	for _, cidr := range options.PrivateIPRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid private IP range: %w", err)
		}

		httpCtx.privateIPRanges = append(httpCtx.privateIPRanges, ipNet)
	}

	//
	// Creates a new HTTP dialer that validates IP addresses at connection time.
	// This prevents DNS rebinding attacks by checking the resolved IP just before connecting.
	//
	httpCtx.dialer = &net.Dialer{
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

			if err := httpCtx.validateIP(ip); err != nil {
				return fmt.Errorf("connection blocked: %w", err)
			}

			return nil
		},
	}

	// Use a longer timeout so slow external APIs (e.g. GCP Pub/Sub, Logging) can complete during webhook provisioning.
	httpCtx.client = &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return httpCtx.dialer.DialContext(ctx, network, addr)
			},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}

			if err := httpCtx.validateURL(req.URL); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}

			return nil
		},
	}

	return httpCtx, nil
}

func (c *HTTPContext) Do(request *http.Request) (*http.Response, error) {
	if len(c.privateIPRanges) == 0 && len(c.blockedHosts) == 0 {
		return c.do(request)
	}

	err := c.validateURL(request.URL)
	if err != nil {
		return nil, err
	}

	return c.do(request)
}

func (c *HTTPContext) do(request *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}

	if c.maxResponseBytes <= 0 {
		return resp, nil
	}

	//
	// Content-Length is not truly reliable,
	// but it's a good first check to enforce the maximum response size.
	//
	if resp.ContentLength > c.maxResponseBytes {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("response too large: %d bytes exceeds maximum size of %d bytes", resp.ContentLength, c.maxResponseBytes)
	}

	//
	// We replace the body with a LimitedReadCloser that will return an error
	// if the response body is larger than the maximum allowed size.
	//
	resp.Body = &LimitedReadCloser{
		reader:          resp.Body,
		remaining:       c.maxResponseBytes,
		maxResponseSize: c.maxResponseBytes,
	}

	return resp, nil
}

type LimitedReadCloser struct {
	reader          io.ReadCloser
	remaining       int64
	maxResponseSize int64
}

func (r *LimitedReadCloser) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		var buf [1]byte
		n, err := r.reader.Read(buf[:])
		if n > 0 {
			return 0, fmt.Errorf("response too large: exceeds maximum size of %d bytes", r.maxResponseSize)
		}
		return 0, err
	}

	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}

	n, err := r.reader.Read(p)
	r.remaining -= int64(n)
	return n, err
}

func (r *LimitedReadCloser) Close() error {
	return r.reader.Close()
}

/*
 * Performs URL-level SSRF checks (scheme, blocked hostnames).
 * IP-level checks are performed at connection time by the dialer's Control function
 * to prevent DNS rebinding attacks.
 */
func (c *HTTPContext) validateURL(URL *url.URL) error {
	scheme := strings.ToLower(URL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}

	host := URL.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a host")
	}

	//
	// Check blocked hostnames
	//
	hostLower := strings.ToLower(host)
	for _, blocked := range c.blockedHosts {
		if hostLower == blocked || strings.HasSuffix(hostLower, "."+blocked) {
			return fmt.Errorf("access to %s is not allowed", host)
		}
	}

	//
	// If host is an IP address, validate it immediately
	// For hostnames, IP validation happens at connection time via the dialer.
	//
	if ip := net.ParseIP(host); ip != nil {
		if err := c.validateIP(ip); err != nil {
			return err
		}
	}

	return nil
}

func (c *HTTPContext) validateIP(ip net.IP) error {
	//
	// Handle IPv4-mapped IPv6 addresses (e.g., ::ffff:127.0.0.1)
	//
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	for _, ipNet := range c.privateIPRanges {
		if ipNet.Contains(ip) {
			return fmt.Errorf("access to private IP address %s is not allowed", ip.String())
		}
	}

	return nil
}
