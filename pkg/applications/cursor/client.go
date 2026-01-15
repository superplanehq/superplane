package cursor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const defaultBaseURL = "https://api.cursor.com/v0"

type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.AppInstallationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no app installation context")
	}

	apiKey, err := ctx.GetConfig("apiKey")
	if err != nil {
		return nil, err
	}

	return &Client{
		APIKey:  string(apiKey),
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

type ModelsResponse struct {
	Models []string `json:"models"`
}

type LaunchAgentPrompt struct {
	Text string `json:"text"`
}

type LaunchAgentSource struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
}

type LaunchAgentTarget struct {
	AutoCreatePr          bool   `json:"autoCreatePr,omitempty"`
	OpenAsCursorGithubApp bool   `json:"openAsCursorGithubApp,omitempty"`
	SkipReviewerRequest   bool   `json:"skipReviewerRequest,omitempty"`
	BranchName            string `json:"branchName,omitempty"`
}

type LaunchAgentWebhook struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

type LaunchAgentRequest struct {
	Prompt  LaunchAgentPrompt   `json:"prompt"`
	Source  LaunchAgentSource   `json:"source"`
	Target  *LaunchAgentTarget  `json:"target,omitempty"`
	Webhook *LaunchAgentWebhook `json:"webhook,omitempty"`
}

type LaunchAgentResponse struct {
	ID        string             `json:"id"`
	Name      string             `json:"name,omitempty"`
	Status    string             `json:"status,omitempty"`
	Source    *LaunchAgentSource `json:"source,omitempty"`
	Target    *LaunchAgentTarget `json:"target,omitempty"`
	CreatedAt string             `json:"createdAt,omitempty"`
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	return err
}

func (c *Client) ListModels() ([]string, error) {
	responseBody, err := c.execRequest(http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Models, nil
}

func (c *Client) LaunchAgent(request LaunchAgentRequest) (*LaunchAgentResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, c.BaseURL+"/agents", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response LaunchAgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &response, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.APIKey, "")

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
