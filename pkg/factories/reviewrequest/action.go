package reviewrequest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/factories/lifecycle"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
)

// prepareAction checks authority at dispatch and again immediately before
// mutation. The caller keeps the PR lock through the GitHub action.
func prepareAction(action lifecycle.Action) (bool, error) {
	c, ctx := action, action.Context
	if c.Canvas.FactoryID == nil || (c.Execution.NodeID != selectionNodeID && c.Execution.NodeID != requestNodeID) {
		return false, nil
	}
	exists, err := installed(c.Tx, c.Canvas)
	if err != nil || !exists {
		return false, err
	}
	run, err := models.FindCanvasRunInTransaction(c.Tx, c.Canvas.ID, c.Execution.RunID)
	if err != nil {
		return false, err
	}
	source, err := models.CheckReviewRequestAttempt(c.Tx, c.Canvas, run)
	if err != nil {
		return false, err
	}
	if !needsReview(source.Report) {
		return true, ctx.ExecutionState.Pass()
	}
	client, err := githubClient(c)
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
	if c.Execution.NodeID == selectionNodeID {
		return false, prepareSelectionAgent(c, source, candidates)
	}
	outputs, err := models.ListReviewRequestDecisionEvents(c.Tx, run, selectionNodeID)
	if err != nil {
		return false, err
	}
	if len(outputs) != 1 {
		return false, fmt.Errorf("Review request has no unique validated decision")
	}
	envelope, _ := outputs[0].Data.Data().(map[string]any)
	data, _ := envelope["data"].(map[string]any)
	var decision selectionDecision
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
		if slices.ContainsFunc(remote.RequestedReviewers, func(user *github.User) bool { return strings.EqualFold(user.GetLogin(), selected.Login) }) || hasCurrentApproval(reviews, selected.Login, sha) {
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
	if _, err := models.CheckReviewRequestAttempt(c.Tx, c.Canvas, run); err != nil {
		return false, err
	}
	metadata := maps.Clone(c.Execution.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["reviewRequest"] = decision
	if err := c.Execution.UpdateMetadata(c.Tx, metadata); err != nil {
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

func githubClient(c lifecycle.Action) (*githubcommon.Client, error) {
	ctx := c.Context
	if ctx.Integration != nil {
		return githubcommon.NewClient(ctx.Integration, ctx.HTTP)
	}
	workspace, err := models.FindFactory(c.Tx, c.Canvas.OrganizationID, *c.Canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(workspace.OnboardingConfigValue().VCSIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("Review request requires the workspace GitHub integration")
	}
	integration, err := models.FindIntegrationInTransaction(c.Tx, c.Canvas.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if integration.AppName != "github" || integration.State != models.IntegrationStateReady || c.Integration == nil {
		return nil, fmt.Errorf("Review request GitHub integration is unavailable")
	}
	return githubcommon.NewClient(c.Integration(integration), ctx.HTTP)
}

func prepareSelectionAgent(c lifecycle.Action, source *models.ReviewRequestSource, candidates []*github.User) error {
	ctx := c.Context
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
	metadata := maps.Clone(c.Execution.Metadata.Data())
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["reviewRequestCandidates"] = logins
	metadata["reviewRequestCandidateIDs"] = identities
	metadata["reviewRequestDiscovered"] = true
	if err := c.Execution.UpdateMetadata(c.Tx, metadata); err != nil {
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

func hasCurrentApproval(reviews []*github.PullRequestReview, login, sha string) bool {
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
