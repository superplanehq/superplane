package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	EventType  string   `json:"eventType"`
	EventTypes []string `json:"eventTypes"` // Multiple event types (takes precedence over EventType if set)
	Repository string   `json:"repository"`
}

type Webhook struct {
	ID          int64  `json:"id"`
	WebhookName string `json:"name"`
}

type GitHubWebhookHandler struct{}

func (h *GitHubWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := WebhookConfiguration{}
	configB := WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	if configA.Repository != configB.Repository {
		return false, nil
	}

	// Compare event types - normalize to slices for comparison
	eventsA := configA.EventTypes
	if len(eventsA) == 0 && configA.EventType != "" {
		eventsA = []string{configA.EventType}
	}

	eventsB := configB.EventTypes
	if len(eventsB) == 0 && configB.EventType != "" {
		eventsB = []string{configB.EventType}
	}

	if len(eventsA) != len(eventsB) {
		return false, nil
	}

	// Create a map to compare events regardless of order
	eventMap := make(map[string]bool)
	for _, e := range eventsA {
		eventMap[e] = true
	}
	for _, e := range eventsB {
		if !eventMap[e] {
			return false, nil
		}
	}

	return true, nil
}

func (h *GitHubWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *GitHubWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return nil, err
	}

	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return nil, err
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, err
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %v", err)
	}

	// Use EventTypes if set, otherwise fall back to single EventType
	events := config.EventTypes
	if len(events) == 0 && config.EventType != "" {
		events = []string{config.EventType}
	}

	hook := &github.Hook{
		Active: github.Ptr(true),
		Events: events,
		Config: &github.HookConfig{
			URL:         github.Ptr(ctx.Webhook.GetURL()),
			Secret:      github.Ptr(string(secret)),
			ContentType: github.Ptr("json"),
		},
	}

	createdHook, _, err := client.Repositories.CreateHook(context.Background(), metadata.Owner, config.Repository, hook)
	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %v", err)
	}

	return &Webhook{ID: createdHook.GetID(), WebhookName: *createdHook.Name}, nil
}

func (h *GitHubWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.Integration, metadata.GitHubApp.ID, metadata.InstallationID)
	if err != nil {
		return err
	}

	webhook := Webhook{}
	err = mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook)
	if err != nil {
		return err
	}

	configuration := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return err
	}

	_, err = client.Repositories.DeleteHook(context.Background(), metadata.Owner, configuration.Repository, webhook.ID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}
