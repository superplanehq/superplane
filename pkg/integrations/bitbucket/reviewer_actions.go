package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const reviewCandidatesPayload = "bitbucket.pullRequestReviewCandidates"
const reviewersEnsuredPayload = "bitbucket.pullRequestReviewersEnsured"

type ListPullRequestReviewCandidates struct{ pullRequestAction }
type EnsurePullRequestReviewers struct{ pullRequestAction }

type reviewCandidatesConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	PullNumber any    `json:"pullNumber" mapstructure:"pullNumber"`
}

// ReviewerRequest selects identities for one assessed pull request.
// PullNumber accepts a number from an expression or text from the configuration form.
type ReviewerRequest struct {
	Repository      string             `json:"repository" mapstructure:"repository"`
	PullNumber      any                `json:"pullNumber" mapstructure:"pullNumber"`
	ExpectedHeadSHA string             `json:"expectedHeadSha" mapstructure:"expectedHeadSha"`
	Reviewers       []ReviewerIdentity `json:"reviewers" mapstructure:"reviewers"`
}

func (*ListPullRequestReviewCandidates) Name() string {
	return "bitbucket.listPullRequestReviewCandidates"
}
func (*ListPullRequestReviewCandidates) Label() string { return "List Pull Request Review Candidates" }
func (*ListPullRequestReviewCandidates) Description() string {
	return "Find active human reviewers with effective repository write access"
}
func (*ListPullRequestReviewCandidates) Documentation() string {
	return `List eligible humans, excluding the pull request author, app users, bots, and inactive accounts. Read all effective permission pages, including group grants. Repository admin access is required.

## Configuration

- **Repository**: Repository in workspace/repository format.
- **Pull Request Number**: Pull request number. Supports expressions.

## Output

The default channel emits provider, integrationId, repository, repositoryId, pullNumber, headSha, state, author, and candidates. Each identity has uuid and displayName. UUIDs are canonical and have no braces. Candidate evidence includes default_reviewer and recent_contributor. Defaults include project inheritance. Contributor evidence samples the latest 100 repository commits. contributorCommits and contributorHistoryLimited describe this sample. Empty evidence means an eligible fallback candidate. Defaults and commits do not prove ownership. SuperPlane does not interpret CODEOWNERS.

Use an API token with read:repository:bitbucket, read:pullrequest:bitbucket, and read:user:bitbucket scopes. The account must have repository admin permission. Hosted Forge and access tokens do not currently support account verification. Permission or incomplete-data failures fail the execution. See docs/contributing/bitbucket-reviewers.md for credential limits.`
}
func (*ListPullRequestReviewCandidates) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (*ListPullRequestReviewCandidates) Configuration() []configuration.Field {
	return []configuration.Field{repositoryField(), {Name: "pullNumber", Label: "Pull Request Number", Type: configuration.FieldTypeString, Required: true, Description: "Pull request number. Supports expressions."}}
}

func newReviewerClient(httpContext core.HTTPContext, integration core.IntegrationContext) (*Client, error) {
	metadata := Metadata{}
	if err := mapstructure.Decode(integration.GetMetadata(), &metadata); err != nil {
		return nil, err
	}
	if metadata.AuthType != AuthTypeAPIToken {
		return nil, fmt.Errorf("reviewer verification requires an API token with repository and user read scopes plus repository admin access; hosted Forge and access tokens are not supported")
	}
	client, err := newIntegrationClient(httpContext, integration)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (*ListPullRequestReviewCandidates) Setup(ctx core.SetupContext) error {
	var config reviewCandidatesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if !isExpression(fmt.Sprint(config.PullNumber)) {
		if _, err := parsePullRequestID(fmt.Sprint(config.PullNumber)); err != nil {
			return err
		}
	}
	client, err := newReviewerClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}
	if err := setupRepository(ctx, config.Repository); err != nil {
		return err
	}
	if isExpression(config.Repository) {
		return nil
	}
	_, err = client.ListEligibleReviewers(config.Repository)
	return err
}

func reviewerScope(ctx core.ExecutionContext, repository string) (*Client, string, error) {
	if err := requireRepositoryInWorkspace(ctx.Integration, repository); err != nil {
		return nil, "", err
	}
	client, err := newReviewerClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, "", err
	}
	repo, err := client.GetRepository(repository)
	if err != nil {
		return nil, "", err
	}
	id, err := NormalizeAccountID(repo.UUID)
	if err != nil {
		return nil, "", fmt.Errorf("repository UUID is missing or invalid")
	}
	return client, id, nil
}

