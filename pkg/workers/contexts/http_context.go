package contexts

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

type HTTPContext struct {
	httpClient *http.Client
}

func NewHTTPContext(httpClient *http.Client) core.HTTPContext {
	return &HTTPContext{
		httpClient: httpClient,
	}
}

func (c *HTTPContext) Do(request *http.Request) (*http.Response, error) {
	return c.httpClient.Do(request)
}
