package cli

import (
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type ClientConfig struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

func NewClientConfig() *ClientConfig {
	return &ClientConfig{
		BaseURL:  GetAPIURL(),
		APIToken: GetAPIToken(),
		HTTPClient: &http.Client{
			Timeout: time.Second * 30,
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
