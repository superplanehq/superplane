package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Claude family names the CLI accepts in place of a versioned model id.
const (
	claudeAliasHaiku  = "haiku"
	claudeAliasOpus   = "opus"
	claudeAliasSonnet = "sonnet"

	claudeHaikuFallback  = "claude-haiku-4-5"
	claudeOpusFallback   = "claude-opus-5-5"
	claudeSonnetFallback = "claude-sonnet-4-6"
)

// IsClaudeFamilyAlias reports whether stored is haiku, opus, or sonnet,
// with an optional single provider prefix such as "anthropic/opus".
func IsClaudeFamilyAlias(stored string) bool {
	_, leaf := modelProviderAndLeaf(strings.TrimSpace(stored))
	return claudeAliasFallbackID(leaf) != ""
}

// ConcreteClaudeModelID replaces a Claude family alias with a versioned id.
// A versioned id is returned unchanged. The newest candidate in that family
// wins. With no candidate, a fixed versioned id is used so the alias is not
// returned.
func ConcreteClaudeModelID(stored string, candidates []string) string {
	trimmed := strings.TrimSpace(stored)
	if trimmed == "" || strings.Contains(trimmed, "::") {
		return trimmed
	}
	provider, leaf := modelProviderAndLeaf(trimmed)
	fallback := claudeAliasFallbackID(leaf)
	if fallback == "" {
		return trimmed
	}
	if match := firstVersionedModelID(candidates, leaf); match != "" {
		return joinModelProvider(provider, canonicalSpendingModelName(match))
	}
	return joinModelProvider(provider, fallback)
}

// ClaudeAliasCandidateIDs lists selectable model ids that can replace an alias.
func ClaudeAliasCandidateIDs(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID) ([]string, error) {
	if tx == nil || orgID == uuid.Nil {
		return nil, nil
	}
	providers := []string{UsageProviderAnthropic, UsageProviderOpenAI, UsageProviderOpenRouter}
	sources := []string{UsageFundingSourceBYOK, UsageFundingSourceHosted}
	ids := make([]string, 0)
	for _, provider := range providers {
		for _, source := range sources {
			models, err := ResolveSelectableLLMModels(tx, orgID, factoryID, provider, source)
			if err != nil {
				return nil, err
			}
			ids = append(ids, models...)
		}
	}
	return CompactModelIDs(ids), nil
}

func claudeAliasFallbackID(leaf string) string {
	switch strings.ToLower(strings.TrimSpace(leaf)) {
	case claudeAliasHaiku:
		return claudeHaikuFallback
	case claudeAliasOpus:
		return claudeOpusFallback
	case claudeAliasSonnet:
		return claudeSonnetFallback
	default:
		return ""
	}
}

func modelProviderAndLeaf(stored string) (string, string) {
	provider, leaf, found := strings.Cut(stored, "/")
	if !found || leaf == "" || strings.Contains(leaf, "/") || strings.Contains(provider, "::") {
		return "", stored
	}
	return provider, leaf
}

func joinModelProvider(provider, model string) string {
	if provider == "" || strings.Contains(model, "/") {
		return model
	}
	return provider + "/" + model
}
