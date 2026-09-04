package opencodego

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://opencode.ai/zen/go/v1"

// generationTimeout bounds chat completion requests. They block until the
// model finishes, which regularly takes longer than the platform's default
// request timeout.
const generationTimeout = 5 * time.Minute

type Client struct {
	APIKey string
	http   core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	apiKey, _ := ctx.GetConfig("apiKey")
	if len(apiKey) == 0 {
		return nil, fmt.Errorf("apiKey is required")
	}

	return &Client{
		APIKey: string(apiKey),
		http:   httpClient,
	}, nil
}

type ChatCompletionRequest struct {
	Model          string          `json:"model,omitempty"`
	Messages       []Message       `json:"messages"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// Message is one chat message. Content is a plain string for text-only turns
// and a []ContentPart when attachments are inlined alongside the prompt.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentPart is one block of a multipart user message: text, image_url, or
// file. OpenCode Go has no Files API, so attachments are inlined here as
// base64 data URLs.
type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
	File     *FilePart `json:"file,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type FilePart struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

// ResponseFormat asks for a schema-constrained reply. The gateway forwards it
// as the standard OpenAI-compatible response_format object.
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

type JSONSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type MessagesRequest struct {
	Model        string                `json:"model"`
	Messages     []Message             `json:"messages"`
	System       string                `json:"system,omitempty"`
	MaxTokens    int                   `json:"max_tokens"`
	Temperature  *float64              `json:"temperature,omitempty"`
	OutputConfig *MessagesOutputConfig `json:"output_config,omitempty"`
}

type MessageContentBlock struct {
	Type   string                `json:"type"`
	Text   string                `json:"text,omitempty"`
	Source *MessageContentSource `json:"source,omitempty"`
}

type MessageContentSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type MessagesOutputConfig struct {
	Format *MessagesOutputFormat `json:"format,omitempty"`
}

type MessagesOutputFormat struct {
	Type   string `json:"type"`
	Schema any    `json:"schema"`
}

type MessagesResponse struct {
	ID         string                `json:"id"`
	Type       string                `json:"type"`
	Role       string                `json:"role"`
	Content    []MessageContentBlock `json:"content"`
	Model      string                `json:"model"`
	StopReason string                `json:"stop_reason"`
	Usage      MessagesUsage         `json:"usage"`
}

type MessagesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ResponsesRequest struct {
	Model           string               `json:"model"`
	Input           any                  `json:"input"`
	MaxOutputTokens *int                 `json:"max_output_tokens,omitempty"`
	Temperature     *float64             `json:"temperature,omitempty"`
	Text            *ResponsesTextConfig `json:"text,omitempty"`
}

type ResponsesTextConfig struct {
	Format *ResponsesFormat `json:"format,omitempty"`
}

type ResponsesFormat struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Schema any    `json:"schema"`
	Strict bool   `json:"strict,omitempty"`
}

type ResponsesContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal,omitempty"`
}

type ResponsesOutput struct {
	Type    string             `json:"type"`
	Role    string             `json:"role"`
	Content []ResponsesContent `json:"content"`
}

type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type ResponsesResponse struct {
	ID         string            `json:"id"`
	Model      string            `json:"model"`
	OutputText string            `json:"output_text"`
	Output     []ResponsesOutput `json:"output"`
	Usage      *ResponsesUsage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int           `json:"index"`
	FinishReason string        `json:"finish_reason"`
	Message      ChoiceMessage `json:"message"`
}

// ChoiceMessage carries the generated message. Content is a pointer because a
// null content is distinct from an empty string.
type ChoiceMessage struct {
	Role    string  `json:"role"`
	Content *string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PlanUsageWindow is one Go plan limit window: rolling 5 hours, weekly, or
// monthly. Percent is the share of the window's dollar cap already used, and
// status reports whether requests are still being served ("ok") or throttled.
type PlanUsageWindow struct {
	Status   string  `json:"status"`
	Percent  float64 `json:"percent"`
	ResetsAt string  `json:"resetsAt"`
}

// PlanUsage holds the three limit windows of the Go subscription.
type PlanUsage struct {
	Rolling PlanUsageWindow `json:"rolling"`
	Weekly  PlanUsageWindow `json:"weekly"`
	Monthly PlanUsageWindow `json:"monthly"`
}

// planUsageResponse wraps the gateway's {"usage": {...}} envelope.
type planUsageResponse struct {
	Usage PlanUsage `json:"usage"`
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, baseURL+"/usage", nil)
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models response: %v", err)
	}

	return response.Data, nil
}

func (c *Client) CreateChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	responseBody, err := c.execRequestCtx(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat completion response: %v", err)
	}

	return &response, nil
}

func (c *Client) CreateMessage(req MessagesRequest) (*MessagesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	// The gateway reads the key from x-api-key on the Messages endpoint and
	// from Authorization on the other endpoints.
	responseBody, err := c.execRequestCtx(ctx, http.MethodPost, baseURL+"/messages", bytes.NewReader(body), extraHeaders{
		"x-api-key": c.APIKey,
	})
	if err != nil {
		return nil, err
	}

	var response MessagesResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages response: %v", err)
	}

	return &response, nil
}

func (c *Client) CreateResponse(req ResponsesRequest) (*ResponsesResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	responseBody, err := c.execRequestCtx(ctx, http.MethodPost, baseURL+"/responses", bytes.NewReader(body), nil)
	if err != nil {
		return nil, err
	}

	var response ResponsesResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal responses response: %v", err)
	}

	return &response, nil
}

// GetPlanUsage reads the Go subscription's limit windows from the gateway.
func (c *Client) GetPlanUsage() (*PlanUsage, error) {
	responseBody, err := c.execRequest(http.MethodGet, baseURL+"/usage", nil)
	if err != nil {
		return nil, err
	}

	var response planUsageResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage response: %v", err)
	}

	return &response.Usage, nil
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestCtx(context.Background(), method, URL, body, nil)
}

// extraHeaders holds per-endpoint headers on top of the defaults.
type extraHeaders map[string]string

func (c *Client) execRequestCtx(ctx context.Context, method, URL string, body io.Reader, extra extraHeaders) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	for name, value := range extra {
		req.Header.Set(name, value)
	}

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
		return nil, apiError(res.StatusCode, responseBody)
	}

	return responseBody, nil
}

// apiError turns an error body into a useful message. The gateway answers a
// missing or invalid key with Anthropic-style errors ({"type":"error",...}),
// while most other failures come back OpenAI-style ({"error":{"message":...}}).
func apiError(statusCode int, body []byte) error {
	var parsed struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Error.Message == "" {
		return fmt.Errorf("request got %d code: %s", statusCode, string(body))
	}

	message := parsed.Error.Message
	if parsed.Error.Type != "" {
		message = fmt.Sprintf("%s: %s", parsed.Error.Type, message)
	}

	return fmt.Errorf("request got %d code: %s", statusCode, message)
}
