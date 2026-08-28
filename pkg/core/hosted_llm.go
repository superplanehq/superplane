package core

import (
	"slices"
	"strings"
)

// HostedLLMAccess is one SuperPlane-held provider key and allowlist.
type HostedLLMAccess struct {
	APIKey        string
	BaseURL       string
	AllowedModels []string
}

func (a HostedLLMAccess) AllowsModel(model string) bool {
	normalized := strings.TrimSpace(model)
	if normalized == "" {
		return false
	}
	return slices.Contains(a.AllowedModels, normalized)
}

// HostedLLMContext resolves installation-hosted provider credentials.
type HostedLLMContext interface {
	Resolve(provider string) (HostedLLMAccess, error)
	AssertCreditAvailable() error
	AssertModelSelectable(provider, fundingSource, model string) error
}
