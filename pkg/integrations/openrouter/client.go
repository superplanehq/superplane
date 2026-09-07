package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/core"
)

const baseURL = "https://openrouter.ai/api/v1"

// OpenRouter attributes requests to the calling app through these headers and
// uses them for its public app rankings.
const (
	attributionReferer = "https://superplane.com"
	attributionTitle   = "SuperPlane"
)

// generationTimeout bounds chat completion requests. They block until the model
// finishes, and reasoning models regularly run longer than the platform's
// default request timeout.
const generationTimeout = 5 * time.Minute

type Client struct {
	APIKey        string
	ManagementKey string
	http          core.HTTPContext
}

func NewClient(httpClient core.HTTPContext, ctx core.IntegrationContext) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no integration context")
	}

	// The inference key is issued by the OAuth exchange and stored as a secret,
	// not entered as configuration.
	apiKey, err := findSecret(ctx, SecretAPIKey)
	if err != nil {
		return nil, err
	}

	managementKey, _ := ctx.GetConfig("managementKey")

	return &Client{
		APIKey:        apiKey,
		ManagementKey: string(managementKey),
		http:          httpClient,
	}, nil
}

// readAll drains a response body.
func readAll(res *http.Response) ([]byte, error) {
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	return body, nil
}

// ChatCompletionRequest is the OpenAI-compatible chat completions body.
// Model and Models are mutually exclusive: Models replaces Model with an
// ordered fallback chain whose first entry is the primary. Provider carries the
// per-request provider routing preferences.
type ChatCompletionRequest struct {
	Model          string           `json:"model,omitempty"`
	Models         []string         `json:"models,omitempty"`
	Messages       []Message        `json:"messages"`
	MaxTokens      *int             `json:"max_tokens,omitempty"`
	Temperature    *float64         `json:"temperature,omitempty"`
	Provider       *ProviderRouting `json:"provider,omitempty"`
	ResponseFormat *ResponseFormat  `json:"response_format,omitempty"`
	Plugins        []Plugin         `json:"plugins,omitempty"`
}

// ResponseFormat constrains the reply to a JSON schema. Note this is the chat
// completions shape (response_format.json_schema), not the Responses API's
// text.format that the OpenAI integration uses.
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

type JSONSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

// Plugin enables an OpenRouter plugin. The web plugin runs a search before the
// model answers and is billed on top of tokens, even on free models.
type Plugin struct {
	ID         string `json:"id"`
	MaxResults *int   `json:"max_results,omitempty"`
}

// ProviderRouting controls which upstream provider serves the request. A model
// ID is a listing served by many providers that differ in price, context
// length and quantization, so these preferences are how a workflow pins that
// choice instead of letting OpenRouter pick silently.
type ProviderRouting struct {
	Sort              string   `json:"sort,omitempty"`
	Only              []string `json:"only,omitempty"`
	Ignore            []string `json:"ignore,omitempty"`
	AllowFallbacks    *bool    `json:"allow_fallbacks,omitempty"`
	RequireParameters *bool    `json:"require_parameters,omitempty"`
	DataCollection    string   `json:"data_collection,omitempty"`
}

// Message is one chat message. Content is a plain string for text-only
// messages, or []ContentPart when attachments are inlined.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// ContentPart is a content block: text, image_url, or file. OpenRouter has no
// Files API, so attachments are inlined here as base64 data URLs.
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

type ChatCompletionResponse struct {
	ID       string   `json:"id"`
	Object   string   `json:"object"`
	Created  int64    `json:"created"`
	Model    string   `json:"model"`
	Provider string   `json:"provider"`
	Choices  []Choice `json:"choices"`
	Usage    *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index              int           `json:"index"`
	FinishReason       string        `json:"finish_reason"`
	NativeFinishReason string        `json:"native_finish_reason"`
	Message            ChoiceMessage `json:"message"`
}

// ChoiceMessage carries the generated message. Content is a pointer because
// OpenRouter returns null when reasoning tokens consumed the whole token
// budget, which is distinct from an empty string.
type ChoiceMessage struct {
	Role        string       `json:"role"`
	Content     *string      `json:"content"`
	Reasoning   string       `json:"reasoning,omitempty"`
	Refusal     string       `json:"refusal,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation is a source the web plugin cited.
type Annotation struct {
	Type        string       `json:"type"`
	URLCitation *URLCitation `json:"url_citation,omitempty"`
}

type URLCitation struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content,omitempty"`
}

