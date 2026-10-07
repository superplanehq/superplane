package config

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

const (
	EnvDevHostedOpenRouter        = "SUPERPLANE_DEV_HOSTED_OPENROUTER"
	EnvDevOpenRouterAPIKey        = "SUPERPLANE_DEV_OPENROUTER_API_KEY"
	EnvDevOpenRouterManagementKey = "SUPERPLANE_DEV_OPENROUTER_MANAGEMENT_KEY"
	EnvDevOpenRouterModels        = "SUPERPLANE_DEV_OPENROUTER_MODELS"
	EnvDevOpenRouterBaseURL       = "SUPERPLANE_DEV_OPENROUTER_BASE_URL"
	EnvDevHostedDefaultModel      = "SUPERPLANE_DEV_HOSTED_DEFAULT_MODEL"
)

// DevHostedOpenRouterConfig is the opt-in local seed for SuperPlane-hosted
// OpenRouter. Production and other developers leave these empty.
type DevHostedOpenRouterConfig struct {
	APIKey        string
	ManagementKey string
	Models        []string
	BaseURL       string
	DefaultModel  string
}

// Enabled reports whether the local seed has every required value.
func (c DevHostedOpenRouterConfig) Enabled() bool {
	return c.APIKey != "" && c.ManagementKey != "" && len(c.Models) > 0
}

// LoadDevHostedOpenRouterConfig reads the local OpenRouter seed. It is a
// no-op unless APP_ENV is development and SUPERPLANE_DEV_HOSTED_OPENROUTER
// is yes. A half-filled flag returns an error.
func LoadDevHostedOpenRouterConfig() (DevHostedOpenRouterConfig, error) {
	if os.Getenv("APP_ENV") != "development" {
		return DevHostedOpenRouterConfig{}, nil
	}
	if os.Getenv(EnvDevHostedOpenRouter) != "yes" {
		return DevHostedOpenRouterConfig{}, nil
	}

	apiKey := strings.TrimSpace(os.Getenv(EnvDevOpenRouterAPIKey))
	managementKey := strings.TrimSpace(os.Getenv(EnvDevOpenRouterManagementKey))
	models, err := parseDevOpenRouterModels(os.Getenv(EnvDevOpenRouterModels))
	if err != nil {
		return DevHostedOpenRouterConfig{}, err
	}
	if apiKey == "" || managementKey == "" || len(models) == 0 {
		return DevHostedOpenRouterConfig{}, fmt.Errorf(
			"%s=yes requires %s, %s, and %s",
			EnvDevHostedOpenRouter,
			EnvDevOpenRouterAPIKey,
			EnvDevOpenRouterManagementKey,
			EnvDevOpenRouterModels,
		)
	}

	defaultModel := strings.TrimSpace(os.Getenv(EnvDevHostedDefaultModel))
	if defaultModel != "" && !slices.Contains(models, defaultModel) {
		return DevHostedOpenRouterConfig{}, fmt.Errorf(
			"%s must be in %s",
			EnvDevHostedDefaultModel,
			EnvDevOpenRouterModels,
		)
	}

	return DevHostedOpenRouterConfig{
		APIKey:        apiKey,
		ManagementKey: managementKey,
		Models:        models,
		BaseURL:       strings.TrimSpace(os.Getenv(EnvDevOpenRouterBaseURL)),
		DefaultModel:  defaultModel,
	}, nil
}

func parseDevOpenRouterModels(raw string) ([]string, error) {
	seen := map[string]struct{}{}
	var models []string
	for _, part := range strings.Split(raw, ",") {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("duplicate allowed model: %s", id)
		}
		seen[id] = struct{}{}
		models = append(models, id)
	}
	return models, nil
}
