package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"golang.org/x/oauth2"
)

/*
 * For GitHub, we are using the SDK instead of plain HTTP calls,
 * so we need a way to create the client using the HTTP context.
 */
type transport struct {
	http core.HTTPContext
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.http.Do(req)
}

type Client struct {
	authMethod string
	ownerType  string
	owner      string
	underlying *github.Client
}

func IsNotFoundError(err error) bool {
	var githubErr *github.ErrorResponse
	if errors.As(err, &githubErr) && githubErr.Response != nil && githubErr.Response.StatusCode == http.StatusNotFound {
		return true
	}

	var installationErr *ghinstallation.HTTPError
	return errors.As(err, &installationErr) &&
		installationErr.Response != nil &&
		installationErr.Response.StatusCode == http.StatusNotFound
}

func (c *Client) FindRepository(repository string) (*github.Repository, error) {
	repo, _, err := c.underlying.Repositories.Get(context.Background(), c.owner, repository)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	return repo, nil
}

func (c *Client) listAppRepositories() ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		repos, resp, err := c.underlying.Apps.ListRepos(context.Background(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories from GitHub API: %w", err)
		}

		allRepos = append(allRepos, repos.Repositories...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) ListRepositories() ([]*github.Repository, error) {
	switch c.authMethod {
	case AuthMethodApp:
		return c.listAppRepositories()
	case AuthMethodPAT:
		return c.listOwnerRepositories()
	}

	return nil, fmt.Errorf("invalid auth method: %s", c.authMethod)
}

func (c *Client) listOwnerRepositories() ([]*github.Repository, error) {
	switch c.ownerType {
	case OwnerTypeUser:
		return c.listUserRepositories()
	case OwnerTypeOrganization:
		return c.listOrganizationRepositories()
	default:
		return nil, fmt.Errorf("invalid owner type: %s", c.ownerType)
	}
}

func (c *Client) listUserRepositories() ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opts := &github.RepositoryListByAuthenticatedUserOptions{
		Affiliation: "owner",
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.underlying.Repositories.ListByAuthenticatedUser(context.Background(), opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories from GitHub API: %w", err)
		}

		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) listOrganizationRepositories() ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opts := &github.RepositoryListByOrgOptions{
		Sort:        "updated",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.underlying.Repositories.ListByOrg(context.Background(), c.owner, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories from GitHub API: %w", err)
		}

		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (c *Client) ListBranches(ctx context.Context, repository string, opts *github.BranchListOptions) ([]*github.Branch, *github.Response, error) {
	return c.underlying.Repositories.ListBranches(ctx, c.owner, repository, opts)
}

func (c *Client) CreateIssueReaction(ctx context.Context, repository string, commentID int64, content string) (*github.Reaction, *github.Response, error) {
	return c.underlying.Reactions.CreateIssueCommentReaction(ctx, c.owner, repository, commentID, content)
}

func (c *Client) CreateReviewCommentReaction(ctx context.Context, repository string, commentID int64, content string) (*github.Reaction, *github.Response, error) {
	return c.underlying.Reactions.CreatePullRequestCommentReaction(ctx, c.owner, repository, commentID, content)
}

func (c *Client) CreatePullRequestReview(ctx context.Context, repository string, pullNumber int, review *github.PullRequestReviewRequest) (*github.PullRequestReview, *github.Response, error) {
	return c.underlying.PullRequests.CreateReview(ctx, c.owner, repository, pullNumber, review)
}

func (c *Client) CreatePullRequest(ctx context.Context, repository string, pullRequest *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	return c.underlying.PullRequests.Create(ctx, c.owner, repository, pullRequest)
}

func (c *Client) GetPullRequest(ctx context.Context, repository string, pullNumber int) (*github.PullRequest, *github.Response, error) {
	return c.underlying.PullRequests.Get(ctx, c.owner, repository, pullNumber)
}

func (c *Client) EditPullRequest(ctx context.Context, repository string, pullNumber int, pullRequest *github.PullRequest) (*github.PullRequest, *github.Response, error) {
	return c.underlying.PullRequests.Edit(ctx, c.owner, repository, pullNumber, pullRequest)
}

// MarkPullRequestReadyForReview takes a pull request out of the draft state.
// GitHub exposes this only through the GraphQL API, so it is the one pull
// request operation that cannot delegate to the REST client. It takes the pull
// request node ID (github.PullRequest.NodeID), not the pull request number.
func (c *Client) MarkPullRequestReadyForReview(ctx context.Context, pullRequestID string) error {
	const mutation = `mutation($pullRequestId: ID!) {
	  markPullRequestReadyForReview(input: {pullRequestId: $pullRequestId}) {
	    pullRequest {
	      number
	      isDraft
	    }
	  }
	}`

	return c.doGraphQL(ctx, mutation, map[string]any{"pullRequestId": pullRequestID})
}

func (c *Client) MergePullRequest(ctx context.Context, repository string, pullNumber int, commitMessage string, options *github.PullRequestOptions) (*github.PullRequestMergeResult, *github.Response, error) {
	return c.underlying.PullRequests.Merge(ctx, c.owner, repository, pullNumber, commitMessage, options)
}

func (c *Client) AddPullRequestReviewers(ctx context.Context, repository string, pullNumber int, reviewers github.ReviewersRequest) (*github.PullRequest, *github.Response, error) {
	return c.underlying.PullRequests.RequestReviewers(ctx, c.owner, repository, pullNumber, reviewers)
}

func (c *Client) CreateStatus(ctx context.Context, repository string, sha string, status github.RepoStatus) (*github.RepoStatus, *github.Response, error) {
	return c.underlying.Repositories.CreateStatus(ctx, c.owner, repository, sha, status)
}

func (c *Client) GetCombinedStatus(ctx context.Context, repository string, ref string, opts *github.ListOptions) (*github.CombinedStatus, *github.Response, error) {
	return c.underlying.Repositories.GetCombinedStatus(ctx, c.owner, repository, ref, opts)
}

func (c *Client) ListCheckRunsForRef(ctx context.Context, repository string, ref string, opts *github.ListCheckRunsOptions) (*github.ListCheckRunsResults, *github.Response, error) {
	return c.underlying.Checks.ListCheckRunsForRef(ctx, c.owner, repository, ref, opts)
}

func (c *Client) CreateDeployment(ctx context.Context, repository string, request *github.DeploymentRequest) (*github.Deployment, *github.Response, error) {
	return c.underlying.Repositories.CreateDeployment(ctx, c.owner, repository, request)
}

func (c *Client) CreateDeploymentStatus(ctx context.Context, repository string, deploymentID int64, request *github.DeploymentStatusRequest) (*github.DeploymentStatus, *github.Response, error) {
	return c.underlying.Repositories.CreateDeploymentStatus(ctx, c.owner, repository, deploymentID, request)
}

func (c *Client) CreateWorkflowDispatchEvent(ctx context.Context, repository string, workflowFile string, request github.CreateWorkflowDispatchEventRequest) (*github.WorkflowDispatchRunDetails, *github.Response, error) {
	return c.underlying.Actions.CreateWorkflowDispatchEventByFileName(ctx, c.owner, repository, workflowFile, request)
}

func (c *Client) GetRepositoryPermissionLevel(ctx context.Context, repository string, username string) (*github.RepositoryPermissionLevel, *github.Response, error) {
	return c.underlying.Repositories.GetPermissionLevel(ctx, c.owner, repository, username)
}

func (c *Client) CancelWorkflowRun(repository string, workflowRunID int64) (*github.Response, error) {
	return c.underlying.Actions.CancelWorkflowRunByID(context.Background(), c.owner, repository, workflowRunID)
}

func (c *Client) GetWorkflowRun(repository string, workflowRunID int64) (*github.WorkflowRun, *github.Response, error) {
	return c.underlying.Actions.GetWorkflowRunByID(context.Background(), c.owner, repository, workflowRunID)
}

func (c *Client) AddIssueAssignees(ctx context.Context, repository string, issueNumber int, assignees []string) (*github.Issue, *github.Response, error) {
	return c.underlying.Issues.AddAssignees(ctx, c.owner, repository, issueNumber, assignees)
}

func (c *Client) RemoveIssueAssignees(ctx context.Context, repository string, issueNumber int, assignees []string) (*github.Issue, *github.Response, error) {
	return c.underlying.Issues.RemoveAssignees(ctx, c.owner, repository, issueNumber, assignees)
}

func (c *Client) CreateIssueComment(ctx context.Context, repository string, issueNumber int, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	return c.underlying.Issues.CreateComment(ctx, c.owner, repository, issueNumber, comment)
}

func (c *Client) EditIssueComment(ctx context.Context, repository string, commentID int64, comment *github.IssueComment) (*github.IssueComment, *github.Response, error) {
	return c.underlying.Issues.EditComment(ctx, c.owner, repository, commentID, comment)
}

func (c *Client) CreateIssue(ctx context.Context, repository string, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	return c.underlying.Issues.Create(ctx, c.owner, repository, issue)
}

func (c *Client) GetIssue(ctx context.Context, repository string, issueNumber int) (*github.Issue, *github.Response, error) {
	return c.underlying.Issues.Get(ctx, c.owner, repository, issueNumber)
}

func (c *Client) EditIssue(ctx context.Context, repository string, issueNumber int, issue *github.IssueRequest) (*github.Issue, *github.Response, error) {
	return c.underlying.Issues.Edit(ctx, c.owner, repository, issueNumber, issue)
}

func (c *Client) AddLabelsToIssue(ctx context.Context, repository string, issueNumber int, labels []string) ([]*github.Label, *github.Response, error) {
	return c.underlying.Issues.AddLabelsToIssue(ctx, c.owner, repository, issueNumber, labels)
}

func (c *Client) RemoveLabelForIssue(ctx context.Context, repository string, issueNumber int, label string) (*github.Response, error) {
	return c.underlying.Issues.RemoveLabelForIssue(ctx, c.owner, repository, issueNumber, label)
}

func (c *Client) ListLabelsForIssue(ctx context.Context, repository string, issueNumber int) ([]*github.Label, *github.Response, error) {
	return c.underlying.Issues.ListLabelsByIssue(ctx, c.owner, repository, issueNumber, nil)
}

func (c *Client) GetRef(repository string, ref string) (*github.Reference, *github.Response, error) {
	return c.underlying.Git.GetRef(context.Background(), c.owner, repository, ref)
}

func (c *Client) DeleteRef(ctx context.Context, repository string, ref string) (*github.Response, error) {
	return c.underlying.Git.DeleteRef(ctx, c.owner, repository, ref)
}

func (c *Client) GetRelease(ctx context.Context, repository string, id int64) (*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.GetRelease(ctx, c.owner, repository, id)
}

func (c *Client) CreateRelease(ctx context.Context, repository string, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.CreateRelease(ctx, c.owner, repository, release)
}

func (c *Client) EditRelease(ctx context.Context, repository string, id int64, release *github.RepositoryRelease) (*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.EditRelease(ctx, c.owner, repository, id, release)
}

func (c *Client) GenerateReleaseNotes(ctx context.Context, repository string, options *github.GenerateNotesOptions) (*github.RepositoryReleaseNotes, *github.Response, error) {
	return c.underlying.Repositories.GenerateReleaseNotes(ctx, c.owner, repository, options)
}

func (c *Client) ListReleases(ctx context.Context, repository string, options *github.ListOptions) ([]*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.ListReleases(ctx, c.owner, repository, options)
}

func (c *Client) GetReleaseByTag(ctx context.Context, repository string, tag string) (*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.GetReleaseByTag(ctx, c.owner, repository, tag)
}

func (c *Client) GetLatestRelease(ctx context.Context, repository string) (*github.RepositoryRelease, *github.Response, error) {
	return c.underlying.Repositories.GetLatestRelease(ctx, c.owner, repository)
}

func (c *Client) DeleteRelease(ctx context.Context, repository string, id int64) (*github.Response, error) {
	return c.underlying.Repositories.DeleteRelease(ctx, c.owner, repository, id)
}

func (c *Client) CreateHook(ctx context.Context, repository string, hook *github.Hook) (*github.Hook, *github.Response, error) {
	return c.underlying.Repositories.CreateHook(ctx, c.owner, repository, hook)
}

func (c *Client) DeleteHook(ctx context.Context, repository string, hookID int64) (*github.Response, error) {
	return c.underlying.Repositories.DeleteHook(ctx, c.owner, repository, hookID)
}

func (c *Client) GetOrganizationUsageReport() (*github.UsageReport, *github.Response, error) {
	return c.underlying.Billing.GetOrganizationUsageReport(context.Background(), c.owner, nil)
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// doGraphQL sends a mutation or query to GitHub's GraphQL endpoint, reusing the
// REST client so that authentication and the HTTP context transport apply.
// GraphQL reports failures as an `errors` array on an HTTP 200, so those are
// turned into a regular error here.
func (c *Client) doGraphQL(ctx context.Context, query string, variables map[string]any) error {
	request, err := c.underlying.NewRequest(http.MethodPost, "graphql", graphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return fmt.Errorf("failed to build GraphQL request: %w", err)
	}

	var response graphQLResponse
	if _, err := c.underlying.Do(ctx, request, &response); err != nil {
		return err
	}

	if len(response.Errors) == 0 {
		return nil
	}

	messages := []string{}
	for _, e := range response.Errors {
		if e.Message != "" {
			messages = append(messages, e.Message)
		}
	}

	if len(messages) == 0 {
		return errors.New("GraphQL request failed")
	}

	return errors.New(strings.Join(messages, ": "))
}

func NewClient(ctx core.IntegrationContext, httpCtx core.HTTPContext) (*Client, error) {
	if !ctx.LegacySetup() {
		return newClientFromStorageContexts(httpCtx, ctx.Properties(), ctx.Secrets())
	}

	var metadata Metadata
	if err := mapstructure.Decode(ctx.GetMetadata(), &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %v", err)
	}

	ID, err := strconv.Atoi(metadata.InstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := FindSecret(ctx, GitHubAppPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to find PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		&transport{http: httpCtx},
		metadata.GitHubApp.ID,
		int64(ID),
		[]byte(pem),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return &Client{
		authMethod: AuthMethodApp,
		ownerType:  determineLegacyOwnerType(ctx),
		owner:      metadata.Owner,
		underlying: github.NewClient(&http.Client{Transport: itr}),
	}, nil
}

func determineLegacyOwnerType(ctx core.IntegrationContext) string {
	_, err := ctx.GetConfig("organization")
	if err != nil {
		return OwnerTypeUser
	}

	return OwnerTypeOrganization
}

func newClientFromStorageContexts(httpCtx core.HTTPContext, properties core.IntegrationPropertyStorageReader, secrets core.IntegrationSecretStorageReader) (*Client, error) {
	authMethod, err := properties.GetString(PropertyAuthMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to get authentication method: %v", err)
	}

	owner, err := properties.GetString(PropertyOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner: %v", err)
	}

	ownerType, err := properties.GetString(PropertyOwnerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get owner type: %v", err)
	}

	switch authMethod {
	case AuthMethodPAT:
		return newPATClient(owner, ownerType, httpCtx, secrets)
	case AuthMethodApp:
		return newGitHubAppClient(owner, ownerType, httpCtx, properties, secrets)
	default:
		return nil, fmt.Errorf("invalid authentication method: %s", authMethod)
	}
}

func newPATClient(owner string, ownerType string, httpCtx core.HTTPContext, secrets core.IntegrationSecretStorageReader) (*Client, error) {
	pat, err := secrets.Get(SecretPAT)
	if err != nil {
		return nil, fmt.Errorf("failed to get PAT: %v", err)
	}

	pat = strings.TrimSpace(string(pat))
	if pat == "" {
		return nil, fmt.Errorf("PAT is required")
	}

	return &Client{
		authMethod: AuthMethodPAT,
		ownerType:  ownerType,
		owner:      owner,
		underlying: github.NewClient(&http.Client{
			Transport: &oauth2.Transport{
				Base: &transport{http: httpCtx},
				Source: oauth2.StaticTokenSource(&oauth2.Token{
					AccessToken: strings.TrimSpace(pat),
				}),
			},
		}),
	}, nil
}

func newGitHubAppClient(owner string, ownerType string, httpCtx core.HTTPContext, properties core.IntegrationPropertyStorageReader, secrets core.IntegrationSecretStorageReader) (*Client, error) {
	appID, err := properties.GetString(PropertyAppID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub app ID: %v", err)
	}

	appNumber, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub app ID: %v", err)
	}

	installationID, err := properties.GetString(PropertyAppInstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get installation ID: %v", err)
	}

	installationNumber, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse installation ID: %v", err)
	}

	pem, err := secrets.Get(SecretAppPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to get PEM: %v", err)
	}

	itr, err := ghinstallation.New(
		&transport{http: httpCtx},
		appNumber,
		installationNumber,
		[]byte(pem),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create apps transport: %v", err)
	}

	return &Client{
		authMethod: AuthMethodApp,
		ownerType:  ownerType,
		owner:      owner,
		underlying: github.NewClient(&http.Client{Transport: itr}),
	}, nil
}

func FindSecret(ctx core.IntegrationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}
