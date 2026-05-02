package common

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

var expressionPlaceholderRegex = regexp.MustCompile(`(?s)\{\{.*?\}\}`)

type Repository struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

func EnsureRepoInMetadata(ctx core.MetadataWriter, app core.IntegrationContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	repository := getRepositoryFromConfiguration(configuration)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}
	if expressionPlaceholderRegex.MatchString(repository) {
		return nil
	}

	//
	// Validate that the app has access to this repository
	//
	var appMetadata Metadata
	if err := mapstructure.Decode(app.GetMetadata(), &appMetadata); err != nil {
		return fmt.Errorf("failed to decode application metadata: %w", err)
	}

	repoIndex := slices.IndexFunc(appMetadata.Repositories, func(r Repository) bool {
		return r.Name == repository
	})

	if nodeMetadata.Repository != nil && nodeMetadata.Repository.Name == repository {
		return nil
	}

	// Prefer cached metadata when present (fast path).
	if repoIndex != -1 {
		return ctx.Set(NodeMetadata{
			Repository: &appMetadata.Repositories[repoIndex],
		})
	}

	// Cached metadata can be incomplete (pagination, stale cache). If we have enough information,
	// validate live against GitHub to avoid false rejections.
	if appMetadata.InstallationID == "" || appMetadata.GitHubApp.ID == 0 || appMetadata.Owner == "" {
		return fmt.Errorf("repository %s is not accessible to app installation", repository)
	}

	client, err := NewClient(app, appMetadata.GitHubApp.ID, appMetadata.InstallationID)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	repo, _, err := client.Repositories.Get(context.Background(), appMetadata.Owner, repository)
	if err != nil {
		return fmt.Errorf("repository %s is not accessible to app installation", repository)
	}

	live := Repository{
		ID:   repo.GetID(),
		Name: repo.GetName(),
		URL:  repo.GetHTMLURL(),
	}

	return ctx.Set(NodeMetadata{Repository: &live})
}

func getRepositoryFromConfiguration(c any) string {
	configMap, ok := c.(map[string]any)
	if !ok {
		return ""
	}

	r, ok := configMap["repository"]
	if !ok {
		return ""
	}

	repository, ok := r.(string)
	if !ok {
		return ""
	}

	return repository
}

func SanitizeAssignees(assignees []string) []string {
	result := make([]string, len(assignees))
	for i, a := range assignees {
		result[i] = strings.TrimPrefix(a, "@")
	}

	return result
}

func WithWebhookLogger(ctx core.WebhookRequestContext, triggerName string) core.WebhookRequestContext {
	ctx.Logger = ctx.Logger.WithFields(log.Fields{
		"gh-id":   ctx.Headers.Get("X-GitHub-Delivery"),
		"trigger": triggerName,
	})

	return ctx
}

func VerifySignature(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature-256")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error authenticating request")
	}

	err = crypto.VerifySignature(secret, ctx.Body, signature)
	if err != nil {
		return http.StatusForbidden, err
	}

	return http.StatusOK, nil
}

func WhitelistedAction(data map[string]any, allowed []string) bool {
	action, ok := ExtractAction(data)
	if !ok {
		return false
	}

	return slices.Contains(allowed, action)
}

func ExtractAction(data map[string]any) (string, bool) {
	action, ok := data["action"].(string)
	return action, ok
}
