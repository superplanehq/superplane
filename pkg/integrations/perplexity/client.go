package perplexity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://api.perplexity.ai"

type Client struct {
	APIKey string
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
		APIKey: string(apiKey),
		http:   httpClient,
	}, nil
}

// ModelsResponse represents the response from GET /v1/models.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
}

// AgentTool represents a tool in an agent request.
type AgentTool struct {
	Type string `json:"type"`
}

// AgentRequest represents the request body for POST /v1/responses.
type AgentRequest struct {
	Preset       string      `json:"preset,omitempty"`
	Model        string      `json:"model,omitempty"`
	Input        string      `json:"input"`
	Instructions string      `json:"instructions,omitempty"`
	Tools        []AgentTool `json:"tools,omitempty"`
	Temperature  float64     `json:"temperature,omitempty"`
}

// AgentAnnotation is a citation annotation in the agent response.
type AgentAnnotation struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// AgentContent is a single content block in an agent output item.
type AgentContent struct {
	Type        string            `json:"type"`
	Text        string            `json:"text"`
	Annotations []AgentAnnotation `json:"annotations"`
}

// AgentOutput is a single output item in the agent response.
type AgentOutput struct {
	Content []AgentContent `json:"content"`
}

// AgentCost holds cost information from usage.
type AgentCost struct {
	TotalCost float64 `json:"total_cost"`
}

// AgentUsage holds token and cost usage from the agent response.
type AgentUsage struct {
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	Cost         AgentCost `json:"cost"`
}

// AgentResponse represents the response from POST /v1/responses.
type AgentResponse struct {
	ID     string        `json:"id"`
	Model  string        `json:"model"`
	Status string        `json:"status"`
	Output []AgentOutput `json:"output"`
	Usage  *AgentUsage   `json:"usage,omitempty"`
}

func (c *Client) Verify() error {
	_, err := c.ListModels()
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) CreateAgentResponse(req AgentRequest) (*AgentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent request: %v", err)
	}

	responseBody, err := c.execRequest(http.MethodPost, baseURL+"/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response AgentResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent response: %v", err)
	}

	return &response, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

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
