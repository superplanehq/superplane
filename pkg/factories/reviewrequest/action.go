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
	"gorm.io/gorm"
)

// pullRequestState is the GitHub state that loadAction reads before the
// action transaction opens. prepareAction checks it against locked records.
type pullRequestState struct {
	executionID uuid.UUID
	repository  string
	number      int
	sha         string
	requested   []*github.User
	candidates  []*github.User
	permissions map[string]string
	reviews     []*github.PullRequestReview
}

// loadAction reads GitHub before the action transaction opens, so that no
// database lock is held during provider calls. It uses the unverified root
// event only to choose what to read. prepareAction verifies the source.
func loadAction(load lifecycle.ActionLoad) (any, error) {
	execution := load.Execution
	if execution.NodeID != selectionNodeID && execution.NodeID != requestNodeID {
		return nil, nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(load.DB, execution.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil {
		return nil, nil
	}
	exists, err := installed(load.DB, canvas)
	if err != nil || !exists {
		return nil, err
	}
	run, err := models.FindCanvasRunInTransaction(load.DB, canvas.ID, execution.RunID)
	if err != nil {
		return nil, err
	}
	root, err := run.RootEvent(load.DB)
	if err != nil {
		return nil, err
	}
	message, _ := models.RootEventSourcePayload(root.Data.Data()).(map[string]any)
	payload, _ := message["payload"].(map[string]any)
	report, _ := payload["mergeConfidence"].(map[string]any)
	if !needsReview(report) {
		return nil, nil
	}
	state := &pullRequestState{executionID: execution.ID}
	state.repository, _ = payload["repository"].(string)
	state.sha, _ = report["reviewedCommit"].(string)
	state.number = pullNumber(payload["pullNumber"])
	client, err := githubClient(load, canvas)
	if err != nil {
		return nil, err
	}
	remote, _, err := client.GetPullRequest(context.Background(), state.repository, state.number)
	if err != nil {
		return nil, fmt.Errorf("Read Review request pull request: %w", err)
	}
	if remote.GetState() != "open" || remote.GetHead().GetSHA() != state.sha || remote.GetMerged() {
		return nil, fmt.Errorf("Review request report does not assess the current open pull request")
	}
	author := remote.GetUser().GetLogin()
	if author == "" {
		return nil, fmt.Errorf("Review request pull request author is missing")
	}
	state.requested = remote.RequestedReviewers
	candidates, err := client.ListHumanReviewers(context.Background(), state.repository)
	if err != nil {
		return nil, fmt.Errorf("Discover Review request reviewers: %w", err)
	}
	state.candidates = slices.DeleteFunc(candidates, func(user *github.User) bool {
		return strings.EqualFold(user.GetLogin(), author)
	})
	if execution.NodeID == selectionNodeID {
		return state, nil
	}
	decision, err := loadDecision(load.DB, run)
	if err != nil {
		return nil, err
	}
	state.permissions = map[string]string{}
	for _, selected := range decision.Reviewers {
		if !isCandidate(state.candidates, selected) {
			continue
		}
		permission, _, err := client.GetRepositoryPermissionLevel(context.Background(), state.repository, selected.Login)
		if err != nil {
			return nil, fmt.Errorf("Read reviewer permission: %w", err)
		}
		state.permissions[strings.ToLower(selected.Login)] = permission.GetPermission()
	}
	if len(decision.Reviewers) > 0 {
		state.reviews, err = client.ListReviews(context.Background(), state.repository, state.number)
		if err != nil {
			return nil, fmt.Errorf("Read existing pull request reviews: %w", err)
		}
	}
	return state, nil
}

// prepareAction checks authority under the PR lock and applies the GitHub
// state that loadAction read. It makes no provider calls. The saved PR
// revision is the commit fence while the transaction stays open.
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
	sha, _ := source.Report["reviewedCommit"].(string)
	state, _ := c.Loaded.(*pullRequestState)
	if state == nil || state.executionID != c.Execution.ID || state.repository != source.PullRequest.Repository || state.number != int(source.PullRequest.Number) || state.sha != sha {
		return false, fmt.Errorf("Review request pull request state does not match the source report")
	}
	if c.Execution.NodeID == selectionNodeID {
		return false, prepareSelectionAgent(c, source, state.candidates)
	}
	decision, err := loadDecision(c.Tx, run)
	if err != nil {
		return false, err
	}
	if decision.ReportID != source.Run.ID.String() || decision.SourceAppID != source.App.ID.String() || decision.ReviewedCommit != sha {
		return false, fmt.Errorf("Review request decision does not match its source report")
	}
	for _, selected := range decision.Reviewers {
		if !isCandidate(state.candidates, selected) {
			return false, fmt.Errorf("Review request reviewer is no longer eligible: %s", selected.Login)
		}
		permission, ok := state.permissions[strings.ToLower(selected.Login)]
		if !ok {
			return false, fmt.Errorf("Review request reviewer permission was not read: %s", selected.Login)
		}
		if !slices.Contains([]string{"write", "maintain", "admin"}, permission) {
			return false, fmt.Errorf("Review request reviewer requires repository write access: %s", selected.Login)
		}
	}
	toRequest := []string{}
	for _, selected := range decision.Reviewers {
		if slices.ContainsFunc(state.requested, func(user *github.User) bool { return strings.EqualFold(user.GetLogin(), selected.Login) }) || hasCurrentApproval(state.reviews, selected.Login, sha) {
			continue
		}
		toRequest = append(toRequest, selected.Login)
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
	configuration["repository"] = source.PullRequest.Repository
	configuration["pullNumber"] = int(source.PullRequest.Number)
	configuration["reviewers"] = toRequest
	configuration["teamReviewers"] = []string{}
	ctx.Configuration = configuration
	return false, nil
}

// loadDecision returns the unique saved selection decision of the run.
func loadDecision(tx *gorm.DB, run *models.CanvasRun) (*selectionDecision, error) {
	outputs, err := models.ListReviewRequestDecisionEvents(tx, run, selectionNodeID)
	if err != nil {
		return nil, err
	}
	if len(outputs) != 1 {
		return nil, fmt.Errorf("Review request has no unique validated decision")
	}
	envelope, _ := outputs[0].Data.Data().(map[string]any)
	data, _ := envelope["data"].(map[string]any)
	encoded, err := json.Marshal(data["reviewRequest"])
	if err != nil {
		return nil, err
	}
	var decision selectionDecision
	if err := json.Unmarshal(encoded, &decision); err != nil || decision.ReviewRequired == nil || decision.NoCandidate == nil {
		return nil, fmt.Errorf("Review request decision does not match its source report")
	}
	return &decision, nil
}

func pullNumber(value any) int {
	encoded, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	var number int
	if err := json.Unmarshal(encoded, &number); err != nil {
		return 0
	}
	return number
}

func isCandidate(candidates []*github.User, selected selectedReviewer) bool {
	return slices.ContainsFunc(candidates, func(user *github.User) bool {
		return strings.EqualFold(user.GetLogin(), selected.Login) && user.GetID() == selected.ID
	})
}

func githubClient(load lifecycle.ActionLoad, canvas *models.Canvas) (*githubcommon.Client, error) {
	node, err := models.FindCanvasNode(load.DB, canvas.ID, load.Execution.NodeID)
	if err != nil {
		return nil, err
	}
	var integration *models.Integration
	if node.AppInstallationID != nil {
		integration, err = models.FindUnscopedIntegrationInTransaction(load.DB, *node.AppInstallationID)
		if err != nil {
			return nil, err
		}
	} else {
		integration, err = workspaceIntegration(load.DB, canvas)
		if err != nil {
			return nil, err
		}
	}
	if load.Integration == nil {
		return nil, fmt.Errorf("Review request GitHub integration is unavailable")
	}
	return githubcommon.NewClient(load.Integration(integration), load.HTTP)
}

func workspaceIntegration(db *gorm.DB, canvas *models.Canvas) (*models.Integration, error) {
	workspace, err := models.FindFactory(db, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(workspace.OnboardingConfigValue().VCSIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("Review request requires the workspace GitHub integration")
	}
	integration, err := models.FindIntegrationInTransaction(db, canvas.OrganizationID, id)
	if err != nil {
		return nil, err
	}
	if integration.AppName != "github" || integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("Review request GitHub integration is unavailable")
	}
	return integration, nil
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