func (*ListPullRequestReviewCandidates) Execute(ctx core.ExecutionContext) error {
	var config reviewCandidatesConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	number, err := parsePullRequestID(fmt.Sprint(config.PullNumber))
	if err != nil {
		return err
	}
	client, repositoryID, err := reviewerScope(ctx, config.Repository)
	if err != nil {
		return err
	}
	candidates, err := client.ListPullRequestReviewCandidates(config.Repository, number)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit("default", reviewCandidatesPayload, []any{struct {
		Provider      string `json:"provider"`
		IntegrationID string `json:"integrationId"`
		Repository    string `json:"repository"`
		RepositoryID  string `json:"repositoryId"`
		PullNumber    int64  `json:"pullNumber"`
		ReviewCandidates
	}{"bitbucket", ctx.Integration.ID().String(), config.Repository, repositoryID, number, candidates}})
}

func (*EnsurePullRequestReviewers) Name() string  { return "bitbucket.ensurePullRequestReviewers" }
func (*EnsurePullRequestReviewers) Label() string { return "Ensure Pull Request Reviewers" }
func (*EnsurePullRequestReviewers) Description() string {
	return "Request eligible reviewers and preserve existing reviewers"
}
func (*EnsurePullRequestReviewers) Documentation() string {
	return `Request selected humans by UUID after rechecking effective access, account status, the pull request state, and its assessed commit.

## Configuration

- **Repository**: Repository in workspace/repository format.
- **Pull Request Number**: Pull request number. Supports expressions.
- **Expected Head SHA**: Required full assessed commit hash. Changed or closed pull requests emit stale_pr.
- **Reviewers**: Distinct Bitbucket UUIDs in objects with a uuid field. The repository reviewer picker stores UUIDs. displayName is optional.

## Output

The default channel emits executionId, provider, integrationId, repository, repositoryId, pullNumber, headSha, status, reason, requested, alreadyCovered, and unresolvedApprovals. Status is requested, already_covered, or stale_pr. Errors fail the execution and emit no success payload. Existing reviewer assignments count as covered. Participant approvals without reliable commit evidence stay unresolved.

SuperPlane serializes its own reviewer writes for one integration, repository, and pull request. It merges current reviewers, writes only the reviewers field, and verifies the result. Conflicts and missing reviewers allow three reconciliation attempts. Strong ETags are sent as If-Match when present. Bitbucket does not document an atomic additive update. Concurrent external edits can still race. Missing reviewers, changed approvals, and uncertain updates remain errors.

Use an API token with read:repository:bitbucket, read:user:bitbucket, read:pullrequest:bitbucket, and write:pullrequest:bitbucket scopes. Repository admin access is required. Hosted Forge and access tokens are not supported. Approval-evidence integration remains a merge requirement. See docs/contributing/bitbucket-reviewers.md.`
}
func (*EnsurePullRequestReviewers) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (*EnsurePullRequestReviewers) Configuration() []configuration.Field {
	fields := (&ListPullRequestReviewCandidates{}).Configuration()
	return append(fields,
		configuration.Field{Name: "expectedHeadSha", Label: "Expected Head SHA", Type: configuration.FieldTypeString, Required: true, Description: "Full assessed commit hash. A changed commit skips the request."},
		configuration.Field{Name: "reviewers", Label: "Reviewers", Type: configuration.FieldTypeList, Required: true, TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{ItemLabel: "Reviewer", ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeObject, Schema: []configuration.Field{
			{Name: "uuid", Label: "Reviewer", Type: configuration.FieldTypeIntegrationResource, Required: true, TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: resourceTypeReviewer, Parameters: []configuration.ParameterRef{{Name: "repository", ValueFrom: &configuration.ParameterValueFrom{Field: "repository"}}}}}},
			{Name: "displayName", Label: "Display Name", Type: configuration.FieldTypeString, Description: "Optional label. The UUID identifies the reviewer."},
		}}}}},
	)
}
func (*EnsurePullRequestReviewers) Setup(ctx core.SetupContext) error {
	fields, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("configuration must be an object")
	}
	if value, ok := fields["reviewers"].(string); !ok || !isExpression(value) {
		var config ReviewerRequest
		if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
			return err
		}
		if _, err := normalizeReviewers(config.Reviewers); err != nil {
			return err
		}
	}
	sha, _ := fields["expectedHeadSha"].(string)
	if strings.TrimSpace(sha) == "" {
		return fmt.Errorf("expected head SHA is required")
	}
	return (&ListPullRequestReviewCandidates{}).Setup(ctx)
}

// ReviewerWriteKey is scoped by provider, stable repository UUID, and PR number.
// PublicationStore adds the integration boundary. All automation writers must use it.
func ReviewerWriteKey(repositoryID string, number int64) string {
	return fmt.Sprintf("bitbucket/pull-request-reviewers/%s/%d", repositoryID, number)
}

type reviewerHTTPContext struct {
	core.HTTPContext
	context context.Context
}