type Usage struct {
	PromptTokens            int                      `json:"prompt_tokens"`
	CompletionTokens        int                      `json:"completion_tokens"`
	TotalTokens             int                      `json:"total_tokens"`
	Cost                    float64                  `json:"cost"`
	CostDetails             *CostDetails             `json:"cost_details,omitempty"`
	PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type CostDetails struct {
	UpstreamInferenceCost           float64 `json:"upstream_inference_cost"`
	UpstreamInferencePromptCost     float64 `json:"upstream_inference_prompt_cost"`
	UpstreamInferenceCompletionCost float64 `json:"upstream_inference_completions_cost"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
	AudioTokens  int `json:"audio_tokens"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
	AudioTokens     int `json:"audio_tokens"`
}

type ModelsResponse struct {
	Data []Model `json:"data"`
}

type Model struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ModelEndpointsResponse struct {
	Data struct {
		ID        string          `json:"id"`
		Endpoints []ModelEndpoint `json:"endpoints"`
	} `json:"data"`
}

// ModelEndpoint is one provider serving a model. Tag identifies the specific
// endpoint and may carry a region suffix (e.g. "azure/swedencentral"); the part
// before the slash is the provider slug that routing accepts.
type ModelEndpoint struct {
	Name         string `json:"name"`
	ProviderName string `json:"provider_name"`
	Tag          string `json:"tag"`
	Quantization string `json:"quantization"`
}

type ProvidersResponse struct {
	Data []Provider `json:"data"`
}

type Provider struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CreditsResponse struct {
	Data Credits `json:"data"`
}

type Credits struct {
	TotalCredits float64 `json:"total_credits"`
	TotalUsage   float64 `json:"total_usage"`
}

type KeyResponse struct {
	Data KeyInfo `json:"data"`
}

type KeyInfo struct {
	Label          string   `json:"label"`
	Limit          *float64 `json:"limit"`
	LimitRemaining *float64 `json:"limit_remaining"`
	LimitReset     *string  `json:"limit_reset"`
	Usage          float64  `json:"usage"`
	UsageDaily     float64  `json:"usage_daily"`
	UsageWeekly    float64  `json:"usage_weekly"`
	UsageMonthly   float64  `json:"usage_monthly"`
	IsFreeTier     bool     `json:"is_free_tier"`
}

func (c *Client) Verify() error {
	_, err := c.execRequest(http.MethodGet, baseURL+"/key", nil)
	return err
}

// VerifyManagement checks the management key against an endpoint that rejects
// normal API keys with 403.
func (c *Client) VerifyManagement() error {
	_, err := c.execRequestWithKey(context.Background(), http.MethodGet, baseURL+"/activity", nil, c.ManagementKey)
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

// ListModelEndpoints returns every provider endpoint serving the given model.
func (c *Client) ListModelEndpoints(model string) ([]ModelEndpoint, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/models/"+modelPath(model)+"/endpoints", nil)
	if err != nil {
		return nil, err
	}

	var response ModelEndpointsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal model endpoints response: %v", err)
	}

	return response.Data.Endpoints, nil
}

func (c *Client) ListProviders() ([]Provider, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/providers", nil)
	if err != nil {
		return nil, err
	}

	var response ProvidersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal providers response: %v", err)
	}

	return response.Data, nil
}

// GetCredits reads the account credit totals. OpenRouter documents /credits as a
// management-key endpoint, so it is signed with the provisioning key rather than
// the inference key, even though the endpoint currently also accepts the latter.
func (c *Client) GetCredits() (*Credits, error) {
	if c.ManagementKey == "" {
		return nil, fmt.Errorf("provisioning API key is not configured")
	}

	body, err := c.execRequestWithKey(context.Background(), http.MethodGet, baseURL+"/credits", nil, c.ManagementKey)
	if err != nil {
		return nil, err
	}

	var response CreditsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credits response: %v", err)
	}

	return &response.Data, nil
}

// GetKey reports usage and limits for the key that signs the request, so it uses
// the inference key: the provisioning key's own usage is not what the component
// reports on.
func (c *Client) GetKey() (*KeyInfo, error) {
	body, err := c.execRequest(http.MethodGet, baseURL+"/key", nil)
	if err != nil {
		return nil, err
	}

	var response KeyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key response: %v", err)
	}

	return &response.Data, nil
}

func (c *Client) CreateChatCompletion(req ChatCompletionRequest) (*ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), generationTimeout)
	defer cancel()

	responseBody, err := c.execRequestCtx(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat completion response: %v", err)
	}

	return &response, nil
}

// modelPath escapes a model ID for use in a URL path. IDs are author/slug, so
// the separating slash must survive escaping.
func modelPath(model string) string {
	parts := strings.Split(model, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (c *Client) execRequest(method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestCtx(context.Background(), method, URL, body)
}

func (c *Client) execRequestCtx(ctx context.Context, method, URL string, body io.Reader) ([]byte, error) {
	return c.execRequestWithKey(ctx, method, URL, body, c.APIKey)
}

func (c *Client) execRequestWithKey(ctx context.Context, method, URL string, body io.Reader, key string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("HTTP-Referer", attributionReferer)
	req.Header.Set("X-Title", attributionTitle)

	res, err := c.http.Do(req)
	if err != nil {
		message := fmt.Sprintf("request failed: %v", err)
		return nil, core.NewProviderTransportError(message, errors.New(message))
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

// apiError turns OpenRouter's {"error":{...}} body into a message that keeps
// the details callers act on: the providers actually serving a model when
// routing excluded them all, and the backoff hint on a rate limit. The
// returned error is a *core.ProviderAPIError so callers can classify the
// failure (auth, rate limit, unavailable) without matching on message text.
func apiError(statusCode int, body []byte) error {
	var parsed struct {
		Error struct {
			Message  string         `json:"message"`
			Metadata map[string]any `json:"metadata"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Error.Message == "" {
		message := fmt.Sprintf("request got %d code: %s", statusCode, string(body))
		return core.NewProviderAPIError(statusCode, message, errors.New(message))
	}

	message := fmt.Sprintf("request got %d code: %s", statusCode, parsed.Error.Message)

	if available, ok := parsed.Error.Metadata["available_providers"]; ok {
		message += fmt.Sprintf(" (available providers: %v)", available)
	}

	if retryAfter, ok := parsed.Error.Metadata["retry_after_seconds"]; ok {
		message += fmt.Sprintf(" (retry after %v seconds)", retryAfter)
	}

	return core.NewProviderAPIError(statusCode, message, errors.New(message))
}
