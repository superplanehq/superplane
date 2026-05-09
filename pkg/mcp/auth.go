package mcp

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	defaultAPIURL         = "http://localhost:8000"
	configKeyContexts     = "contexts"
	configKeyCurrentContext = "currentContext"
)

type Config struct {
	BaseURL  string
	APIToken string
}

type configContext struct {
	URL            string  `json:"url" yaml:"url"`
	Organization   string  `json:"organization" yaml:"organization"`
	OrganizationID string  `json:"organizationId,omitempty" yaml:"organizationId,omitempty"`
	APIToken       string  `json:"apiToken" yaml:"apiToken"`
	Canvas         *string `json:"canvas,omitempty" yaml:"canvas,omitempty"`
}

// LoadConfig reads configuration from ~/.superplane.yaml or environment variables
func LoadConfig() (*Config, error) {
	home, err := homedir.Dir()
	if err != nil {
		return nil, fmt.Errorf("failed to find home directory: %w", err)
	}

	v := viper.New()
	v.AddConfigPath(home)
	v.SetConfigName(".superplane")
	v.SetEnvPrefix("SUPERPLANE")
	v.AutomaticEnv()

	// Try to read config file (it's ok if it doesn't exist)
	_ = v.ReadInConfig()

	// First check environment variables
	apiToken := os.Getenv("SUPERPLANE_API_TOKEN")
	baseURL := os.Getenv("SUPERPLANE_API_URL")

	// If not in env vars, try to get from config file
	if apiToken == "" {
		if currentContext, ok := getCurrentContext(v); ok {
			apiToken = currentContext.APIToken
			if baseURL == "" {
				baseURL = currentContext.URL
			}
		}
	}

	// Default to localhost if no URL configured
	if baseURL == "" {
		baseURL = defaultAPIURL
	}

	if apiToken == "" {
		return nil, fmt.Errorf("API token not found. Set SUPERPLANE_API_TOKEN env var or run 'superplane connect'")
	}

	return &Config{
		BaseURL:  baseURL,
		APIToken: apiToken,
	}, nil
}

// getCurrentContext reads the current context from viper config
func getCurrentContext(v *viper.Viper) (configContext, bool) {
	var contexts []configContext
	if err := v.UnmarshalKey(configKeyContexts, &contexts); err != nil {
		return configContext{}, false
	}

	if len(contexts) == 0 {
		return configContext{}, false
	}

	currentSelector := v.GetString(configKeyCurrentContext)
	if currentSelector == "" {
		return configContext{}, false
	}

	for _, context := range contexts {
		if contextSelector(context) == currentSelector {
			return context, true
		}
	}

	return configContext{}, false
}

// contextSelector generates a unique selector for a context
func contextSelector(context configContext) string {
	context = normalizeContext(context)
	id := context.OrganizationID
	if id == "" {
		id = context.Organization
	}
	return fmt.Sprintf("%s/%s", context.URL, id)
}

// normalizeContext normalizes the fields in a config context
func normalizeContext(context configContext) configContext {
	context.URL = normalizeBaseURL(context.URL)
	context.Organization = strings.TrimSpace(context.Organization)
	context.OrganizationID = strings.TrimSpace(context.OrganizationID)
	context.APIToken = strings.TrimSpace(context.APIToken)
	return context
}

// normalizeBaseURL trims whitespace and trailing slashes
func normalizeBaseURL(raw string) string {
	baseURL := strings.TrimSpace(raw)
	return strings.TrimRight(baseURL, "/")
}

// NewAPIClient creates an authenticated openapi_client
func NewAPIClient(config *Config) *openapi_client.APIClient {
	apiConfig := openapi_client.NewConfiguration()

	apiConfig.Servers = openapi_client.ServerConfigurations{
		{
			URL: config.BaseURL,
		},
	}

	if config.APIToken != "" {
		apiConfig.DefaultHeader["Authorization"] = "Bearer " + config.APIToken
	}

	apiConfig.HTTPClient = &http.Client{
		Timeout: time.Second * 30,
	}

	return openapi_client.NewAPIClient(apiConfig)
}
