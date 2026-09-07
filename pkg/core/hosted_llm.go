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

// DefaultHostedLLMModel is the instance model Run SuperPlane Agent uses.
type DefaultHostedLLMModel struct {
	Provider string
	Model    string
}

func (d DefaultHostedLLMModel) IsSet() bool {
	return strings.TrimSpace(d.Provider) != "" && strings.TrimSpace(d.Model) != ""
}

// HostedLLMContext resolves installation-hosted provider credentials.
type HostedLLMContext interface {
	Resolve(provider string) (HostedLLMAccess, error)
	AssertCreditAvailable() error
	AssertModelSelectable(provider, fundingSource, model string) error
	DefaultModel() (DefaultHostedLLMModel, error)
}
