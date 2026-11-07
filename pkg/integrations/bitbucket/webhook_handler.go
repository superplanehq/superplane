package bitbucket

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	EventTypes     []string `json:"eventTypes"`
	RepositorySlug string   `json:"repositorySlug"`
	// Repositories lists the full names of every repository that delivers
	// to this webhook. It takes precedence over RepositorySlug when set.
	Repositories []string `json:"repositories,omitempty" mapstructure:"repositories"`
}

// sameRepositories reports whether both configurations deliver from the
// same repositories. Order and case do not matter.
func (c WebhookConfiguration) sameRepositories(other WebhookConfiguration) bool {
	if len(c.Repositories) == 0 && len(other.Repositories) == 0 {
		return c.RepositorySlug == other.RepositorySlug
	}
	return slices.Equal(normalizedRepositories(c.Repositories), normalizedRepositories(other.Repositories))
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

type BitbucketWebhook struct {
	UUID               string `json:"uuid"`
	RepositoryUUID     string `json:"repositoryUUID,omitempty"`
	RepositoryFullName string `json:"repositoryFullName,omitempty"`
	// Hooks has one entry per repository for a multi-repository webhook.
	Hooks []RepositoryWebhook `json:"hooks,omitempty" mapstructure:"hooks"`
}

type RepositoryWebhook struct {
	Repository         string `json:"repository" mapstructure:"repository"`
	UUID               string `json:"uuid,omitempty" mapstructure:"uuid"`
	RepositoryUUID     string `json:"repositoryUUID,omitempty" mapstructure:"repositoryUUID"`
	RepositoryFullName string `json:"repositoryFullName,omitempty" mapstructure:"repositoryFullName"`
}

// RepositoryFullNameFor returns the full name of the Forge repository with
// the given UUID.
func (w BitbucketWebhook) RepositoryFullNameFor(repositoryUUID string) string {
	repositoryUUID = strings.Trim(repositoryUUID, "{}")
	for _, hook := range w.Hooks {
		if strings.EqualFold(strings.Trim(hook.RepositoryUUID, "{}"), repositoryUUID) {
			return hook.RepositoryFullName
		}
	}
	return w.RepositoryFullName
}

type BitbucketWebhookHandler struct{}

func (h *BitbucketWebhookHandler) CompareConfig(a, b any) (bool, error) {
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

	if !configA.sameRepositories(configB) {
		return false, nil
	}

	// Check if A contains all events from B (A is superset of B)
	// This allows webhook sharing when existing webhook has more events than needed
	for _, eventB := range configB.EventTypes {
		if !slices.Contains(configA.EventTypes, eventB) {
			return false, nil
		}
	}

	return true, nil
}

func (h *BitbucketWebhookHandler) Merge(current, requested any) (any, bool, error) {
	currentConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(current, &currentConfig); err != nil {
		return nil, false, err
	}
	requestedConfig := WebhookConfiguration{}
	if err := mapstructure.Decode(requested, &requestedConfig); err != nil {
		return nil, false, err
	}

	// Webhooks are per repository. A different repository never merges.
	if !currentConfig.sameRepositories(requestedConfig) {
		return current, false, nil
	}

	// ponytail: union event types so one repository webhook serves every
	// trigger; re-provision updates that remote hook with the union
	merged := WebhookConfiguration{
		EventTypes:     slices.Clone(currentConfig.EventTypes),
		RepositorySlug: currentConfig.RepositorySlug,
		Repositories:   slices.Clone(currentConfig.Repositories),
	}
	changed := false
	for _, event := range requestedConfig.EventTypes {
		if !slices.Contains(merged.EventTypes, event) {
			merged.EventTypes = append(merged.EventTypes, event)
			changed = true
		}
	}
	return merged, changed, nil
}

func (h *BitbucketWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if config.RepositorySlug == "" {
		return nil, fmt.Errorf("repository is required")
	}

	if metadata.AuthType == AuthTypeForgeApp {
		return setupForgeWebhook(client, metadata, config)
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	existing := BitbucketWebhook{}
	if raw := ctx.Webhook.GetMetadata(); raw != nil {
		if err := mapstructure.Decode(raw, &existing); err != nil {
			return nil, fmt.Errorf("failed to decode webhook metadata: %w", err)
		}
	}

	if len(config.Repositories) > 0 {
		return setupRepositoryWebhooks(client, config, existing, ctx.Webhook.GetURL(), string(secret))
	}

	uuid, _, err := saveRepositoryWebhook(client, metadata.Workspace.Slug, config.RepositorySlug, existing.UUID, ctx.Webhook.GetURL(), string(secret), config.EventTypes)
	if err != nil {
		return nil, err
	}
	return &BitbucketWebhook{UUID: uuid}, nil
}

func setupForgeWebhook(client *Client, metadata Metadata, config WebhookConfiguration) (*BitbucketWebhook, error) {
	for _, event := range config.EventTypes {
		switch event {
		case pullRequestEventCreated, pullRequestEventUpdated, pullRequestEventFulfilled, pullRequestEventRejected,
			waitBuildsStatusCreated, waitBuildsStatusUpdated, pullRequestCommentCreated:
		default:
			return nil, fmt.Errorf("Bitbucket Forge does not support event %q", event)
		}
	}

	repositories := config.Repositories
	if len(repositories) == 0 {
		repositories = []string{metadata.Workspace.Slug + "/" + config.RepositorySlug}
	}
	hooks := make([]RepositoryWebhook, 0, len(repositories))
	for _, repository := range repositories {
		repo, err := client.GetRepository(repository)
		if err != nil {
			return nil, fmt.Errorf("error resolving Forge webhook repository: %w", err)
		}
		if repo.UUID == "" || repo.FullName == "" {
			return nil, fmt.Errorf("missing Forge webhook repository identity")
		}
		hooks = append(hooks, RepositoryWebhook{Repository: repository, RepositoryUUID: repo.UUID, RepositoryFullName: repo.FullName})
	}

	webhook := &BitbucketWebhook{RepositoryUUID: hooks[0].RepositoryUUID, RepositoryFullName: hooks[0].RepositoryFullName}
	if len(config.Repositories) > 0 {
		webhook.Hooks = hooks
	}
	return webhook, nil
}

// setupRepositoryWebhooks saves the same hook in each repository. When a
// repository fails, it removes the hooks it created.
func setupRepositoryWebhooks(client *Client, config WebhookConfiguration, existing BitbucketWebhook, url, secret string) (*BitbucketWebhook, error) {
	hooks := make([]RepositoryWebhook, 0, len(config.Repositories))
	created := []RepositoryWebhook{}
	for _, repository := range config.Repositories {
		workspace, slug, ok := strings.Cut(strings.TrimSpace(repository), "/")
		if !ok || workspace == "" || slug == "" {
			_ = deleteRepositoryWebhooks(client, created)
			return nil, fmt.Errorf("repository must be in workspace/repository format: %q", repository)
		}

		uuid, isNew, err := saveRepositoryWebhook(client, workspace, slug, existing.hookUUID(repository), url, secret, config.EventTypes)
		if err != nil {
			_ = deleteRepositoryWebhooks(client, created)
			return nil, fmt.Errorf("%s: %w", repository, err)
		}

		hook := RepositoryWebhook{Repository: repository, UUID: uuid}
		if isNew {
			created = append(created, hook)
		}
		hooks = append(hooks, hook)
	}

	return &BitbucketWebhook{UUID: hooks[0].UUID, Hooks: hooks}, nil
}

func (w BitbucketWebhook) hookUUID(repository string) string {
	for _, hook := range w.Hooks {
		if strings.EqualFold(hook.Repository, repository) {
			return hook.UUID
		}
	}
	return ""
}

// saveRepositoryWebhook updates the existing hook, or creates a hook when
// none exists. It reports whether it created the hook.
func saveRepositoryWebhook(client *Client, workspace, slug, existingUUID, url, secret string, events []string) (string, bool, error) {
	if existingUUID != "" {
		hook, err := client.UpdateWebhook(workspace, slug, existingUUID, url, secret, events)
		if err == nil {
			if hook.UUID == "" {
				return existingUUID, false, nil
			}
			return hook.UUID, false, nil
		}
		if !errors.Is(err, errBitbucketWebhookNotFound) {
			return "", false, fmt.Errorf("error updating webhook: %w", err)
		}
	}

	hook, err := client.CreateWebhook(workspace, slug, url, secret, events)
	if err != nil {
		return "", false, fmt.Errorf("error creating webhook: %w", err)
	}
	return hook.UUID, true, nil
}

// deleteRepositoryWebhooks tries every hook, because the caller deletes the
// saved hook IDs even when cleanup fails. A hook that is already gone counts
// as deleted.
func deleteRepositoryWebhooks(client *Client, hooks []RepositoryWebhook) error {
	var errs []error
	for _, hook := range hooks {
		workspace, slug, _ := strings.Cut(hook.Repository, "/")
		err := client.DeleteWebhook(workspace, slug, hook.UUID)
		if err != nil && !errors.Is(err, errBitbucketWebhookNotFound) {
			errs = append(errs, fmt.Errorf("error deleting webhook in %s: %w", hook.Repository, err))
		}
	}
	return errors.Join(errs...)
}

func (h *BitbucketWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}
	if metadata.AuthType == AuthTypeForgeApp {
		return nil
	}

	client, err := NewClient(metadata.AuthType, ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	webhook := BitbucketWebhook{}
	err = mapstructure.Decode(ctx.Webhook.GetMetadata(), &webhook)
	if err != nil {
		return fmt.Errorf("failed to decode webhook metadata: %w", err)
	}

	// If the webhook was never created (Setup failed), there's nothing to clean up.
	if webhook.UUID == "" {
		return nil
	}

	if len(webhook.Hooks) > 0 {
		return deleteRepositoryWebhooks(client, webhook.Hooks)
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	err = client.DeleteWebhook(metadata.Workspace.Slug, config.RepositorySlug, webhook.UUID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %w", err)
	}

	return nil
}
