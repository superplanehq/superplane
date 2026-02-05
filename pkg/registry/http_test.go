package registry

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__NewHTTPContext_InvalidCIDR(t *testing.T) {

	t.Run("invalid CIDR", func(t *testing.T) {
		ctx, err := NewHTTPContext(HTTPOptions{
			PrivateIPRanges: []string{"not-a-cidr"},
		})

		require.Error(t, err)
		require.Nil(t, ctx)
	})

	t.Run("no private IP ranges or blocked hosts", func(t *testing.T) {
		ctx, err := NewHTTPContext(HTTPOptions{})
		require.NoError(t, err)
		require.NotNil(t, ctx)
	})
}

func Test__HTTPContext__ValidateURL__DefaultConfiguration(t *testing.T) {
	ctx, err := NewHTTPContext(defaultHTTPOptions())
	require.NoError(t, err)

	tests := []struct {
		name    string
		rawURL  string
		wantErr string
	}{
		{
			name:    "scheme not allowed",
			rawURL:  "ftp://example.com/file",
			wantErr: "only http and https schemes are allowed",
		},
		{
			name:    "missing host",
			rawURL:  "http:///path",
			wantErr: "URL must have a host",
		},
		{
			name:    "blocked host subdomain",
			rawURL:  "http://api.metadata.google.internal",
			wantErr: "access to api.metadata.google.internal is not allowed",
		},
		{
			name:    "external IP",
			rawURL:  "http://8.8.8.8",
			wantErr: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parsed, err := url.Parse(test.rawURL)
			require.NoError(t, err)

			err = ctx.validateURL(parsed)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}

	t.Run("blocked hosts", func(t *testing.T) {
		for _, host := range defaultHTTPOptions().BlockedHosts {
			t.Run(host, func(t *testing.T) {
				rawURL := "http://" + host
				if strings.Contains(host, ":") {
					rawURL = "http://[" + host + "]"
				}

				parsed, err := url.Parse(rawURL)
				require.NoError(t, err)

				err = ctx.validateURL(parsed)
				require.Error(t, err)
				assert.Contains(t, err.Error(), "access to "+host+" is not allowed")
			})
		}
	})
}

func Test__HTTPContext__ValidateURL_AllowsNonBlockedHost(t *testing.T) {
	ctx := &HTTPContext{
		blockedHosts:    []string{"example.com"},
		privateIPRanges: nil,
	}

	parsed, err := url.Parse("https://example.org")
	require.NoError(t, err)
	require.NoError(t, ctx.validateURL(parsed))
}

func Test__HTTPContext__ValidateURL__BlockedHostSubdomains(t *testing.T) {
	ctx, err := NewHTTPContext(defaultHTTPOptions())
	require.NoError(t, err)

	for _, host := range defaultHTTPOptions().BlockedHosts {
		if strings.Contains(host, ":") || net.ParseIP(host) != nil {
			continue
		}

		subdomain := "sub." + host
		t.Run(subdomain, func(t *testing.T) {
			parsed, err := url.Parse("http://" + subdomain)
			require.NoError(t, err)

			err = ctx.validateURL(parsed)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "access to "+subdomain+" is not allowed")
		})
	}
}

func Test__HTTPContext__ValidateIP_IPv4MappedIPv6(t *testing.T) {
	ctx, err := NewHTTPContext(HTTPOptions{
		PrivateIPRanges: []string{"127.0.0.0/8"},
	})

	require.NoError(t, err)

	ip := net.ParseIP("::ffff:127.0.0.1")
	require.NotNil(t, ip)

	err = ctx.validateIP(ip)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "access to private IP address 127.0.0.1 is not allowed")
}

