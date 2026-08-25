package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type anthropicClient struct {
	http    core.HTTPContext
	apiKey  string
	baseURL string
}

func newAnthropicClient(httpClient core.HTTPContext, creds Credentials) *anthropicClient {
	return &anthropicClient{
		http:    httpClient,
		apiKey:  creds.APIKey,
		baseURL: resolveBaseURL(creds.BaseURL, defaultAnthropicBaseURL),
	}
}

func (c *anthropicClient) Provider() string { return ProviderAnthropic }

type anthropicModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *anthropicClient) ListModels(ctx context.Context) ([]Model, error) {
	body, err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/models", nil, false)
	if err != nil {
		return nil, err
	}

	var response anthropicModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode anthropic models: %w", err)
	}

	models := make([]Model, 0, len(response.Data))
	for _, item := range response.Data {
		if id := strings.TrimSpace(item.ID); id != "" {
			models = append(models, Model{ID: id})
		}
	}
	return models, nil
}

type anthropicMessageRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicMessageResponse struct {
	Model   string `json:"model"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage anthropicUsage `json:"usage"`
}

type anthropicUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
}

func (c *anthropicClient) Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, error) {
	payload, err := json.Marshal(c.messageRequest(req, false))
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	body, err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(payload), false)
	if err != nil {
		return nil, err
	}

	var response anthropicMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode anthropic message: %w", err)
	}

	return &CompleteResponse{
		Text:  anthropicText(response.Content),
		Model: firstNonEmpty(response.Model, req.Model),
		Usage: response.Usage.toUsage(),
	}, nil
}

func (c *anthropicClient) Stream(ctx context.Context, req CompleteRequest, emit func(StreamEvent) error) error {
	if emit == nil {
		return fmt.Errorf("stream handler is required")
	}

	payload, err := json.Marshal(c.messageRequest(req, true))
	if err != nil {
		return fmt.Errorf("marshal anthropic stream request: %w", err)
	}

	res, err := c.do(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(payload), true)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return readSSE(res.Body, func(data string) error {
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil
		}
		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text == "" {
				return nil
			}
			return emit(StreamEvent{Delta: event.Delta.Text})
		case "message_delta":
			if event.Usage == nil {
				return nil
			}
			usage := event.Usage.toUsage()
			return emit(StreamEvent{Done: true, Usage: &usage})
		}
		return nil
	})
}

func (c *anthropicClient) messageRequest(req CompleteRequest, stream bool) anthropicMessageRequest {
	return anthropicMessageRequest{
		Model:     req.Model,
		MaxTokens: resolveMaxTokens(req.MaxTokens),
		System:    strings.TrimSpace(req.System),
		Messages:  []anthropicMessage{{Role: "user", Content: req.Prompt}},
		Stream:    stream,
	}
}

func (c *anthropicClient) doJSON(ctx context.Context, method, url string, body io.Reader, stream bool) ([]byte, error) {
	res, err := c.do(ctx, method, url, body, stream)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func (c *anthropicClient) do(ctx context.Context, method, url string, body io.Reader, stream bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build anthropic request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		defer res.Body.Close()
		msg, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("anthropic request failed (%d): %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return res, nil
}

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"delta"`
	Usage *anthropicUsage `json:"usage"`
}

func (u anthropicUsage) toUsage() Usage {
	total := u.InputTokens + u.OutputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
	return Usage{
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		CacheReadTokens:  u.CacheReadInputTokens,
		CacheWriteTokens: u.CacheCreationInputTokens,
		TotalTokens:      total,
	}
}

func anthropicText(blocks []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	var b strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	return b.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
