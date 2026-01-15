package dash0

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
	Token   string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(http core.HTTPContext, ctx core.AppInstallationContext) (*Client, error) {
	apiToken, err := ctx.GetConfig("apiToken")
	if err != nil {
		return nil, fmt.Errorf("error getting api token: %v", err)
	}

	baseURL := ""
	baseURLConfig, err := ctx.GetConfig("baseURL")
	if err == nil && baseURLConfig != nil && len(baseURLConfig) > 0 {
		baseURL = strings.TrimSuffix(string(baseURLConfig), "/")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("baseURL is required for Dash0 Cloud. Find your API URL in Dash0 dashboard under Organization Settings > Endpoints Reference")
	}

	// Strip /api/prometheus if user included it in the base URL
	baseURL = strings.TrimSuffix(baseURL, "/api/prometheus")

	return &Client{
		Token:   string(apiToken),
		BaseURL: baseURL,
		http:    http,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader, contentType string) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error building request: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing request: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %v", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type PrometheusResponse struct {
	Status string                 `json:"status"`
	Data   PrometheusResponseData `json:"data"`
}

type PrometheusResponseData struct {
	ResultType string                  `json:"resultType"`
	Result     []PrometheusQueryResult `json:"result"`
}

type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`  // For instant queries: [timestamp, value]
	Values [][]interface{}   `json:"values,omitempty"` // For range queries: [[timestamp, value], ...]
}

func (c *Client) ExecutePrometheusInstantQuery(promQLQuery, dataset string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/prometheus/api/v1/query", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)

	responseBody, err := c.execRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	var response PrometheusResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}

func (c *Client) ExecutePrometheusRangeQuery(promQLQuery, dataset, start, end, step string) (map[string]any, error) {
	apiURL := fmt.Sprintf("%s/api/prometheus/api/v1/query_range", c.BaseURL)

	data := url.Values{}
	data.Set("dataset", dataset)
	data.Set("query", promQLQuery)
	data.Set("start", start)
	data.Set("end", end)
	data.Set("step", step)

	responseBody, err := c.execRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}

	var response PrometheusResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", response.Status)
	}

	return map[string]any{
		"status": response.Status,
		"data":   response.Data,
	}, nil
}