func (h reviewerHTTPContext) Do(request *http.Request) (*http.Response, error) {
	if err := h.context.Err(); err != nil {
		return nil, err
	}
	return h.HTTPContext.Do(request.WithContext(h.context))
}

// EnsurePullRequestReviewersSerialized reuses durable publication leases. No
// database transaction stays open during provider calls. Requests end before lease expiry.
func (c *Client) EnsurePullRequestReviewersSerialized(store core.PublicationStore, repositoryID string, request ReviewerRequest) (ReviewerRequestOutcome, error) {
	if store == nil {
		return ReviewerRequestOutcome{}, fmt.Errorf("durable reviewer write serialization is unavailable")
	}
	number, err := parsePullRequestID(fmt.Sprint(request.PullNumber))
	if err != nil {
		return ReviewerRequestOutcome{}, err
	}
	id, err := NormalizeAccountID(repositoryID)
	if err != nil {
		return ReviewerRequestOutcome{}, err
	}
	key := ReviewerWriteKey(id, number)
	lease, err := core.BeginPublicationLease(store, key, 2*time.Minute)
	if err != nil {
		return ReviewerRequestOutcome{}, err
	}
	deadline, cancel := context.WithDeadline(context.Background(), lease.ExpiresAt.Add(-15*time.Second))
	defer cancel()
	bounded := *c
	bounded.HTTP = reviewerHTTPContext{HTTPContext: c.HTTP, context: deadline}
	outcome, err := bounded.ensurePullRequestReviewers(request.Repository, number, request.ExpectedHeadSHA, request.Reviewers)
	if releaseErr := core.FinishPublicationLease(store, key, lease); releaseErr != nil && err == nil {
		return outcome, fmt.Errorf("cannot release reviewer write lease: %w", releaseErr)
	}
	return outcome, err
}

func (*EnsurePullRequestReviewers) Execute(ctx core.ExecutionContext) error {
	var config ReviewerRequest
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return err
	}
	if _, err := normalizeReviewers(config.Reviewers); err != nil {
		return err
	}
	if strings.TrimSpace(config.ExpectedHeadSHA) == "" {
		return fmt.Errorf("expected head SHA is required")
	}
	number, err := parsePullRequestID(fmt.Sprint(config.PullNumber))
	if err != nil {
		return err
	}
	client, repositoryID, err := reviewerScope(ctx, config.Repository)
	if err != nil {
		return err
	}
	outcome, err := client.EnsurePullRequestReviewersSerialized(ctx.Publications, repositoryID, config)
	if err != nil {
		return err
	}
	return ctx.ExecutionState.Emit("default", reviewersEnsuredPayload, []any{struct {
		ExecutionID   string `json:"executionId"`
		Provider      string `json:"provider"`
		IntegrationID string `json:"integrationId"`
		Repository    string `json:"repository"`
		RepositoryID  string `json:"repositoryId"`
		PullNumber    int64  `json:"pullNumber"`
		ReviewerRequestOutcome
	}{ctx.ID.String(), "bitbucket", ctx.Integration.ID().String(), config.Repository, repositoryID, number, outcome}})
}

func (*ListPullRequestReviewCandidates) ExampleOutput() map[string]any {
	return map[string]any{"type": reviewCandidatesPayload, "timestamp": "2026-04-22T10:00:00Z", "data": map[string]any{"provider": "bitbucket", "integrationId": "11111111-1111-1111-1111-111111111111", "repository": "acme/widgets", "repositoryId": "33333333-3333-3333-3333-333333333333", "pullNumber": 42, "headSha": "abc123", "state": "OPEN", "author": ReviewerIdentity{UUID: "44444444-4444-4444-4444-444444444444", DisplayName: "Author"}, "candidates": []ReviewCandidate{{ReviewerIdentity: ReviewerIdentity{UUID: "22222222-2222-2222-2222-222222222222", DisplayName: "Reviewer"}, Evidence: []string{"default_reviewer", "recent_contributor"}}}, "contributorCommits": 100, "contributorHistoryLimited": true}}
}
func (*EnsurePullRequestReviewers) ExampleOutput() map[string]any {
	return map[string]any{"type": reviewersEnsuredPayload, "timestamp": "2026-04-22T10:00:00Z", "data": map[string]any{"executionId": "55555555-5555-5555-5555-555555555555", "provider": "bitbucket", "integrationId": "11111111-1111-1111-1111-111111111111", "repository": "acme/widgets", "repositoryId": "33333333-3333-3333-3333-333333333333", "pullNumber": 42, "status": "requested", "headSha": "abc123", "requested": []ReviewerIdentity{{UUID: "22222222-2222-2222-2222-222222222222", DisplayName: "Reviewer"}}, "alreadyCovered": []ReviewerIdentity{}, "unresolvedApprovals": []ReviewerIdentity{}}}
}
