package common

import (
	"slices"
	"strings"
)

type WebhookConfiguration struct {
	EventType  string   `json:"eventType"`
	EventTypes []string `json:"eventTypes"` // Multiple event types (takes precedence over EventType if set)
	Repository string   `json:"repository"`
	// Repositories lists every repository that delivers to this webhook.
	// It takes precedence over Repository when set.
	Repositories []string `json:"repositories,omitempty" mapstructure:"repositories"`
}

// WebhookRepositories returns the repositories of the webhook.
func (c WebhookConfiguration) WebhookRepositories() []string {
	if len(c.Repositories) > 0 {
		return c.Repositories
	}
	if strings.TrimSpace(c.Repository) == "" {
		return nil
	}
	return []string{c.Repository}
}

// SameRepositories reports whether both configurations deliver from the
// same repositories. Order and case do not matter.
func (c WebhookConfiguration) SameRepositories(other WebhookConfiguration) bool {
	if (len(c.Repositories) > 0) != (len(other.Repositories) > 0) {
		return false
	}
	return slices.Equal(normalizedRepositories(c.WebhookRepositories()), normalizedRepositories(other.WebhookRepositories()))
}

func normalizedRepositories(repositories []string) []string {
	normalized := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		repository = strings.ToLower(strings.TrimSpace(repository))
		if repository == "" || slices.Contains(normalized, repository) {
			continue
		}
		normalized = append(normalized, repository)
	}
	slices.Sort(normalized)
	return normalized
}
