package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	defaultBaseURL    = "https://api.groq.com/openai/v1"
	generationTimeout = 5 * time.Minute
)

// Client wraps the subset of Groq's OpenAI-compatible API used by the
// integration. The base URL is fixed so model verification and generation
// always target GroqCloud.
type Client struct {
	APIKey  string
	BaseURL string
	http    core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	key, err := integrationAPIKey(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		APIKey:  key,
		BaseURL: defaultBaseURL,
		http:    httpClient,
	}, nil
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

// Model is the model metadata returned by GET /models. Groq currently marks
// inactive models with active=false; a missing active field remains usable for
// compatibility with older responses and test doubles.
type Model struct {
	ID                  string `json:"id"`
	Object              string `json:"object"`
	Created             int64  `json:"created"`
	OwnedBy             string `json:"owned_by"`
	Active              *bool  `json:"active"`
	ContextWindow       int    `json:"context_window"`
	MaxCompletionTokens int    `json:"max_completion_tokens"`
}

const (
	speechToTextModelMarker  = "whisper"
	textToSpeechModelMarker  = "tts"
	orpheusSpeechModelPrefix = "canopylabs/orpheus-"
)

func (m Model) IsSelectable() bool {
	if strings.TrimSpace(m.ID) == "" || (m.Active != nil && !*m.Active) {
		return false
	}

	// Groq's model metadata does not expose a chat capability flag. Exclude
	// known speech models and keep other active IDs discoverable for chat.
	return !isKnownNonChatModel(m.ID)
}

func isKnownNonChatModel(modelID string) bool {
	normalizedID := strings.ToLower(strings.TrimSpace(modelID))
	return strings.Contains(normalizedID, speechToTextModelMarker) ||
		strings.Contains(normalizedID, textToSpeechModelMarker) ||
		strings.HasPrefix(normalizedID, orpheusSpeechModelPrefix)
}

type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID                string               `json:"id"`
	Object            string               `json:"object"`
	Created           int64                `json:"created"`
	Model             string               `json:"model"`
	SystemFingerprint string               `json:"system_fingerprint,omitempty"`
	Choices           []ChatChoice         `json:"choices"`
	Usage             *ChatCompletionUsage `json:"usage,omitempty"`
}

type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type ChatCompletionUsage struct {
	QueueTime        float64 `json:"queue_time,omitempty"`
	PromptTime       float64 `json:"prompt_time,omitempty"`
	CompletionTime   float64 `json:"completion_time,omitempty"`
	TotalTime        float64 `json:"total_time,omitempty"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
}

func (c *Client) Verify() error {
	_, err := c.ListModels()
	return err
}

func (c *Client) ListModels() ([]Model, error) {
	responseBody, err := c.execRequest(context.Background(), http.MethodGet, c.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}

	var response ModelsResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("Groq API returned an invalid models response")
	}

	return response.Data, nil
}

func (c *Client) CreateChatCompletion(request ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Groq chat completion request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	responseBody, err := c.execRequest(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("Groq API returned an invalid chat completion response")
	}

	return &response, nil
}

func (c *Client) execRequest(ctx context.Context, method, requestURL string, body io.Reader) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("could not create Groq API request")
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.APIKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("could not reach Groq API; check your connection and try again")
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil, groqAPIStatusError(response.StatusCode)
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read the Groq API response; try again")
	}

	return responseBody, nil
}

func groqAPIStatusError(statusCode int) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return fmt.Errorf("Groq API rejected the request (%d); verify your API key", statusCode)
	case statusCode == http.StatusRequestTimeout:
		return fmt.Errorf("Groq API request timed out (%d); try again", statusCode)
	case statusCode == http.StatusTooManyRequests:
		return fmt.Errorf("Groq API rate limit reached (%d); wait and try again", statusCode)
	case statusCode >= http.StatusInternalServerError:
		return fmt.Errorf("Groq API is unavailable (%d); try again later", statusCode)
	default:
		return fmt.Errorf("Groq API rejected the request (%d); review the request and try again", statusCode)
	}
}
