package contexts

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

// PrepareReviewRequest checks authority at dispatch and again immediately
// before mutation. The caller keeps the PR lock through the GitHub action.
func (c *FactoryContext) PrepareReviewRequest(ctx *core.ExecutionContext) (bool, error) {
	if c.canvas.FactoryID == nil || (c.execution.NodeID != "select-reviewers" && c.execution.NodeID != "request-reviewers") {
		return false, nil
	}
	installed, err := models.CanvasHasFactoryAppTemplate(c.tx, c.canvas.ID, "review-request")
	if err != nil || !installed {
		return false, err
	}
	run, err := models.FindCanvasRunInTransaction(c.tx, c.canvas.ID, c.execution.RunID)
	if err != nil {
		return false, err
	}
	source, err := models.CheckReviewRequestAttempt(c.tx, c.canvas, run)
	if err != nil {
		return false, err
	}
	client, err := c.reviewRequestClient(ctx)
	if err != nil {
		return false, err
	}
	repository := source.PullRequest.Repository
	number := int(source.PullRequest.Number)
	remote, _, err := client.GetPullRequest(context.Background(), repository, number)
	if err != nil {
		return false, fmt.Errorf("Read Review request pull request: %w", err)
	}
	sha, _ := source.Report["reviewedCommit"].(string)
	if remote.GetState() != "open" || remote.GetHead().GetSHA() != sha || remote.GetMerged() {
		return false, fmt.Errorf("Review request report does not assess the current open pull request")
	}
	if remote.GetUser().GetLogin() == "" {
		return false, fmt.Errorf("Review request pull request author is missing")
	}
	candidates, err := client.ListHumanReviewers(context.Background(), repository)
	if err != nil {
		return false, fmt.Errorf("Discover Review request reviewers: %w", err)
	}
	candidates = slices.DeleteFunc(candidates, func(user *github.User) bool {
		return strings.EqualFold(user.GetLogin(), remote.GetUser().GetLogin())
	})
	if c.execution.NodeID == "select-reviewers" {
		return false, c.prepareReviewRequestAgent(ctx, source, candidates)
	}
	outputs, err := models.ListReviewRequestDecisionEvents(c.tx, run, "select-reviewers")
	if err != nil {
		return false, err
	}
	if len(outputs) != 1 {
		return false, fmt.Errorf("Review request has no unique validated decision")
	}
	envelope, _ := outputs[0].Data.Data().(map[string]any)
	data, _ := envelope["data"].(map[string]any)
	var decision reviewRequestDecision
	encoded, err := json.Marshal(data["reviewRequest"])
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(encoded, &decision); err != nil || decision.ReviewRequired == nil || decision.NoCandidate == nil || decision.ReportID != source.Run.ID.String() || decision.SourceAppID != source.App.ID.String() || decision.ReviewedCommit != sha {
		return false, fmt.Errorf("Review request decision does not match its source report")
	}
	for _, selected := range decision.Reviewers {
		if !slices.ContainsFunc(candidates, func(user *github.User) bool {
			return strings.EqualFold(user.GetLogin(), selected.Login) && user.GetID() == selected.ID
		}) {
			return false, fmt.Errorf("Review request reviewer is no longer eligible: %s", selected.Login)
		}
		permission, _, err := client.GetRepositoryPermissionLevel(context.Background(), repository, selected.Login)
		if err != nil {
			return false, fmt.Errorf("Read reviewer permission: %w", err)
		}
		if !slices.Contains([]string{"write", "maintain", "admin"}, permission.GetPermission()) {
			return false, fmt.Errorf("Review request reviewer requires repository write access: %s", selected.Login)
		}
	}
	var reviews []*github.PullRequestReview
	if len(decision.Reviewers) > 0 {
		reviews, err = client.ListReviews(context.Background(), repository, number)
		if err != nil {
			return false, fmt.Errorf("Read existing pull request reviews: %w", err)
		}
	}
	toRequest := []string{}
	for _, selected := range decision.Reviewers {
		if slices.ContainsFunc(remote.RequestedReviewers, func(user *github.User) bool { return strings.EqualFold(user.GetLogin(), selected.Login) }) || hasCurrentReviewRequestApproval(reviews, selected.Login, sha) {
			continue
		}
		toRequest = append(toRequest, selected.Login)
	}
	// Permission and review lookups can take time. Reload the head immediately
	// before the write, while the persisted attempt remains fenced by the lock.
	remote, _, err = client.GetPullRequest(context.Background(), repository, number)
	if err != nil {
		return false, fmt.Errorf("Revalidate Review request pull request: %w", err)
	}
	if remote.GetState() != "open" || remote.GetMerged() || remote.GetHead().GetSHA() != sha {
		return false, fmt.Errorf("Review request pull request changed before applying the decision")
	}
	toRequest = slices.DeleteFunc(toRequest, func(login string) bool {
		return slices.ContainsFunc(remote.RequestedReviewers, func(user *github.User) bool { return strings.EqualFold(user.GetLogin(), login) })
	})
	if _, err := models.CheckReviewRequestAttempt(c.tx, c.canvas, run); err != nil {
		return false, err
	}
	metadata := maps.Clone(c.execution.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["reviewRequest"] = decision
	if err := c.execution.UpdateMetadata(c.tx, metadata); err != nil {
		return false, err
	}
	if len(toRequest) == 0 {
		return true, ctx.ExecutionState.Emit("default", "reviewRequest.applied", []any{map[string]any{"reviewRequest": decision}})
	}
	configuration, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return false, fmt.Errorf("Review request configuration must be an object")
	}
	configuration = maps.Clone(configuration)
	configuration["repository"] = repository
	configuration["pullNumber"] = number
	configuration["reviewers"] = toRequest
	configuration["teamReviewers"] = []string{}
	ctx.Configuration = configuration
	return false, nil
}

