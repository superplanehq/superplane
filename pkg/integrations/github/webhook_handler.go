package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type Webhook struct {
	ID          int64  `json:"id"`
	WebhookName string `json:"name"`
	// Hooks has one entry per repository for a multi-repository webhook.
	Hooks []RepositoryWebhook `json:"hooks,omitempty" mapstructure:"hooks"`
}

type RepositoryWebhook struct {
	Repository string `json:"repository" mapstructure:"repository"`
	ID         int64  `json:"id" mapstructure:"id"`
	Name       string `json:"name" mapstructure:"name"`
}

type GitHubWebhookHandler struct{}

func (h *GitHubWebhookHandler) CompareConfig(a, b any) (bool, error) {
	configA := common.WebhookConfiguration{}
	configB := common.WebhookConfiguration{}

	err := mapstructure.Decode(a, &configA)
	if err != nil {
		return false, err
	}

	err = mapstructure.Decode(b, &configB)
	if err != nil {
		return false, err
	}

	if len(configA.Repositories) == 0 && len(configB.Repositories) == 0 && configA.Repository != configB.Repository {
		return false, nil
	}
	if !configA.SameRepositories(configB) {
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
	config := common.WebhookConfiguration{}
	err := mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
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

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return nil, err
	}

	if len(config.Repositories) == 0 {
		webhook := Webhook{}
		if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook); err != nil {
			return nil, fmt.Errorf("error decoding webhook metadata: %w", err)
		}
		if webhook.ID > 0 {
			updatedHook, _, err := client.EditHook(context.Background(), config.Repository, webhook.ID, hook)
			if err == nil {
				return &Webhook{ID: updatedHook.GetID(), WebhookName: updatedHook.GetName()}, nil
			}
			if !common.IsNotFoundError(err) {
				return nil, fmt.Errorf("error updating webhook: %w", err)
			}
		}

		newHook, _, err := client.CreateHook(context.Background(), config.Repository, hook)
		if err != nil {
			return nil, fmt.Errorf("error creating webhook: %v", err)
		}
		return &Webhook{ID: newHook.GetID(), WebhookName: *newHook.Name}, nil
	}

	return setupRepositoryWebhooks(client, config.Repositories, hook)
}

// setupRepositoryWebhooks creates the same hook in each repository. When a
// repository fails, it removes the hooks it already created.
func setupRepositoryWebhooks(client *common.Client, repositories []string, hook *github.Hook) (*Webhook, error) {
	hooks := make([]RepositoryWebhook, 0, len(repositories))
	for _, repository := range repositories {
		newHook, _, err := client.CreateHook(context.Background(), repository, hook)
		if err != nil {
			_ = deleteRepositoryWebhooks(client, hooks)
			return nil, fmt.Errorf("error creating webhook in %s: %v", repository, err)
		}
		hooks = append(hooks, RepositoryWebhook{Repository: repository, ID: newHook.GetID(), Name: newHook.GetName()})
	}

	return &Webhook{ID: hooks[0].ID, WebhookName: hooks[0].Name, Hooks: hooks}, nil
}

func deleteRepositoryWebhooks(client *common.Client, hooks []RepositoryWebhook) error {
	for _, hook := range hooks {
		_, err := client.DeleteHook(context.Background(), hook.Repository, hook.ID)
		if err != nil && !common.IsNotFoundError(err) {
			return fmt.Errorf("error deleting webhook in %s: %v", hook.Repository, err)
		}
	}
	return nil
}

func (h *GitHubWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return err
	}

	webhook := Webhook{}
	err = mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook)
	if err != nil {
		return err
	}

	if len(webhook.Hooks) > 0 {
		return deleteRepositoryWebhooks(client, webhook.Hooks)
	}

	configuration := common.WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &configuration)
	if err != nil {
		return err
	}

	_, err = client.DeleteHook(context.Background(), configuration.Repository, webhook.ID)
	if err != nil {
		if common.IsNotFoundError(err) {
			return nil
		}

		return fmt.Errorf("error deleting webhook: %v", err)
	}

	return nil
}
