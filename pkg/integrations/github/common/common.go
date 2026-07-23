package common

import (
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

// IsExpression reports whether the given string contains an expression
// placeholder (e.g. `{{ ... }}`). Useful in Setup paths to skip strict
// validation of values that will only be known at execution time.
func IsExpression(s string) bool {
	return expressionPlaceholderRegex.MatchString(s)
}

type Repository struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

func EnsureRepoInMetadata(ctx core.MetadataWriter, integration core.IntegrationContext, httpCtx core.HTTPContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	repository := getRepositoryFromConfiguration(configuration)
	if repository == "" {
		return fmt.Errorf("repository is required")
	}
	if IsExpression(repository) {
		return nil
	}

	if nodeMetadata.Repository != nil && repositoryRefersTo(nodeMetadata.Repository.Name, repository) {
		return nil
	}

	client, err := NewClient(integration, httpCtx)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	repo, err := client.FindRepository(repository)
	if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	return ctx.Set(NodeMetadata{Repository: &Repository{
		ID:   repo.GetID(),
		Name: repo.GetName(),
		URL:  repo.GetHTMLURL(),
	}})
}

// repositoryRefersTo reports whether configured (short name or owner/repo)
// identifies the same repository as storedName (typically the short name).
func repositoryRefersTo(storedName, configured string) bool {
	if storedName == "" || configured == "" {
		return false
	}
	if storedName == configured {
		return true
	}
	if i := strings.LastIndex(configured, "/"); i >= 0 {
		return storedName == configured[i+1:]
	}
	return false
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
