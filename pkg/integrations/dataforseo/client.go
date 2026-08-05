package dataforseo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.dataforseo.com/v3"

type Client struct {
	apiKey string
	http   core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	return &Client{
		apiKey: string(apiKey),
		http:   httpClient,
	}, nil
}

func (c *Client) execRequest(method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+c.apiKey)

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer res.Body.Close()

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("request got %d code: %s", res.StatusCode, string(responseBody))
	}

	return responseBody, nil
}

type userDataResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		ID string `json:"id"`
	} `json:"tasks"`
}

func (c *Client) Verify() error {
	body, err := c.execRequest(http.MethodGet, baseURL+"/appendix/user_data", nil)
	if err != nil {
		return err
	}

	var response userDataResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to unmarshal user_data response: %v", err)
	}

	if response.StatusCode != 20000 {
		return fmt.Errorf("DataForSEO returned status_code %d", response.StatusCode)
	}

	return nil
}

type taskPostResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		ID         string `json:"id"`
		StatusCode int    `json:"status_code"`
	} `json:"tasks"`
}

func (c *Client) PostAudit(target string, maxCrawlPages int) (string, error) {
	payload := []map[string]any{
		{"target": target, "max_crawl_pages": maxCrawlPages},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task_post request: %v", err)
	}

	respBody, err := c.execRequest(http.MethodPost, baseURL+"/on_page/task_post", bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	var response taskPostResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal task_post response: %v", err)
	}

	if len(response.Tasks) == 0 || response.Tasks[0].ID == "" {
		return "", fmt.Errorf("task_post response missing task id")
	}

	return response.Tasks[0].ID, nil
}
