package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		baseURL: resolveBaseURL(creds.BaseURL, DefaultAnthropicBaseURL),
	}
}

func (c *anthropicClient) Provider() string { return ProviderAnthropic }

type anthropicModelsResponse struct {
	Data []struct {
		ID           string `json:"id"`
		Capabilities *struct {
			Effort *anthropicEffortCapability `json:"effort"`
		} `json:"capabilities"`
	} `json:"data"`
	HasMore bool   `json:"has_more"`
	LastID  string `json:"last_id"`
}

type anthropicCapability struct {
	Supported bool `json:"supported"`
}

type anthropicEffortCapability struct {
	Supported bool                `json:"supported"`
	Low       anthropicCapability `json:"low"`
	Medium    anthropicCapability `json:"medium"`
	High      anthropicCapability `json:"high"`
	XHigh     anthropicCapability `json:"xhigh"`
	Max       anthropicCapability `json:"max"`
}

func (e *anthropicEffortCapability) thinkingLevels() []string {
	levels := []string{}
	if !e.Supported {
		return levels
	}
	for _, level := range []struct {
		name       string
		capability anthropicCapability
	}{
		{"low", e.Low}, {"medium", e.Medium}, {"high", e.High}, {"xhigh", e.XHigh}, {"max", e.Max},
	} {
		if level.capability.Supported {
			levels = append(levels, level.name)
		}
	}
	return levels
}

func (c *anthropicClient) ListModels(ctx context.Context) ([]Model, error) {
	listed := []Model{}
	pageURL := c.baseURL + "/models"
	seenCursors := map[string]bool{}
	for {
		body, err := c.doJSON(ctx, http.MethodGet, pageURL, nil, false)
		if err != nil {
			return nil, err
		}
		var response anthropicModelsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("decode anthropic models: %w", err)
		}
		for _, item := range response.Data {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			model := Model{ID: id}
			if item.Capabilities != nil && item.Capabilities.Effort != nil {
				model.ThinkingLevels = item.Capabilities.Effort.thinkingLevels()
			}
			listed = append(listed, model)
		}
		if !response.HasMore {
			return listed, nil
		}
		if response.LastID == "" || seenCursors[response.LastID] {
			return nil, fmt.Errorf("anthropic models returned an invalid pagination cursor")
		}
		seenCursors[response.LastID] = true
		pageURL = c.baseURL + "/models?after_id=" + url.QueryEscape(response.LastID)
	}
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
		return nil, core.NewProviderTransportError("anthropic request failed", err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		message := strings.TrimSpace(string(body))
		var response struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &response) == nil && response.Error.Message != "" {
			message = response.Error.Message
		}
		err := fmt.Errorf("request failed (%d): %s", res.StatusCode, message)
		if res.StatusCode == http.StatusUnauthorized {
			err = fmt.Errorf("Claude credentials are invalid or expired: %s", message)
		}
		return nil, core.NewProviderAPIError(res.StatusCode, err.Error(), err)
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
