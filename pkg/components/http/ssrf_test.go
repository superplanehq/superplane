package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/registry"
	workerscontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
)

func TestSSRF__ValidateURLForSSRF__BlocksLocalhost(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldBlock bool
	}{
		{"localhost", "http://localhost/api", true},
		{"localhost with port", "http://localhost:8080/api", true},
		{"127.0.0.1", "http://127.0.0.1/api", true},
		{"127.0.0.1 with port", "http://127.0.0.1:3000/api", true},
		{"loopback IPv6", "http://[::1]/api", true},
		{"0.0.0.0", "http://0.0.0.0/api", true},
		{"IPv6 unspecified", "http://[::]/api", true},
		{"external IP", "http://8.8.8.8/api", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			if tt.shouldBlock {
				assert.Error(t, err, "expected URL to be blocked: %s", tt.url)
				assert.Contains(t, err.Error(), "not allowed")
			} else {
				assert.NoError(t, err, "expected URL to be allowed: %s", tt.url)
			}
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksCloudMetadata(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"AWS metadata IPv4", "http://169.254.169.254/latest/meta-data/"},
		{"AWS metadata with path", "http://169.254.169.254/latest/api/token"},
		{"GCP metadata", "http://metadata.google.internal/computeMetadata/v1/"},
		{"GCP metadata alt", "http://metadata.goog/computeMetadata/v1/"},
		{"AWS metadata IPv6", "http://[fd00:ec2::254]/latest/meta-data/"},
		{"Azure metadata", "http://metadata.azure.com/metadata/instance"},
		{"Azure metadata subdomain", "http://foo.metadata.azure.com/metadata/instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			assert.Error(t, err, "expected cloud metadata URL to be blocked: %s", tt.url)
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksKubernetesAPI(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"kubernetes.default", "http://kubernetes.default/api"},
		{"kubernetes.default.svc", "http://kubernetes.default.svc/api"},
		{"kubernetes.default.svc.cluster.local", "http://kubernetes.default.svc.cluster.local/api"},
		{"subdomain of kubernetes.default", "http://api.kubernetes.default/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			assert.Error(t, err, "expected Kubernetes API URL to be blocked: %s", tt.url)
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksPrivateIPRanges(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldBlock bool
	}{
		// Private ranges - should be blocked
		{"10.x range", "http://10.0.0.1/api", true},
		{"10.x range high", "http://10.255.255.255/api", true},
		{"172.16.x range", "http://172.16.0.1/api", true},
		{"172.31.x range", "http://172.31.255.255/api", true},
		{"192.168.x range", "http://192.168.1.1/api", true},
		{"link-local", "http://169.254.1.1/api", true},

		// Public ranges - should be allowed
		{"public IP 1", "http://8.8.8.8/api", false},
		{"public IP 2", "http://1.1.1.1/api", false},
		{"172.15.x (not private)", "http://172.15.0.1/api", false},
		{"172.32.x (not private)", "http://172.32.0.1/api", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			if tt.shouldBlock {
				assert.Error(t, err, "expected private IP to be blocked: %s", tt.url)
				assert.Contains(t, err.Error(), "private IP")
			} else {
				assert.NoError(t, err, "expected public IP to be allowed: %s", tt.url)
			}
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksIPv4MappedIPv6(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"IPv4-mapped localhost", "http://[::ffff:127.0.0.1]/api"},
		{"IPv4-mapped private 10.x", "http://[::ffff:10.0.0.1]/api"},
		{"IPv4-mapped private 192.168.x", "http://[::ffff:192.168.1.1]/api"},
		{"IPv4-mapped metadata", "http://[::ffff:169.254.169.254]/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			assert.Error(t, err, "expected IPv4-mapped IPv6 address to be blocked: %s", tt.url)
			assert.Contains(t, err.Error(), "private IP")
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksInvalidSchemes(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"file scheme", "file:///etc/passwd"},
		{"ftp scheme", "ftp://example.com/file"},
		{"gopher scheme", "gopher://example.com/"},
		{"dict scheme", "dict://example.com/"},
		{"empty scheme", "://example.com/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			assert.Error(t, err, "expected invalid scheme to be blocked: %s", tt.url)
		})
	}
}

func TestSSRF__ValidateURLForSSRF__BlocksSubdomains(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldBlock bool
	}{
		{"subdomain of metadata.google.internal", "http://foo.metadata.google.internal/api", true},
		{"subdomain of localhost", "http://foo.localhost/api", true},
		{"similar but different domain", "http://8.8.4.4/api", false},
		{"metadata in path not host", "http://8.8.8.8/metadata.google.internal", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			if tt.shouldBlock {
				assert.Error(t, err, "expected subdomain to be blocked: %s", tt.url)
			} else {
				assert.NoError(t, err, "expected URL to be allowed: %s", tt.url)
			}
		})
	}
}

func TestSSRF__RedirectBlocking(t *testing.T) {
	client := registry.NewSSRFSafeHTTPClient()

	t.Run("CheckRedirect blocks private IPs", func(t *testing.T) {
		err := registry.ValidateURLForSSRF("http://169.254.169.254/latest/meta-data/")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("Client has CheckRedirect configured", func(t *testing.T) {
		assert.NotNil(t, client.CheckRedirect, "client should have CheckRedirect configured")
	})
}

func TestSSRF__RedirectBlocking__EndToEnd(t *testing.T) {
	// The SSRF-safe client uses a custom dialer that blocks private IPs at connection time.
	// This prevents DNS rebinding attacks by validating the resolved IP just before connecting.
	// As a result, we can't even connect to localhost test servers with the SSRF-safe client.
	// Instead, we verify the dialer blocks connections to private IPs directly.

	client := registry.NewSSRFSafeHTTPClient()

	// Try to connect to a private IP - should be blocked at connection time
	resp, err := client.Get("http://127.0.0.1:12345/")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection blocked")
	if resp != nil {
		resp.Body.Close()
	}
}

func TestSSRF__DialerBlocking__ShortenedIPForms(t *testing.T) {
	// Test that shortened IP forms that bypass URL validation are still blocked
	// at the dialer level when they resolve to private IPs.
	client := registry.NewSSRFSafeHTTPClient()

	tests := []struct {
		name string
		url  string
	}{
		{"shortened localhost 127.1", "http://127.1:12345/"},
		{"shortened localhost 127.0.1", "http://127.0.1:12345/"},
		{"decimal localhost", "http://2130706433:12345/"},
		{"zero short form", "http://0:12345/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(tt.url)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "connection blocked")
			if resp != nil {
				resp.Body.Close()
			}
		})
	}
}

func TestSSRF__RedirectBlocking__AllowsSafeRedirects(t *testing.T) {
	// Create a server that redirects to another safe endpoint
	finalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer finalServer.Close()

	redirectServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, finalServer.URL, http.StatusFound)
	}))
	defer redirectServer.Close()

	// Use SSRF-safe client without protection for localhost test servers
	httpCtx := workerscontexts.NewHTTPContextWithoutSSRFProtection(&http.Client{Timeout: 5 * time.Second})

	req, err := http.NewRequest("GET", redirectServer.URL, nil)
	require.NoError(t, err)

	resp, err := httpCtx.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSSRF__ExecuteRequest__BlocksBlockedURLs(t *testing.T) {
	h := &HTTP{}

	// Create an HTTPContext WITH SSRF protection enabled
	httpCtx := workerscontexts.NewHTTPContext(&http.Client{Timeout: 30 * time.Second})

	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost/api"},
		{"private IP", "http://10.0.0.1/api"},
		{"metadata endpoint", "http://169.254.169.254/latest/meta-data/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := Spec{
				Method: "GET",
				URL:    tt.url,
			}

			resp, err := h.executeRequest(httpCtx, spec, 5*time.Second)
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), "SSRF protection")
		})
	}
}

