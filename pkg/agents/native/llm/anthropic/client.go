package anthropic

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
	defaultBaseURL        = "https://api.anthropic.com/v1"
	defaultVersion        = "2023-06-01"
	defaultMaxRetries     = 3
	defaultInitialBackoff = 300 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	defaultMaxTokens      = 4096
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
	return fmt.Sprintf("anthropic llm: API %d: %s", e.statusCode, e.body)
}

var _ llm.Client = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("anthropic llm: APIKey is required")
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
	request := c.messagesRequest(req)
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("anthropic llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("anthropic llm: build request: %w", err)
	}
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", defaultVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("anthropic llm: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return errorFromResponse(resp)
	}
	return forwardStream(resp.Body, onEvent)
}

func (c *Client) messagesRequest(req llm.StreamRequest) messagesRequest {
	model := req.Model
	if model == "" {
		model = c.model
	}
	system, messages := anthropicMessages(req.Messages)
	return messagesRequest{
		Model:     model,
		MaxTokens: defaultMaxTokens,
		System:    system,
		Messages:  messages,
		Tools:     toolDefinitions(req.Tools),
		Stream:    true,
	}
}

func anthropicMessages(messages []llm.Message) (string, []message) {
	system := ""
	out := make([]message, 0, len(messages))
	allowedToolResults := map[string]struct{}{}
	for _, m := range messages {
		if m.Role == llm.RoleSystem && system == "" {
			system = textContent(m.Blocks)
			continue
		}

		role := string(m.Role)
		if m.Role == llm.RoleSystem || m.Role == llm.RoleTool {
			role = "user"
		}
		content := contentBlocks(m, allowedToolResults)
		if len(content) == 0 {
			continue
		}
		out = append(out, message{Role: role, Content: content})

		if role == "assistant" {
			allowedToolResults = toolUseIDs(content)
			continue
		}
		allowedToolResults = map[string]struct{}{}
	}
	return system, out
}

func contentBlocks(m llm.Message, allowedToolResults map[string]struct{}) []contentBlock {
	blocks := []contentBlock{}
	for _, block := range m.Blocks {
		switch block.Type {
		case llm.BlockTypeText:
			if block.Text != "" {
				blocks = append(blocks, contentBlock{Type: "text", Text: block.Text})
			}
		case llm.BlockTypeToolUse:
			if block.ToolCall != nil {
				blocks = append(blocks, contentBlock{
					Type:  "tool_use",
					ID:    block.ToolCall.ID,
					Name:  block.ToolCall.Name,
					Input: jsonInput(block.ToolCall.Input),
				})
			}
		case llm.BlockTypeToolResult:
			if block.ToolResult != nil {
				if _, ok := allowedToolResults[block.ToolResult.ToolCallID]; !ok {
					blocks = append(blocks, contentBlock{
						Type: "text",
						Text: orphanToolResultText(*block.ToolResult),
					})
					continue
				}
				blocks = append(blocks, contentBlock{
					Type:      "tool_result",
					ToolUseID: block.ToolResult.ToolCallID,
					Content:   block.ToolResult.Content,
					IsError:   block.ToolResult.IsError,
				})
			}
		}
	}
	return blocks
}

func toolUseIDs(blocks []contentBlock) map[string]struct{} {
	ids := map[string]struct{}{}
	for _, block := range blocks {
		if block.Type == "tool_use" && block.ID != "" {
			ids[block.ID] = struct{}{}
		}
	}
	return ids
}

func orphanToolResultText(result llm.ToolResult) string {
	status := "ok"
	if result.IsError {
		status = "error"
	}
	return fmt.Sprintf("Historical tool result for %s (%s): %s", result.Name, status, result.Content)
}

func jsonInput(input string) any {
	input = strings.TrimSpace(input)
	if input == "" {
		return map[string]any{}
	}
	var decoded any
	if err := json.Unmarshal([]byte(input), &decoded); err != nil {
		return map[string]any{"input": input}
	}
	return decoded
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

func toolDefinitions(definitions []llm.ToolDefinition) []toolDefinition {
	tools := make([]toolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, toolDefinition{
			Name:        definition.Name,
			Description: definition.Description,
			InputSchema: definition.InputSchema,
		})
	}
	return tools
}

func forwardStream(body io.Reader, onEvent func(llm.StreamEvent) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	accumulator := newToolCallAccumulator()
	for scanner.Scan() {
		payload, ok := ssePayload(scanner.Text())
		if !ok {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return fmt.Errorf("anthropic llm: decode stream event: %w", err)
		}
		if err := forwardEvent(event, accumulator, onEvent); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("anthropic llm: read stream: %w", err)
	}
	return accumulator.flush(onEvent)
}

func forwardEvent(event streamEvent, accumulator *toolCallAccumulator, onEvent func(llm.StreamEvent) error) error {
	switch event.Type {
	case "content_block_start":
		if event.ContentBlock.Type == "tool_use" {
			accumulator.start(event.Index, event.ContentBlock.ID, event.ContentBlock.Name)
		}
	case "content_block_delta":
		if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
			return onEvent(llm.StreamEvent{Type: llm.StreamEventTextDelta, Text: event.Delta.Text})
		}
		if event.Delta.Type == "input_json_delta" {
			accumulator.append(event.Index, event.Delta.PartialJSON)
		}
	case "content_block_stop":
		return accumulator.flushIndex(event.Index, onEvent)
	}
	return nil
}

func errorFromResponse(resp *http.Response) error {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("anthropic llm: read error response: %w", err)
	}
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

func ssePayload(line string) (string, bool) {
	if !strings.HasPrefix(line, "data: ") {
		return "", false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
	return payload, payload != ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

type messagesRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system,omitempty"`
	Messages  []message        `json:"messages"`
	Tools     []toolDefinition `json:"tools,omitempty"`
	Stream    bool             `json:"stream"`
}

type message struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type streamEvent struct {
	Type         string             `json:"type"`
	Index        int                `json:"index"`
	ContentBlock streamContentBlock `json:"content_block"`
	Delta        streamDelta        `json:"delta"`
}

type streamContentBlock struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type streamDelta struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	PartialJSON string `json:"partial_json"`
}

type toolCallAccumulator struct {
	ordered []int
	byIndex map[int]*llm.ToolCall
}

func newToolCallAccumulator() *toolCallAccumulator {
	return &toolCallAccumulator{byIndex: map[int]*llm.ToolCall{}}
}

func (a *toolCallAccumulator) start(index int, id, name string) {
	call, ok := a.byIndex[index]
	if !ok {
		a.ordered = append(a.ordered, index)
		call = &llm.ToolCall{}
		a.byIndex[index] = call
	}
	call.ID = id
	call.Name = name
}

func (a *toolCallAccumulator) append(index int, partialJSON string) {
	call, ok := a.byIndex[index]
	if !ok {
		a.ordered = append(a.ordered, index)
		call = &llm.ToolCall{}
		a.byIndex[index] = call
	}
	call.Input += partialJSON
}

func (a *toolCallAccumulator) flushIndex(index int, onEvent func(llm.StreamEvent) error) error {
	call, ok := a.byIndex[index]
	if !ok || call.Name == "" {
		return nil
	}
	if strings.TrimSpace(call.Input) == "" {
		call.Input = "{}"
	}
	delete(a.byIndex, index)
	return onEvent(llm.StreamEvent{Type: llm.StreamEventToolCall, ToolCall: call})
}

func (a *toolCallAccumulator) flush(onEvent func(llm.StreamEvent) error) error {
	for _, index := range a.ordered {
		if err := a.flushIndex(index, onEvent); err != nil {
			return err
		}
	}
	return nil
}
