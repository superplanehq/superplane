package cloudfunctions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	cloudFunctionsBaseURL = "https://cloudfunctions.googleapis.com"
	cloudRunBaseURL       = "https://run.googleapis.com"
)

// Client is the interface used by Cloud Functions components to call the API.
type Client interface {
	GetURL(ctx context.Context, fullURL string) ([]byte, error)
	PostURL(ctx context.Context, fullURL string, body any) ([]byte, error)
	ProjectID() string
}

var (
	clientFactoryMu sync.RWMutex
	clientFactory   func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)
)

func SetClientFactory(fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error)) {
	clientFactoryMu.Lock()
	defer clientFactoryMu.Unlock()
	clientFactory = fn
}

func getClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error) {
	clientFactoryMu.RLock()
	fn := clientFactory
	clientFactoryMu.RUnlock()
	if fn == nil {
		panic("gcp cloudfunctions: SetClientFactory was not called by the gcp integration")
	}
	return fn(httpCtx, integration)
}

// functionCallURL returns the Cloud Functions v1 API URL for calling a function.
// It works with both v1 and v2 resource names, which share the same format:
// projects/{project}/locations/{location}/functions/{function}
func functionCallURL(resourceName string) string {
	return fmt.Sprintf("%s/v1/%s:call", cloudFunctionsBaseURL, resourceName)
}

// functionGetURL returns the Cloud Functions v2 API URL for getting function details.
func functionGetURL(resourceName string) string {
	return fmt.Sprintf("%s/v2/%s", cloudFunctionsBaseURL, resourceName)
}

type FunctionDetails struct {
	Environment string
	URI         string
}

// GetFunctionDetails fetches function details from Cloud Functions v2 or Cloud Run,
// depending on the resource name format (functions/* vs services/*).
func GetFunctionDetails(ctx context.Context, client Client, resourceName string) (*FunctionDetails, error) {
	if strings.Contains(resourceName, "/services/") {
		return getCloudRunServiceDetails(ctx, client, resourceName)
	}
	return getCloudFunctionDetails(ctx, client, resourceName)
}

func getCloudFunctionDetails(ctx context.Context, client Client, resourceName string) (*FunctionDetails, error) {
	data, err := client.GetURL(ctx, functionGetURL(resourceName))
	if err != nil {
		return nil, fmt.Errorf("get function details: %w", err)
	}

	var resp struct {
		Environment   string `json:"environment"`
		ServiceConfig struct {
			URI string `json:"uri"`
		} `json:"serviceConfig"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse function details: %w", err)
	}

	return &FunctionDetails{
		Environment: resp.Environment,
		URI:         resp.ServiceConfig.URI,
	}, nil
}

func getCloudRunServiceDetails(ctx context.Context, client Client, resourceName string) (*FunctionDetails, error) {
	serviceURL := fmt.Sprintf("%s/v2/%s", cloudRunBaseURL, resourceName)
	data, err := client.GetURL(ctx, serviceURL)
	if err != nil {
		return nil, fmt.Errorf("get Cloud Run service details: %w", err)
	}

	var resp struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse Cloud Run service details: %w", err)
	}

	return &FunctionDetails{
		Environment: "GEN_2",
		URI:         resp.URI,
	}, nil
}