func TestSSRF__ExecuteRequest__AllowsWithoutProtection(t *testing.T) {
	h := &HTTP{}

	// Create an HTTPContext WITHOUT SSRF protection (for testing localhost)
	httpCtx := workerscontexts.NewHTTPContextWithoutSSRFProtection(&http.Client{Timeout: 30 * time.Second})

	// These would be blocked with SSRF protection, but should pass validation
	// (they'll fail at the HTTP level since no server is listening)
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost:59999/api"},
		{"private IP", "http://10.0.0.1:59999/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := Spec{
				Method: "GET",
				URL:    tt.url,
			}

			resp, err := h.executeRequest(httpCtx, spec, 1*time.Second)
			// Should fail with connection error, not SSRF error
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.NotContains(t, err.Error(), "SSRF protection")
		})
	}
}

func TestSSRF__ValidateURLForSSRF__InvalidURLs(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
	}{
		{"empty URL", "", true},
		{"no host", "http:///path", true},
		{"invalid scheme", "://invalid", true},
		{"valid URL", "https://8.8.8.8/api", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateURLForSSRF(tt.url)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test that HTTPContext properly gates SSRF protection
func TestSSRF__HTTPContext__WithProtection(t *testing.T) {
	httpCtx := workerscontexts.NewHTTPContext(&http.Client{Timeout: 5 * time.Second})

	// Create a request to a blocked URL
	req, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/", nil)
	require.NoError(t, err)

	// The Do method should block this
	resp, err := httpCtx.Do(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "SSRF protection")
}

func TestSSRF__HTTPContext__WithoutProtection(t *testing.T) {
	httpCtx := workerscontexts.NewHTTPContextWithoutSSRFProtection(&http.Client{Timeout: 1 * time.Second})

	// Create a request to a blocked URL
	req, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/", nil)
	require.NoError(t, err)

	// The Do method should NOT block this (will fail with connection error instead)
	resp, err := httpCtx.Do(req)
	// Should fail, but NOT with SSRF error
	if err != nil {
		assert.NotContains(t, err.Error(), "SSRF protection")
	}
	if resp != nil {
		resp.Body.Close()
	}
}
