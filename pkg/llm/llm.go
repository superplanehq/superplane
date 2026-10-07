package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ProviderAnthropic  = "anthropic"
	ProviderOpenAI     = "openai"
	ProviderOpenRouter = "openrouter"

	defaultAnthropicBaseURL  = "https://api.anthropic.com/v1"
	defaultOpenAIBaseURL     = "https://api.openai.com/v1"
	defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	defaultMaxTokens         = 4096
)

// Credentials are provider API credentials. Hosted catalog and later BYOK
// both supply this shape so the HTTP client stays funding-source agnostic.
type Credentials struct {
	APIKey  string
	BaseURL string
}

// Model is one id from a provider list-models response.
type Model struct {
	ID string
}

// Usage is normalized token counts for one completion.
type Usage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	TotalTokens      int64
	CostMicros       *int64
}

// ToRecord maps usage onto the ledger call-site payload.
func (u Usage) ToRecord(provider, model string) core.UsageRecord {
	total := u.TotalTokens
	if total == 0 {
		total = u.InputTokens + u.OutputTokens + u.CacheReadTokens + u.CacheWriteTokens + u.ReasoningTokens
	}
	return core.UsageRecord{
		Provider:         provider,
		Model:            model,
		InputTokens:      u.InputTokens,
		OutputTokens:     u.OutputTokens,
		CacheReadTokens:  u.CacheReadTokens,
		CacheWriteTokens: u.CacheWriteTokens,
		ReasoningTokens:  u.ReasoningTokens,
		TotalTokens:      total,
		CostMicros:       u.CostMicros,
	}
}

// CompleteRequest is a text completion. Prompt text is not stored on the ledger.
type CompleteRequest struct {
	Model     string
	System    string
	Prompt    string
	MaxTokens int
}

// CompleteResponse is the model text plus normalized usage.
type CompleteResponse struct {
	Text  string
	Model string
	Usage Usage
}

// StreamEvent is one streamed text delta or the terminal usage snapshot.
type StreamEvent struct {
	Delta string
	Done  bool
	Usage *Usage
}

// Client is the shared provider surface for list, complete, and stream.
type Client interface {
	Provider() string
	ListModels(ctx context.Context) ([]Model, error)
	Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, error)
	Stream(ctx context.Context, req CompleteRequest, emit func(StreamEvent) error) error
}

// New builds a client for one of the in-scope billing providers.
func New(httpClient core.HTTPContext, provider string, creds Credentials) (Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if strings.TrimSpace(creds.APIKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}

	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderAnthropic:
		return newAnthropicClient(httpClient, creds), nil
	case ProviderOpenAI:
		return newOpenAICompatClient(httpClient, ProviderOpenAI, defaultOpenAIBaseURL, creds), nil
	case ProviderOpenRouter:
		return newOpenAICompatClient(httpClient, ProviderOpenRouter, defaultOpenRouterBaseURL, creds), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider: %s", provider)
	}
}

func resolveBaseURL(configured, fallback string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(configured), "/")
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func resolveMaxTokens(n int) int {
	if n <= 0 {
		return defaultMaxTokens
	}
	return n
}