func Test__HTTPContext__Do(t *testing.T) {
	ctx, err := NewHTTPContext(HTTPOptions{
		BlockedHosts:    []string{"example.com"},
		PrivateIPRanges: []string{"127.0.0.0/8"},
	})

	require.NoError(t, err)

	t.Run("blocked host", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)

		_, err = ctx.Do(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access to example.com is not allowed")
	})

	t.Run("private IP", func(t *testing.T) {
		var hits atomic.Int32

		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.WriteHeader(http.StatusOK)
		}))

		t.Cleanup(testServer.Close)

		ctx, err := NewHTTPContext(defaultHTTPOptions())
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		_, err = ctx.Do(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access to 127.0.0.1 is not allowed")
		assert.Zero(t, hits.Load())
	})
}

func Test__HTTPContext__Do__RedirectLimit(t *testing.T) {
	var hits atomic.Int32

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		step := strings.TrimPrefix(r.URL.Path, "/r/")
		if step == "" || step == r.URL.Path {
			http.Redirect(w, r, "/r/1", http.StatusFound)
			return
		}

		index, err := strconv.Atoi(step)
		if err != nil {
			http.Error(w, "invalid redirect index", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/r/"+strconv.Itoa(index+1), http.StatusFound)
	}))

	t.Cleanup(testServer.Close)

	ctx, err := NewHTTPContext(HTTPOptions{BlockedHosts: []string{"example.com"}})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/r/0", nil)
	require.NoError(t, err)

	_, err = ctx.Do(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after 10 redirects")
	assert.GreaterOrEqual(t, hits.Load(), int32(10))
}

func Test__HTTPContext__Do__RedirectToBlockedHost(t *testing.T) {
	var hits atomic.Int32

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, "http://example.com/blocked", http.StatusFound)
	}))
	t.Cleanup(testServer.Close)

	ctx, err := NewHTTPContext(HTTPOptions{
		BlockedHosts: []string{"example.com"},
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
	require.NoError(t, err)

	_, err = ctx.Do(req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect blocked")
	assert.Contains(t, err.Error(), "access to example.com is not allowed")
	assert.Equal(t, int32(1), hits.Load())
}

func Test__HTTPContext__ValidateIP__DefaultConfiguration(t *testing.T) {
	ctx, err := NewHTTPContext(defaultHTTPOptions())
	require.NoError(t, err)

	tests := []struct {
		name   string
		ipAddr string
	}{
		{
			name:   "0.0.0.0/8",
			ipAddr: "0.1.2.3",
		},
		{
			name:   "10.0.0.0/8",
			ipAddr: "10.1.2.3",
		},
		{
			name:   "172.16.0.0/12",
			ipAddr: "172.16.5.4",
		},
		{
			name:   "192.168.0.0/16",
			ipAddr: "192.168.1.2",
		},
		{
			name:   "127.0.0.0/8",
			ipAddr: "127.0.0.2",
		},
		{
			name:   "169.254.0.0/16",
			ipAddr: "169.254.1.1",
		},
		{
			name:   "::1/128",
			ipAddr: "::1",
		},
		{
			name:   "fc00::/7",
			ipAddr: "fc00::1",
		},
		{
			name:   "fe80::/10",
			ipAddr: "fe80::1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ip := net.ParseIP(test.ipAddr)
			require.NotNil(t, ip)

			err := ctx.validateIP(ip)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "access to private IP address "+test.ipAddr+" is not allowed")
		})
	}
}

func defaultHTTPOptions() HTTPOptions {
	return HTTPOptions{
		BlockedHosts: []string{
			"metadata.google.internal",
			"metadata.goog",
			"metadata.azure.com",
			"169.254.169.254",
			"fd00:ec2::254",
			"kubernetes.default",
			"kubernetes.default.svc",
			"kubernetes.default.svc.cluster.local",
			"localhost",
			"127.0.0.1",
			"::1",
			"0.0.0.0",
			"::",
		},
		PrivateIPRanges: []string{
			"0.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"127.0.0.0/8",
			"169.254.0.0/16",
			"::1/128",
			"fc00::/7",
			"fe80::/10",
		},
	}
}
