package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

const (
	defaultBaseURL        = "https://api.openai.com/v1"
	defaultMaxRetries     = 3
	defaultInitialBackoff = 300 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
)

type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	MaxRetries int
}

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	maxRetries int
}

type apiError struct {
	statusCode int
	body       string
	retryAfter time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("openai llm: API %d: %s", e.statusCode, e.body)
}

var _ llm.Client = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("openai llm: APIKey is required")
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	return &Client{
		apiKey:     cfg.APIKey,
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      strings.TrimSpace(cfg.Model),
		httpClient: httpClient,
		maxRetries: maxRetries,
	}, nil
}

func (c *Client) Stream(ctx context.Context, req llm.StreamRequest, onEvent func(llm.StreamEvent) error) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		emitted := false
		err := c.streamOnce(ctx, req, func(event llm.StreamEvent) error {
			emitted = true
			return onEvent(event)
		})
		if err == nil {
			return nil
		}
		if emitted || !retryable(err) || attempt == c.maxRetries {
			return err
		}
		lastErr = err
		if err := sleepBeforeRetry(ctx, err, attempt); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *Client) streamOnce(ctx context.Context, req llm.StreamRequest, onEvent func(llm.StreamEvent) error) error {
	request := c.chatCompletionRequest(req)
	request.Stream = true
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("openai llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("openai llm: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai llm: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errorFromResponse(resp)
	}
	return c.forwardStream(resp.Body, onEvent)
}

func (c *Client) Complete(ctx context.Context, req llm.StreamRequest, onEvent func(llm.StreamEvent) error) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		emitted := false
		err := c.completeOnce(ctx, req, func(event llm.StreamEvent) error {
			emitted = true
			return onEvent(event)
		})
		if err == nil {
			return nil
		}
		if emitted || !retryable(err) || attempt == c.maxRetries {
			return err
		}
		lastErr = err
		if err := sleepBeforeRetry(ctx, err, attempt); err != nil {
			return err
		}
	}
	return lastErr
}

func (c *Client) completeOnce(ctx context.Context, req llm.StreamRequest, onEvent func(llm.StreamEvent) error) error {
	request := c.chatCompletionRequest(req)
	request.Stream = false
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("openai llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("openai llm: build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("openai llm: request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("openai llm: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return errorFromData(resp, data)
	}
	return forwardCompletion(data, onEvent)
}

func errorFromResponse(resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("openai llm: read error response: %w", err)
	}
	return errorFromData(resp, data)
}

func errorFromData(resp *http.Response, data []byte) error {
	return &apiError{
		statusCode: resp.StatusCode,
		body:       truncate(string(data), 500),
		retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
	}
}

