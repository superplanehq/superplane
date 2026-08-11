package bitbucket

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
)

const (
	ResourceTypeRepository = "repository"
	ResourceTypeMember     = "member"
)

// Build states accepted by the Bitbucket commit status API.
const (
	StateInProgress = "INPROGRESS"
	StateSuccessful = "SUCCESSFUL"
	StateFailed     = "FAILED"
	StateStopped    = "STOPPED"
)

// Merge strategies accepted by the Bitbucket pull request merge API.
const (
	MergeStrategyMergeCommit = "merge_commit"
	MergeStrategySquash      = "squash"
	MergeStrategyFastForward = "fast_forward"
)

type NodeMetadata struct {
	Repository *RepositoryMetadata `json:"repository" mapstructure:"repository"`
}

type RepositoryMetadata struct {
	UUID     string `json:"uuid" mapstructure:"uuid"`
	Name     string `json:"name" mapstructure:"name"`
	FullName string `json:"full_name" mapstructure:"full_name"`
	Slug     string `json:"slug" mapstructure:"slug"`
}

func ensureRepoInMetadata(http core.HTTPContext, ctx core.MetadataWriter, integration core.IntegrationContext, repository string) (*RepositoryMetadata, error) {
	if repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	var nodeMetadata NodeMetadata
	if err := mapstructure.Decode(ctx.Get(), &nodeMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode node metadata: %w", err)
	}

	if nodeMetadata.Repository != nil && repositoryMetadataMatches(*nodeMetadata.Repository, repository) {
		return nodeMetadata.Repository, nil
	}

	var integrationMetadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	client, err := NewClient(integrationMetadata.AuthType, http, integration)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	repositories, err := client.ListRepositories(integrationMetadata.Workspace.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	repoIndex := slices.IndexFunc(repositories, func(r Repository) bool {
		return repositoryMatches(r, repository)
	})

	if repoIndex == -1 {
		return nil, fmt.Errorf("repository %s is not accessible to workspace", repository)
	}

	repoMetadata := &RepositoryMetadata{
		UUID:     repositories[repoIndex].UUID,
		Name:     repositories[repoIndex].Name,
		FullName: repositories[repoIndex].FullName,
		Slug:     repositories[repoIndex].Slug,
	}

	return repoMetadata, ctx.Set(NodeMetadata{Repository: repoMetadata})
}

func repositoryMetadataMatches(repo RepositoryMetadata, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}

func repositoryMatches(repo Repository, repository string) bool {
	return repo.FullName == repository || repo.Name == repository || repo.Slug == repository || repo.UUID == repository
}

// integrationClient builds a Bitbucket client and returns the integration metadata
// alongside it, since every API path is scoped to the connected workspace.
func integrationClient(httpCtx core.HTTPContext, integration core.IntegrationContext) (*Client, *Metadata, error) {
	var metadata Metadata
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if metadata.Workspace == nil {
		return nil, nil, fmt.Errorf("integration is missing workspace metadata")
	}

	client, err := NewClient(metadata.AuthType, httpCtx, integration)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %w", err)
	}

	return client, &metadata, nil
}

// repositoryTarget holds everything a component needs to call the Bitbucket API.
type repositoryTarget struct {
	Client     *Client
	Workspace  string
	Repository string
}

// resolveRepositoryTarget turns the repository the user picked — which may be a
// UUID, a slug, a name or a full name — into the slug the API expects. The lookup
// result is cached in the node metadata by ensureRepoInMetadata, so repeated
// executions do not re-list the workspace repositories.
func resolveRepositoryTarget(httpCtx core.HTTPContext, metadataCtx core.MetadataWriter, integration core.IntegrationContext, repository string) (*repositoryTarget, error) {
	client, metadata, err := integrationClient(httpCtx, integration)
	if err != nil {
		return nil, err
	}

	repo, err := ensureRepoInMetadata(httpCtx, metadataCtx, integration, repository)
	if err != nil {
		return nil, err
	}

	return &repositoryTarget{
		Client:     client,
		Workspace:  metadata.Workspace.Slug,
		Repository: repo.Slug,
	}, nil
}

// verifyWebhookSignature checks the X-Hub-Signature header Bitbucket sends with every
// delivery, and returns the status code the webhook should be answered with on failure.
func verifyWebhookSignature(ctx core.WebhookRequestContext) (int, error) {
	signature := ctx.Headers.Get("X-Hub-Signature")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("missing X-Hub-Signature header")
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	if signature == "" {
		return http.StatusForbidden, fmt.Errorf("invalid signature format")
	}

	secret, err := ctx.Webhook.GetSecret()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting webhook secret")
	}

	if err := crypto.VerifySignature(secret, ctx.Body, signature); err != nil {
		return http.StatusForbidden, fmt.Errorf("invalid signature")
	}

	return http.StatusOK, nil
}

// repositoryField builds the repository picker shared by every Bitbucket component.
func repositoryField() configuration.Field {
	return configuration.Field{
		Name:     "repository",
		Label:    "Repository",
		Type:     configuration.FieldTypeIntegrationResource,
		Required: true,
		TypeOptions: &configuration.TypeOptions{
			Resource: &configuration.ResourceTypeOptions{
				Type:           ResourceTypeRepository,
				UseNameAsValue: true,
			},
		},
	}
}

// commitField builds the commit selector shared by the build status components.
func commitField() configuration.Field {
	return configuration.Field{
		Name:        "commit",
		Label:       "Commit",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Placeholder: "{{ root().data.pullrequest.source.commit.hash }}",
		Description: "The full commit hash to report on",
	}
}

func isValidCommitStatusState(state string) bool {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case StateInProgress, StateSuccessful, StateFailed, StateStopped:
		return true
	default:
		return false
	}
}

// pullRequestIDField builds the pull request selector shared by the pull request
// actions. It is a string rather than a number because the value almost always comes
// from an expression over a trigger payload.
func pullRequestIDField() configuration.Field {
	return configuration.Field{
		Name:        "pullRequestId",
		Label:       "Pull Request ID",
		Type:        configuration.FieldTypeString,
		Required:    true,
		Placeholder: "42 or {{ root().data.pullrequest.id }}",
		Description: "The numeric ID of the pull request",
	}
}
