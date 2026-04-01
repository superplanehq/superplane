package cli

import (
	"fmt"
	"net/http"
	"time"

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

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:  GetAPIURL(),
		APIToken: GetAPIToken(),
		HTTPClient: &http.Client{
			Timeout:       time.Second * 30,
			CheckRedirect: methodSafeRedirectPolicy(),
		},
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

	if config.HTTPClient != nil {
		apiConfig.HTTPClient = config.HTTPClient
	}

	return openapi_client.NewAPIClient(apiConfig)
}

func DefaultClient() *openapi_client.APIClient {
	return NewAPIClient(NewClientConfig())
}
