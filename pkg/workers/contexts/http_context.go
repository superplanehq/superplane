package contexts

import (
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

type HTTPContext struct {
	httpClient      *http.Client
	ssrfProtection  bool
}

func NewHTTPContext(httpClient *http.Client) core.HTTPContext {
	return &HTTPContext{
		httpClient:     httpClient,
		ssrfProtection: true,
	}
}

func NewHTTPContextWithoutSSRFProtection(httpClient *http.Client) core.HTTPContext {
	return &HTTPContext{
		httpClient:     httpClient,
		ssrfProtection: false,
	}
}

func (c *HTTPContext) Do(request *http.Request) (*http.Response, error) {
	if c.ssrfProtection {
		if err := registry.ValidateURLForSSRF(request.URL.String()); err != nil {
			return nil, fmt.Errorf("SSRF protection: %w", err)
		}
	}

	return c.httpClient.Do(request)
}
