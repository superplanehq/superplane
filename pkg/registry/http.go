package registry

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"
)

type HTTPPolicy struct {
	BlockedHosts    []string
	PrivateIPRanges []string
}

type HTTPContext struct {
	client           *http.Client
	dialer           *net.Dialer
	maxResponseBytes int64
	policyResolver   func() (HTTPPolicy, error)
	policyCacheTTL   time.Duration
	policyMu         sync.RWMutex
	policy           compiledHTTPPolicy
	policyExpiresAt  time.Time
}

type HTTPOptions struct {
	BlockedHosts     []string
	PrivateIPRanges  []string
	MaxResponseBytes int64
	PolicyResolver   func() (HTTPPolicy, error)
	PolicyCacheTTL   time.Duration
}

type compiledHTTPPolicy struct {
	blockedHosts    []string
	privateIPRanges []*net.IPNet
}

func NewHTTPContext(options HTTPOptions) (*HTTPContext, error) {
	httpCtx := &HTTPContext{
		maxResponseBytes: options.MaxResponseBytes,
		policyResolver:   options.PolicyResolver,
		policyCacheTTL:   options.PolicyCacheTTL,
	}

	if httpCtx.policyResolver == nil {
		compiledPolicy, err := compileHTTPPolicy(HTTPPolicy{
			BlockedHosts:    options.BlockedHosts,
			PrivateIPRanges: options.PrivateIPRanges,
		})
		if err != nil {
			return nil, err
		}
		httpCtx.policy = compiledPolicy
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

			policy, err := httpCtx.activePolicy()
			if err != nil {
				return err
			}

			if err := httpCtx.validateIPWithPolicy(policy, ip); err != nil {
				return fmt.Errorf("connection blocked: %w", err)
			}

			return nil
		},
	}

	httpCtx.client = &http.Client{
		Timeout: 30 * time.Second,
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

			policy, err := httpCtx.activePolicy()
			if err != nil {
				return err
			}

			if err := httpCtx.validateURLWithPolicy(policy, req.URL); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}

			return nil
		},
	}

	if _, err := httpCtx.activePolicy(); err != nil {
		return nil, err
	}

	return httpCtx, nil
}

func (c *HTTPContext) Do(request *http.Request) (*http.Response, error) {
	policy, err := c.activePolicy()
	if err != nil {
		return nil, err
	}

	if len(policy.privateIPRanges) == 0 && len(policy.blockedHosts) == 0 {
		return c.do(request)
	}

	if err := c.validateURLWithPolicy(policy, request.URL); err != nil {
		return nil, err
	}

	return c.do(request)
}

func (c *HTTPContext) InvalidatePolicyCache() {
	c.policyMu.Lock()
	defer c.policyMu.Unlock()

	c.policyExpiresAt = time.Time{}
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
	policy, err := c.activePolicy()
	if err != nil {
		return err
	}

	return c.validateURLWithPolicy(policy, URL)
}

func (c *HTTPContext) validateURLWithPolicy(policy compiledHTTPPolicy, URL *url.URL) error {
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
	for _, blocked := range policy.blockedHosts {
		if hostLower == blocked || strings.HasSuffix(hostLower, "."+blocked) {
			return fmt.Errorf("access to %s is not allowed", host)
		}
	}

	//
	// If host is an IP address, validate it immediately
	// For hostnames, IP validation happens at connection time via the dialer.
	//
	if ip := net.ParseIP(host); ip != nil {
		if err := c.validateIPWithPolicy(policy, ip); err != nil {
			return err
		}
	}

	return nil
}

func (c *HTTPContext) validateIP(ip net.IP) error {
	policy, err := c.activePolicy()
	if err != nil {
		return err
	}

	return c.validateIPWithPolicy(policy, ip)
}

func (c *HTTPContext) validateIPWithPolicy(policy compiledHTTPPolicy, ip net.IP) error {
	//
	// Handle IPv4-mapped IPv6 addresses (e.g., ::ffff:127.0.0.1)
	//
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}

	for _, ipNet := range policy.privateIPRanges {
		if ipNet.Contains(ip) {
			return fmt.Errorf("access to private IP address %s is not allowed", ip.String())
		}
	}

	return nil
}

func (c *HTTPContext) activePolicy() (compiledHTTPPolicy, error) {
	if c.policyResolver == nil {
		return c.policy, nil
	}

	now := time.Now()

	c.policyMu.RLock()
	if !c.policyExpiresAt.IsZero() && now.Before(c.policyExpiresAt) {
		policy := c.policy
		c.policyMu.RUnlock()
		return policy, nil
	}
	c.policyMu.RUnlock()

	c.policyMu.Lock()
	defer c.policyMu.Unlock()

	now = time.Now()
	if !c.policyExpiresAt.IsZero() && now.Before(c.policyExpiresAt) {
		return c.policy, nil
	}

	policy, err := c.policyResolver()
	if err != nil {
		return compiledHTTPPolicy{}, err
	}

	compiledPolicy, err := compileHTTPPolicy(policy)
	if err != nil {
		return compiledHTTPPolicy{}, err
	}

	c.policy = compiledPolicy
	if c.policyCacheTTL <= 0 {
		c.policyExpiresAt = time.Time{}
	} else {
		c.policyExpiresAt = now.Add(c.policyCacheTTL)
	}

	return c.policy, nil
}

func compileHTTPPolicy(policy HTTPPolicy) (compiledHTTPPolicy, error) {
	compiledPolicy := compiledHTTPPolicy{
		blockedHosts:    make([]string, 0, len(policy.BlockedHosts)),
		privateIPRanges: make([]*net.IPNet, 0, len(policy.PrivateIPRanges)),
	}

	for _, host := range policy.BlockedHosts {
		compiledPolicy.blockedHosts = append(compiledPolicy.blockedHosts, strings.ToLower(host))
	}

	for _, cidr := range policy.PrivateIPRanges {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return compiledHTTPPolicy{}, fmt.Errorf("invalid private IP range: %w", err)
		}

		compiledPolicy.privateIPRanges = append(compiledPolicy.privateIPRanges, ipNet)
	}

	return compiledPolicy, nil
}
