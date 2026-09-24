package models

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	SelectableLLMSourceHostedName = "SuperPlane"
	SelectableLLMSourceBYOKName   = "Your keys"
)

var (
	ErrSelectableLLMModelIncomplete = errors.New("selectable model key is incomplete")
	ErrSelectableLLMModelNotAllowed = errors.New("this workspace does not allow the selected model")
)

type SelectableLLMNamedID struct {
	ID   string
	Name string
}

type SelectableLLMModel struct {
	Source   SelectableLLMNamedID
	Provider SelectableLLMNamedID
	Model    SelectableLLMNamedID
	Key      string
	Label    string
}

func SelectableLLMSourceName(source string) string {
	switch strings.TrimSpace(source) {
	case UsageFundingSourceHosted:
		return SelectableLLMSourceHostedName
	case UsageFundingSourceBYOK:
		return SelectableLLMSourceBYOKName
	default:
		return strings.TrimSpace(source)
	}
}

func SelectableLLMProviderName(provider string) string {
	switch strings.TrimSpace(provider) {
	case UsageProviderAnthropic:
		return "Anthropic"
	case UsageProviderOpenAI:
		return "OpenAI"
	case UsageProviderOpenRouter:
		return "OpenRouter"
	default:
		return strings.TrimSpace(provider)
	}
}

func HostedLLMTechnicalName(provider, model string) string {
	trimmedProvider := strings.TrimSpace(provider)
	trimmedModel := strings.TrimSpace(model)
	if trimmedProvider == "" || trimmedModel == "" {
		return trimmedModel
	}
	if trimmedProvider == UsageProviderOpenRouter {
		return trimmedModel
	}
	return trimmedProvider + "/" + trimmedModel
}

func FormatSelectableLLMModelKey(source, provider, model string) string {
	return strings.TrimSpace(source) + hostedLLMModelKeySeparator + FormatHostedLLMModelKey(provider, model)
}

func ParseSelectableLLMModelKey(value string) (SelectableLLMModel, error) {
	trimmed := strings.TrimSpace(value)
	parts := strings.SplitN(trimmed, hostedLLMModelKeySeparator, 3)
	if len(parts) != 3 {
		return SelectableLLMModel{}, ErrSelectableLLMModelIncomplete
	}
	source, err := normalizeLLMFundingSource(parts[0])
	if err != nil {
		return SelectableLLMModel{}, ErrSelectableLLMModelIncomplete
	}
	parsed, err := NormalizeDefaultHostedLLMModel(parts[1], parts[2])
	if err != nil {
		return SelectableLLMModel{}, err
	}
	if !parsed.IsSet() {
		return SelectableLLMModel{}, ErrSelectableLLMModelIncomplete
	}
	return newSelectableLLMModel(source, parsed.Provider, parsed.Model), nil
}

func ParseHostedLLMModelKey(value string) (DefaultHostedLLMModel, error) {
	trimmed := strings.TrimSpace(value)
	parts := strings.SplitN(trimmed, hostedLLMModelKeySeparator, 3)
	if len(parts) == 3 {
		source, err := normalizeLLMFundingSource(parts[0])
		if err != nil || source != UsageFundingSourceHosted {
			return DefaultHostedLLMModel{}, ErrDefaultHostedModelIncomplete
		}
		return NormalizeDefaultHostedLLMModel(parts[1], parts[2])
	}
	if len(parts) != 2 {
		return DefaultHostedLLMModel{}, ErrDefaultHostedModelIncomplete
	}
	return NormalizeDefaultHostedLLMModel(parts[0], parts[1])
}

func newSelectableLLMModel(source, provider, model string) SelectableLLMModel {
	return SelectableLLMModel{
		Source:   SelectableLLMNamedID{ID: source, Name: SelectableLLMSourceName(source)},
		Provider: SelectableLLMNamedID{ID: provider, Name: SelectableLLMProviderName(provider)},
		Model:    SelectableLLMNamedID{ID: model, Name: model},
		Key:      FormatSelectableLLMModelKey(source, provider, model),
		Label:    HostedLLMTechnicalName(provider, model),
	}
}

// ListSelectableLLMModels returns hosted plus BYOK selected models for a picker.
func ListSelectableLLMModels(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID) ([]SelectableLLMModel, error) {
	out := make([]SelectableLLMModel, 0)
	for _, provider := range KnownHostedLLMProviders() {
		for _, source := range []string{UsageFundingSourceHosted, UsageFundingSourceBYOK} {
			ids, err := ResolveSelectableLLMModels(tx, orgID, factoryID, provider, source)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				out = append(out, newSelectableLLMModel(source, provider, id))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Label != out[j].Label {
			return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
		}
		if out[i].Source.ID != out[j].Source.ID {
			return out[i].Source.ID < out[j].Source.ID
		}
		return out[i].Key < out[j].Key
	})
	return out, nil
}

func FindSelectableLLMModel(tx *gorm.DB, orgID uuid.UUID, factoryID *uuid.UUID, key string) (SelectableLLMModel, error) {
	parsed, err := ParseSelectableLLMModelKey(key)
	if err != nil {
		return SelectableLLMModel{}, err
	}
	allowed, err := ModelIsSelectable(tx, orgID, factoryID, parsed.Provider.ID, parsed.Source.ID, parsed.Model.ID)
	if err != nil {
		return SelectableLLMModel{}, err
	}
	if !allowed {
		return SelectableLLMModel{}, ErrSelectableLLMModelNotAllowed
	}
	return parsed, nil
}

func SelectableLLMRunnerComponent(model SelectableLLMModel) (string, error) {
	switch model.Source.ID {
	case UsageFundingSourceHosted:
		return SuperPlaneRunnerComponent, nil
	case UsageFundingSourceBYOK:
		switch model.Provider.ID {
		case UsageProviderAnthropic:
			return "runnerClaudeCode", nil
		case UsageProviderOpenAI:
			return "runnerCodex", nil
		case UsageProviderOpenRouter:
			return "runnerOpenRouter", nil
		}
	}
	return "", fmt.Errorf("unsupported selectable model %s", model.Key)
}

func SelectableLLMRunnerModel(model SelectableLLMModel) string {
	if model.Source.ID == UsageFundingSourceHosted {
		return model.Key
	}
	return model.Model.ID
}

func SelectableLLMRunnerCredentials(tx *gorm.DB, orgID uuid.UUID, model SelectableLLMModel) (map[string]any, error) {
	if model.Source.ID == UsageFundingSourceHosted {
		return nil, nil
	}
	if tx == nil {
		return nil, fmt.Errorf("provider is not connected")
	}
	integration, err := FindReadyBYOKIntegration(tx, orgID, model.Provider.ID)
	if err != nil {
		return nil, err
	}
	if integration == nil {
		return nil, fmt.Errorf("provider is not connected")
	}
	return map[string]any{
		"source":      "integration",
		"integration": map[string]any{"name": integration.InstallationName},
	}, nil
}
