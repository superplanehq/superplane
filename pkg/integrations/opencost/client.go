package opencost

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type Client struct {
	BaseURL  string
	APIToken string
	http     core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type AllocationResponse struct {
	Code    int              `json:"code"`
	Status  string           `json:"status"`
	Data    []AllocationData `json:"data"`
	Message string           `json:"message,omitempty"`
}

type AllocationData map[string]AllocationItem

type AllocationItem struct {
	Name                   string           `json:"name"`
	Properties             map[string]any   `json:"properties,omitempty"`
	Window                 AllocationWindow `json:"window"`
	Start                  string           `json:"start"`
	End                    string           `json:"end"`
	CPUCoreHours           float64          `json:"cpuCoreHours"`
	CPUCoreRequestAverage  float64          `json:"cpuCoreRequestAverage"`
	CPUCoreUsageAverage    float64          `json:"cpuCoreUsageAverage"`
	CPUCost                float64          `json:"cpuCost"`
	CPUCostAdjustment      float64          `json:"cpuCostAdjustment"`
	CPUEfficiency          float64          `json:"cpuEfficiency"`
	GPUCost                float64          `json:"gpuCost"`
	GPUCostAdjustment      float64          `json:"gpuCostAdjustment"`
	NetworkCost            float64          `json:"networkCost"`
	NetworkCostAdjustment  float64          `json:"networkCostAdjustment"`
	LoadBalancerCost       float64          `json:"loadBalancerCost"`
	PVCost                 float64          `json:"pvCost"`
	PVCostAdjustment       float64          `json:"pvCostAdjustment"`
	RAMByteHours           float64          `json:"ramByteHours"`
	RAMBytesRequestAverage float64          `json:"ramBytesRequestAverage"`
	RAMBytesUsageAverage   float64          `json:"ramBytesUsageAverage"`
	RAMCost                float64          `json:"ramCost"`
	RAMCostAdjustment      float64          `json:"ramCostAdjustment"`
	RAMEfficiency          float64          `json:"ramEfficiency"`
	SharedCost             float64          `json:"sharedCost"`
	ExternalCost           float64          `json:"externalCost"`
	TotalCost              float64          `json:"totalCost"`
	TotalEfficiency        float64          `json:"totalEfficiency"`
}

type AllocationWindow struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiURLBytes, err := ctx.GetConfig("apiURL")
	if err != nil {
		return nil, err
	}

	apiURL := strings.TrimSpace(string(apiURLBytes))
	if apiURL == "" {
		return nil, fmt.Errorf("apiURL is required")
	}

	apiURL = strings.TrimRight(apiURL, "/")

	var apiToken string
	apiTokenBytes, err := ctx.GetConfig("apiToken")
	if err == nil {
		apiToken = strings.TrimSpace(string(apiTokenBytes))
	}

	return &Client{
		BaseURL:  apiURL,
		APIToken: apiToken,
		http:     httpClient,
	}, nil
}

func (c *Client) Verify() error {
	query := url.Values{}
	query.Set("window", "1h")
	query.Set("aggregate", "cluster")

	_, err := c.GetAllocation(query)
	return err
}

func (c *Client) GetAllocation(query url.Values) (*AllocationResponse, error) {
	body, err := c.execRequest(http.MethodGet, "/allocation", query)
	if err != nil {
		return nil, err
	}

	response := &AllocationResponse{}
	if err := json.Unmarshal(body, response); err != nil {
		return nil, fmt.Errorf("failed to parse allocation response: %w", err)
	}

	return response, nil
}

func (c *Client) execRequest(method, path string, query url.Values) ([]byte, error) {
	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, &APIError{StatusCode: res.StatusCode, Body: string(responseBody)}
	}

	return responseBody, nil
}
