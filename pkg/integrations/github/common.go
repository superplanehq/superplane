package github

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type Repository struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type NodeMetadata struct {
	Repository *Repository `json:"repository"`
}

func ensureRepoInMetadata(ctx core.MetadataContext, app core.IntegrationContext, configuration any) error {
	var nodeMetadata NodeMetadata
	err := mapstructure.Decode(ctx.Get(), &nodeMetadata)
	if err != nil {
		return fmt.Errorf("failed to decode node metadata: %w", err)
	}

	repository := getRepositoryFromConfiguration(configuration)
	if repository == "" {
		return fmt.Errorf("repository is required")
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

	if repoIndex == -1 {
		return fmt.Errorf("repository %s is not accessible to app installation", repository)
	}

	if nodeMetadata.Repository != nil && nodeMetadata.Repository.Name == repository {
		return nil
	}

	return ctx.Set(NodeMetadata{
		Repository: &appMetadata.Repositories[repoIndex],
	})
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

func sanitizeAssignees(assignees []string) []string {
	result := make([]string, len(assignees))
	for i, a := range assignees {
		result[i] = strings.TrimPrefix(a, "@")
	}

	return result
}

func buildPRContinuationKey(data map[string]any) (string, bool) {
	repoFullName, ok := extractRepositoryFullName(data)
	if !ok || repoFullName == "" {
		return "", false
	}

	prNumber, ok := extractPRNumber(data)
	if !ok {
		return "", false
	}

	return fmt.Sprintf("github:%s:pr:%d", repoFullName, prNumber), true
}

func extractRepositoryFullName(data map[string]any) (string, bool) {
	repository, ok := data["repository"].(map[string]any)
	if !ok {
		return "", false
	}

	if fullName, ok := repository["full_name"].(string); ok && fullName != "" {
		return fullName, true
	}

	name, nameOK := repository["name"].(string)
	if !nameOK || name == "" {
		return "", false
	}

	owner, ownerOK := repository["owner"].(map[string]any)
	if !ownerOK {
		return "", false
	}

	login, loginOK := owner["login"].(string)
	if !loginOK || login == "" {
		return "", false
	}

	return fmt.Sprintf("%s/%s", login, name), true
}

func extractPRNumber(data map[string]any) (int64, bool) {
	// issue_comment events for PR comments expose issue.number
	if issue, ok := data["issue"].(map[string]any); ok {
		if number, ok := parseNumericID(issue["number"]); ok {
			return number, true
		}
	}

	// pull_request and review events expose pull_request.number
	if pr, ok := data["pull_request"].(map[string]any); ok {
		if number, ok := parseNumericID(pr["number"]); ok {
			return number, true
		}
	}

	return 0, false
}

func parseNumericID(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func verifySignature(ctx core.WebhookRequestContext) (int, error) {
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

func fetchReleaseByStrategy(client *github.Client, owner, repo, strategy, tagName string) (*github.RepositoryRelease, error) {
	switch strategy {
	case "specific":
		// Fetch by specific tag name
		release, _, err := client.Repositories.GetReleaseByTag(
			context.Background(),
			owner,
			repo,
			tagName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to find release with tag %s: %w", tagName, err)
		}
		return release, nil

	case "latest":
		// Fetch latest published release
		release, _, err := client.Repositories.GetLatestRelease(
			context.Background(),
			owner,
			repo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest release: %w", err)
		}
		return release, nil

	case "latestDraft":
		// List releases and find the latest draft
		releases, _, err := client.Repositories.ListReleases(
			context.Background(),
			owner,
			repo,
			&github.ListOptions{PerPage: 100},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		for _, release := range releases {
			if release.GetDraft() {
				return release, nil
			}
		}
		return nil, fmt.Errorf("no draft releases found")

	case "latestPrerelease":
		// List releases and find the latest prerelease
		releases, _, err := client.Repositories.ListReleases(
			context.Background(),
			owner,
			repo,
			&github.ListOptions{PerPage: 100},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to list releases: %w", err)
		}

		for _, release := range releases {
			if release.GetPrerelease() && !release.GetDraft() {
				return release, nil
			}
		}
		return nil, fmt.Errorf("no prerelease releases found")

	default:
		return nil, fmt.Errorf("invalid release strategy: %s", strategy)
	}
}
