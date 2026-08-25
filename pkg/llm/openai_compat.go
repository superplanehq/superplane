package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

type openAICompatClient struct {
	http     core.HTTPContext
	provider string
	apiKey   string
	baseURL  string
}

func newOpenAICompatClient(httpClient core.HTTPContext, provider, defaultBase string, creds Credentials) *openAICompatClient {
	return &openAICompatClient{
		http:     httpClient,
		provider: provider,
		apiKey:   creds.APIKey,
		baseURL:  resolveBaseURL(creds.BaseURL, defaultBase),
	}
}

func (c *openAICompatClient) Provider() string { return c.provider }

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

func (c *openAICompatClient) ListModels(ctx context.Context) ([]Model, error) {
	body, err := c.doJSON(ctx, http.MethodGet, c.baseURL+"/models", nil, false)
	if err != nil {
		return nil, err
	}

	var response openAIModelsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode %s models: %w", c.provider, err)
	}

	models := make([]Model, 0, len(response.Data))
	for _, item := range response.Data {
		if id := strings.TrimSpace(item.ID); id != "" {
			models = append(models, Model{ID: id})
		}
	}
	return models, nil
}

type openAIChatRequest struct {
	Model         string            `json:"model"`
	Messages      []openAIChatMsg   `json:"messages"`
	MaxTokens     int               `json:"max_tokens,omitempty"`
	Stream        bool              `json:"stream,omitempty"`
	StreamOptions *openAIStreamOpts `json:"stream_options,omitempty"`
}

type openAIChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIStreamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage openAIUsage `json:"usage"`
}

type openAIUsage struct {
	PromptTokens        int64 `json:"prompt_tokens"`
	CompletionTokens    int64 `json:"completion_tokens"`
	TotalTokens         int64 `json:"total_tokens"`
	PromptTokensDetails struct {
		CachedTokens int64 `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
	Cost *float64 `json:"cost"`
}

type openAINativeCost struct{}

func (c *openAICompatClient) Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, error) {
	payload, err := json.Marshal(c.chatRequest(req, false))
	if err != nil {
		return nil, fmt.Errorf("marshal %s request: %w", c.provider, err)
	}

	body, err := c.doJSON(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload), false)
	if err != nil {
		return nil, err
	}

	var response openAIChatResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode %s completion: %w", c.provider, err)
	}

	text := ""
	if len(response.Choices) > 0 {
		text = response.Choices[0].Message.Content
	}

	return &CompleteResponse{
		Text:  text,
		Model: firstNonEmpty(response.Model, req.Model),
		Usage: response.Usage.toUsage(),
	}, nil
}

func (c *openAICompatClient) Stream(ctx context.Context, req CompleteRequest, emit func(StreamEvent) error) error {
	if emit == nil {
		return fmt.Errorf("stream handler is required")
	}

	payload, err := json.Marshal(c.chatRequest(req, true))
	if err != nil {
		return fmt.Errorf("marshal %s stream request: %w", c.provider, err)
	}

	res, err := c.do(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload), true)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return readSSE(res.Body, func(data string) error {
		if data == "[DONE]" {
			return nil
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if err := emit(StreamEvent{Delta: chunk.Choices[0].Delta.Content}); err != nil {
				return err
			}
		}
		if chunk.Usage != nil {
			usage := chunk.Usage.toUsage()
			return emit(StreamEvent{Done: true, Usage: &usage})
		}
		return nil
	})
}

func (c *openAICompatClient) chatRequest(req CompleteRequest, stream bool) openAIChatRequest {
	messages := make([]openAIChatMsg, 0, 2)
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, openAIChatMsg{Role: "system", Content: req.System})
	}
	messages = append(messages, openAIChatMsg{Role: "user", Content: req.Prompt})

	out := openAIChatRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: resolveMaxTokens(req.MaxTokens),
		Stream:    stream,
	}
	if stream {
		out.StreamOptions = &openAIStreamOpts{IncludeUsage: true}
	}
	return out
}

func (c *openAICompatClient) doJSON(ctx context.Context, method, url string, body io.Reader, stream bool) ([]byte, error) {
	res, err := c.do(ctx, method, url, body, stream)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

func (c *openAICompatClient) do(ctx context.Context, method, url string, body io.Reader, stream bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build %s request: %w", c.provider, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.provider == ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://superplane.com")
		req.Header.Set("X-Title", "SuperPlane")
	}
	if stream {
		req.Header.Set("Accept", "text/event-stream")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", c.provider, err)
	}
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		defer res.Body.Close()
		msg, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("%s request failed (%d): %s", c.provider, res.StatusCode, strings.TrimSpace(string(msg)))
	}
	return res, nil
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *openAIUsage `json:"usage"`
}

func (u openAIUsage) toUsage() Usage {
	total := u.TotalTokens
	if total == 0 {
		total = u.PromptTokens + u.CompletionTokens
	}
	usage := Usage{
		InputTokens:     u.PromptTokens,
		OutputTokens:    u.CompletionTokens,
		CacheReadTokens: u.PromptTokensDetails.CachedTokens,
		ReasoningTokens: u.CompletionTokensDetails.ReasoningTokens,
		TotalTokens:     total,
	}
	if u.Cost != nil {
		micros := int64(*u.Cost * 1_000_000)
		usage.CostMicros = &micros
	}
	return usage
}

func readSSE(r io.Reader, handle func(data string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		if err := handle(strings.TrimSpace(data)); err != nil {
			return err
		}
	}
	return scanner.Err()
}
