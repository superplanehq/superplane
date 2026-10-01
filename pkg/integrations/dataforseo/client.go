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
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		ID            string `json:"id"`
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
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

	if response.StatusCode != 20000 {
		return "", fmt.Errorf("DataForSEO returned status_code %d: %s", response.StatusCode, response.StatusMessage)
	}

	if len(response.Tasks) == 0 {
		return "", fmt.Errorf("task_post response missing task id")
	}

	task := response.Tasks[0]
	if task.StatusCode != 20000 && task.StatusCode != 20100 {
		return "", fmt.Errorf("DataForSEO rejected task_post request: status_code %d: %s", task.StatusCode, task.StatusMessage)
	}

	if task.ID == "" {
		return "", fmt.Errorf("task_post response missing task id")
	}

	return task.ID, nil
}

type summaryResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		Result []struct {
			CrawlProgress string `json:"crawl_progress"`
		} `json:"result"`
	} `json:"tasks"`
}

func (c *Client) GetSummary(taskID string) (string, error) {
	respBody, err := c.execRequest(http.MethodGet, baseURL+"/on_page/summary/"+taskID, nil)
	if err != nil {
		return "", err
	}

	var response summaryResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal summary response: %v", err)
	}

	if len(response.Tasks) == 0 || len(response.Tasks[0].Result) == 0 {
		return "", fmt.Errorf("summary response missing result for task %s", taskID)
	}

	return response.Tasks[0].Result[0].CrawlProgress, nil
}

type PageChecks struct {
	BrokenLinks          bool `json:"broken_links"`
	DuplicateTitle       bool `json:"duplicate_title"`
	DuplicateDescription bool `json:"duplicate_description"`
	IsBroken             bool `json:"is_broken"`
}

func (c PageChecks) HasIssue() bool {
	return c.BrokenLinks || c.DuplicateTitle || c.DuplicateDescription || c.IsBroken
}

type PageResult struct {
	URL    string     `json:"url"`
	Checks PageChecks `json:"checks"`
}

type pagesResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		Result []struct {
			Items []PageResult `json:"items"`
		} `json:"result"`
	} `json:"tasks"`
}

func (c *Client) GetPages(taskID string, limit int) ([]PageResult, error) {
	payload := []map[string]any{
		{"id": taskID, "limit": limit},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal pages request: %v", err)
	}

	respBody, err := c.execRequest(http.MethodPost, baseURL+"/on_page/pages", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response pagesResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pages response: %v", err)
	}

	if len(response.Tasks) == 0 || len(response.Tasks[0].Result) == 0 {
		return nil, fmt.Errorf("pages response missing result for task %s", taskID)
	}

	return response.Tasks[0].Result[0].Items, nil
}
