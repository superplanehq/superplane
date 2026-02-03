package github

import (
	"context"
	"fmt"
	"net/http"
	"slices"
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

// buildReleaseData converts a GitHub release to a map for output emission
func buildReleaseData(release *github.RepositoryRelease) map[string]any {
	data := map[string]any{
		"id":         release.GetID(),
		"tag_name":   release.GetTagName(),
		"name":       release.GetName(),
		"body":       release.GetBody(),
		"html_url":   release.GetHTMLURL(),
		"draft":      release.GetDraft(),
		"prerelease": release.GetPrerelease(),
	}

	if release.CreatedAt != nil {
		data["created_at"] = release.CreatedAt.Format("2006-01-02T15:04:05Z")
	}

	if release.PublishedAt != nil {
		data["published_at"] = release.PublishedAt.Format("2006-01-02T15:04:05Z")
	}

	if release.Author != nil {
		data["author"] = map[string]any{
			"login":      release.Author.GetLogin(),
			"id":         release.Author.GetID(),
			"avatar_url": release.Author.GetAvatarURL(),
			"html_url":   release.Author.GetHTMLURL(),
		}
	}

	if len(release.Assets) > 0 {
		assets := make([]map[string]any, len(release.Assets))
		for i, asset := range release.Assets {
			assets[i] = map[string]any{
				"id":                 asset.GetID(),
				"name":               asset.GetName(),
				"label":              asset.GetLabel(),
				"content_type":       asset.GetContentType(),
				"size":               asset.GetSize(),
				"download_count":     asset.GetDownloadCount(),
				"browser_download_url": asset.GetBrowserDownloadURL(),
			}
		}
		data["assets"] = assets
	}

	return data
}