func (c *FactoryContext) reviewRequestClient(ctx *core.ExecutionContext) (*githubcommon.Client, error) {
	if ctx.Integration != nil {
		return githubcommon.NewClient(ctx.Integration, ctx.HTTP)
	}
	workspace, err := models.FindFactory(c.tx, c.canvas.OrganizationID, *c.canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(workspace.OnboardingConfigValue().VCSIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("Review request requires the workspace GitHub integration")
	}
	integration, err := models.FindIntegrationInTransaction(c.tx, c.canvas.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if integration.AppName != "github" || integration.State != models.IntegrationStateReady || c.encryptor == nil || c.registry == nil {
		return nil, fmt.Errorf("Review request GitHub integration is unavailable")
	}
	return githubcommon.NewClient(NewIntegrationContext(c.tx, nil, integration, c.encryptor, c.registry, nil), ctx.HTTP)
}

func (c *FactoryContext) prepareReviewRequestAgent(ctx *core.ExecutionContext, source *models.ReviewRequestSource, candidates []*github.User) error {
	type candidateIdentity struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name,omitempty"`
	}
	logins := make([]string, 0, len(candidates))
	identities := make(map[string]int64, len(candidates))
	eligible := make([]candidateIdentity, 0, len(candidates))
	for _, candidate := range candidates {
		logins = append(logins, candidate.GetLogin())
		identities[strings.ToLower(candidate.GetLogin())] = candidate.GetID()
		eligible = append(eligible, candidateIdentity{ID: candidate.GetID(), Login: candidate.GetLogin(), Name: candidate.GetName()})
	}
	metadata := maps.Clone(c.execution.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["reviewRequestCandidates"] = logins
	metadata["reviewRequestCandidateIDs"] = identities
	metadata["reviewRequestDiscovered"] = true
	if err := c.execution.UpdateMetadata(c.tx, metadata); err != nil {
		return err
	}
	configuration, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("Review request configuration must be an object")
	}
	// Keep the execution's map so the runner's resolved hosted model is saved.
	environment, _ := configuration["environment"].([]any)
	environment = slices.Clone(environment)
	for _, variable := range []struct {
		name  string
		value any
	}{
		{"SUPERPLANE_REVIEW_REQUEST_CANDIDATES", eligible},
		{"SUPERPLANE_REVIEW_REQUEST_REPORT", source.Report},
	} {
		encoded, err := json.Marshal(variable.value)
		if err != nil {
			return err
		}
		environment = append(environment, map[string]any{"name": variable.name, "value": string(encoded), "valueSource": "literal"})
	}
	configuration["environment"] = environment
	ctx.Configuration = configuration
	return nil
}

func hasCurrentReviewRequestApproval(reviews []*github.PullRequestReview, login, sha string) bool {
	var latest *github.PullRequestReview
	for _, review := range reviews {
		if !strings.EqualFold(review.GetUser().GetLogin(), login) || review.GetState() == "COMMENTED" || review.GetState() == "PENDING" {
			continue
		}
		if latest == nil || review.GetSubmittedAt().After(latest.GetSubmittedAt().Time) || (review.GetSubmittedAt().Time.Equal(latest.GetSubmittedAt().Time) && review.GetID() > latest.GetID()) {
			latest = review
		}
	}
	return latest != nil && latest.GetState() == "APPROVED" && latest.GetCommitID() == sha
}