func retryable(err error) bool {
	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.statusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func sleepBeforeRetry(ctx context.Context, err error, attempt int) error {
	delay := retryDelay(err, attempt)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryDelay(err error, attempt int) time.Duration {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.retryAfter > 0 {
		return min(apiErr.retryAfter, defaultMaxBackoff)
	}
	backoff := defaultInitialBackoff << attempt
	backoff = min(backoff, defaultMaxBackoff)
	jitter := time.Duration(rand.Int63n(int64(backoff)))
	return backoff/2 + jitter/2
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay > 0 {
			return delay
		}
	}
	return 0
}

func forwardCompletion(data []byte, onEvent func(llm.StreamEvent) error) error {
	var payload chatCompletionResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("openai llm: decode response: %w", err)
	}
	if len(payload.Choices) == 0 {
		return fmt.Errorf("openai llm: response had no choices")
	}
	message := payload.Choices[0].Message
	if message.Content != "" {
		if err := onEvent(llm.StreamEvent{Type: llm.StreamEventTextDelta, Text: message.Content}); err != nil {
			return err
		}
	}
	for _, toolCall := range message.ToolCalls {
		input := toolCall.Function.Arguments
		if strings.TrimSpace(input) == "" {
			input = "{}"
		}
		if err := onEvent(llm.StreamEvent{
			Type: llm.StreamEventToolCall,
			ToolCall: &llm.ToolCall{
				ID:    toolCall.ID,
				Name:  toolCall.Function.Name,
				Input: input,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) forwardStream(body io.Reader, onEvent func(llm.StreamEvent) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	accumulator := newToolCallAccumulator()
	for scanner.Scan() {
		payload, ok := ssePayload(scanner.Text())
		if !ok {
			continue
		}
		if payload == "[DONE]" {
			return accumulator.flush(onEvent)
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("openai llm: decode stream chunk: %w", err)
		}
		if err := forwardChunk(chunk, accumulator, onEvent); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("openai llm: read stream: %w", err)
	}
	return accumulator.flush(onEvent)
}

func forwardChunk(chunk streamChunk, accumulator *toolCallAccumulator, onEvent func(llm.StreamEvent) error) error {
	for _, choice := range chunk.Choices {
		if choice.Delta.Content != "" {
			if err := onEvent(llm.StreamEvent{Type: llm.StreamEventTextDelta, Text: choice.Delta.Content}); err != nil {
				return err
			}
		}
		for _, delta := range choice.Delta.ToolCalls {
			accumulator.add(delta)
		}
	}
	return nil
}

func ssePayload(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
	return payload, payload != ""
}

func (c *Client) chatCompletionRequest(req llm.StreamRequest) chatCompletionRequest {
	model := req.Model
	if model == "" {
		model = c.model
	}
	messages := make([]chatMessage, 0, len(req.Messages))
	for _, message := range req.Messages {
		messages = append(messages, chatMessages(message)...)
	}
	return chatCompletionRequest{
		Model:    model,
		Messages: messages,
		Tools:    toolDefinitions(req.Tools),
	}
}

func chatMessages(message llm.Message) []chatMessage {
	switch message.Role {
	case llm.RoleTool:
		return toolMessages(message.Blocks)
	default:
		content := textContent(message.Blocks)
		if content == "" && len(message.Blocks) == 0 {
			return nil
		}
		out := chatMessage{
			Role:    string(message.Role),
			Content: content,
		}
		for _, block := range message.Blocks {
			if block.Type != llm.BlockTypeToolUse || block.ToolCall == nil {
				continue
			}
			out.ToolCalls = append(out.ToolCalls, chatToolCall{
				ID:   block.ToolCall.ID,
				Type: "function",
				Function: chatFunctionCall{
					Name:      block.ToolCall.Name,
					Arguments: block.ToolCall.Input,
				},
			})
		}
		return []chatMessage{out}
	}
}

func toolMessages(blocks []llm.Block) []chatMessage {
	messages := []chatMessage{}
	for _, block := range blocks {
		if block.Type != llm.BlockTypeToolResult || block.ToolResult == nil {
			continue
		}
		messages = append(messages, chatMessage{
			Role:       "tool",
			Content:    block.ToolResult.Content,
			ToolCallID: block.ToolResult.ToolCallID,
		})
	}
	return messages
}

func textContent(blocks []llm.Block) string {
	parts := []string{}
	for _, block := range blocks {
		if block.Type == llm.BlockTypeText && block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "")
}

func toolDefinitions(definitions []llm.ToolDefinition) []chatTool {
	tools := make([]chatTool, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, chatTool{
			Type: "function",
			Function: chatToolFunction{
				Name:        definition.Name,
				Description: definition.Description,
				Parameters:  definition.InputSchema,
			},
		})
	}
	return tools
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatTool struct {
	Type     string           `json:"type"`
	Function chatToolFunction `json:"function"`
}

type chatToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessageResponse `json:"message"`
}

type chatMessageResponse struct {
	Content   string         `json:"content"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
}

type streamChoice struct {
	Delta streamDelta `json:"delta"`
}

type streamDelta struct {
	Content   string                `json:"content"`
	ToolCalls []streamToolCallDelta `json:"tool_calls"`
}

type streamToolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type toolCallAccumulator struct {
	ordered []int
	byIndex map[int]*llm.ToolCall
}

func newToolCallAccumulator() *toolCallAccumulator {
	return &toolCallAccumulator{byIndex: map[int]*llm.ToolCall{}}
}

func (a *toolCallAccumulator) add(delta streamToolCallDelta) {
	toolCall, ok := a.byIndex[delta.Index]
	if !ok {
		toolCall = &llm.ToolCall{}
		a.byIndex[delta.Index] = toolCall
		a.ordered = append(a.ordered, delta.Index)
	}
	if delta.ID != "" {
		toolCall.ID = delta.ID
	}
	if delta.Function.Name != "" {
		toolCall.Name += delta.Function.Name
	}
	if delta.Function.Arguments != "" {
		toolCall.Input += delta.Function.Arguments
	}
}

func (a *toolCallAccumulator) flush(onEvent func(llm.StreamEvent) error) error {
	for _, index := range a.ordered {
		toolCall := a.byIndex[index]
		if strings.TrimSpace(toolCall.Input) == "" {
			toolCall.Input = "{}"
		}
		if err := onEvent(llm.StreamEvent{
			Type:     llm.StreamEventToolCall,
			ToolCall: toolCall,
		}); err != nil {
			return err
		}
	}
	a.ordered = nil
	a.byIndex = map[int]*llm.ToolCall{}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
