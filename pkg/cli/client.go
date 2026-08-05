package cli

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/superplanehq/superplane/pkg/buildinfo"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type ClientConfig struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

// methodSafeRedirectPolicy returns a CheckRedirect function that rejects
// redirects which would change the HTTP method (e.g., 301/302 on POST),
// as this silently converts mutating requests into GETs, dropping the body.
func methodSafeRedirectPolicy() func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if len(via) > 0 && req.Method != via[0].Method {
			return fmt.Errorf(
				"refusing to follow redirect that changes method from %s to %s (original URL: %s, redirect target: %s) — if you are using an http:// URL, try https:// instead",
				via[0].Method, req.Method, via[0].URL, req.URL,
			)
		}
		return nil
	}
}

// serverVersionTransport wraps an http.RoundTripper and remembers the
// buildinfo.VersionHeader value from the most recent response, so the CLI
// can detect version skew without a dedicated second request. Transports are
// safe to share across http.Client instances, so a single package-level
// instance backs every client the CLI builds.
type serverVersionTransport struct {
	next http.RoundTripper

	mu      sync.RWMutex
	version string
}

func (t *serverVersionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.next.RoundTrip(req)
	if err == nil {
		t.mu.Lock()
		t.version = resp.Header.Get(buildinfo.VersionHeader)
		t.mu.Unlock()
	}
	return resp, err
}

var sharedServerVersionTransport = &serverVersionTransport{next: http.DefaultTransport}

// ServerVersion returns the server version reported by the most recent API
// response, if any response has been received yet. Servers that predate
// version reporting answer without the header, which is reported as
// core.ErrServerVersionUnavailable.
func ServerVersion() (string, error) {
	sharedServerVersionTransport.mu.RLock()
	version := sharedServerVersionTransport.version
	sharedServerVersionTransport.mu.RUnlock()

	if version == "" {
		return "", core.ErrServerVersionUnavailable
	}
	return version, nil
}

// defaultHTTPClient is used by every API client the CLI builds unless a
// caller explicitly overrides ClientConfig.HTTPClient, so all of them route
// through sharedServerVersionTransport and ServerVersion stays accurate.
func defaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       time.Second * 30,
		CheckRedirect: methodSafeRedirectPolicy(),
		Transport:     sharedServerVersionTransport,
	}
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:    GetAPIURL(),
		APIToken:   GetAPIToken(),
		HTTPClient: defaultHTTPClient(),
	}
}

func NewAPIClient(config *ClientConfig) *openapi_client.APIClient {
	apiConfig := openapi_client.NewConfiguration()

	apiConfig.Servers = openapi_client.ServerConfigurations{
		{
			URL: config.BaseURL,
		},
	}

	if config.APIToken != "" {
		apiConfig.DefaultHeader["Authorization"] = "Bearer " + config.APIToken
	}

	apiConfig.HTTPClient = config.HTTPClient
	if apiConfig.HTTPClient == nil {
		apiConfig.HTTPClient = defaultHTTPClient()
	}

	return openapi_client.NewAPIClient(apiConfig)
}

func DefaultClient() *openapi_client.APIClient {
	return NewAPIClient(NewClientConfig())
}
