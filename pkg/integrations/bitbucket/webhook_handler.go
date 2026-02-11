package bitbucket

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

type WebhookConfiguration struct {
	EventType  string `json:"eventType"`
	Repository string `json:"repository"`
}

type BitbucketWebhook struct {
	UUID string `json:"uuid"`
}

type BitbucketWebhookHandler struct{}

// repoSlug extracts the repo slug from a full repository name.
// e.g. "superplane/test" -> "test", "test" -> "test"
func repoSlug(fullName string) string {
	parts := strings.Split(fullName, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return fullName
}

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

	if configA.Repository != configB.Repository {
		return false, nil
	}

	if configA.EventType != configB.EventType {
		return false, nil
	}

	return true, nil
}

func (h *BitbucketWebhookHandler) Merge(current, requested any) (any, bool, error) {
	return current, false, nil
}

func (h *BitbucketWebhookHandler) Setup(ctx core.WebhookHandlerContext) (any, error) {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	if config.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	workspace := metadata.Workspace
	if workspace == "" {
		workspaceBytes, err := ctx.Integration.GetConfig("workspace")
		if err != nil {
			return nil, fmt.Errorf("failed to get workspace config: %w", err)
		}
		workspace = string(workspaceBytes)
	}
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	repoSlugValue, err := resolveRepositorySlug(metadata.Repositories, config.Repository)
	if err != nil {
		return nil, err
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}

	hook, err := client.CreateWebhook(
		workspace,
		repoSlugValue,
		ctx.Webhook.GetURL(),
		string(secret),
		[]string{config.EventType},
	)

	if err != nil {
		return nil, fmt.Errorf("error creating webhook: %w", err)
	}

	return &BitbucketWebhook{UUID: hook.UUID}, nil
}

func (h *BitbucketWebhookHandler) Cleanup(ctx core.WebhookHandlerContext) error {
	metadata := Metadata{}
	err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
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

	config := WebhookConfiguration{}
	err = mapstructure.Decode(ctx.Webhook.GetConfiguration(), &config)
	if err != nil {
		return fmt.Errorf("failed to decode webhook configuration: %w", err)
	}

	workspace := metadata.Workspace
	if workspace == "" {
		workspaceBytes, err := ctx.Integration.GetConfig("workspace")
		if err != nil {
			return fmt.Errorf("failed to get workspace config: %w", err)
		}
		workspace = string(workspaceBytes)
	}
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	repoSlugValue, err := resolveRepositorySlug(metadata.Repositories, config.Repository)
	if err != nil {
		return err
	}

	err = client.DeleteWebhook(workspace, repoSlugValue, webhook.UUID)
	if err != nil {
		return fmt.Errorf("error deleting webhook: %w", err)
	}

	return nil
}

func resolveRepositorySlug(repositories []Repository, repository string) (string, error) {
	if repository == "" {
		return "", fmt.Errorf("repository is required")
	}

	repoIndex := slices.IndexFunc(repositories, func(r Repository) bool {
		return repositoryMatches(r, repository)
	})
	if repoIndex == -1 {
		return "", fmt.Errorf("repository %s is not accessible to workspace", repository)
	}

	repo := repositories[repoIndex]
	if repo.Slug != "" {
		return repo.Slug, nil
	}
	if repo.FullName != "" {
		return repoSlug(repo.FullName), nil
	}
	if repo.Name != "" {
		return repo.Name, nil
	}

	return repoSlug(repository), nil
}
