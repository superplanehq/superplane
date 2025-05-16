package cli

import (
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// ClientConfig contains configuration for the API client
type ClientConfig struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:   GetAPIURL(),
		AuthToken: GetAuthToken(),
		HTTPClient: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

// NewAPIClient creates a new OpenAPI client with the given configuration
func NewAPIClient(config *ClientConfig) *openapi_client.APIClient {
	apiConfig := openapi_client.NewConfiguration()

	apiConfig.Servers = openapi_client.ServerConfigurations{
		{
			URL: config.BaseURL,
		},
	}

	if config.AuthToken != "" {
		apiConfig.DefaultHeader["Authorization"] = "Bearer " + config.AuthToken
	}

	if config.HTTPClient != nil {
		apiConfig.HTTPClient = config.HTTPClient
	}

	return openapi_client.NewAPIClient(apiConfig)
}

func DefaultClient() *openapi_client.APIClient {
	return NewAPIClient(NewClientConfig())
}
