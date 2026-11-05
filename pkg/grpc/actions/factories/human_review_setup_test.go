package factories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/factories/mergeconfidence"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestHumanReviewSetupFailureBlocksTheGateAndRecoversOnRetry(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("human-review"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Review", "", &r.User, nil, nil)
	require.NoError(t, err)
	pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	_, err = pr.ObserveRevision(db, "current-head")
	require.NoError(t, err)
	assessmentCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{NodeID: "on-pr-risk", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1))},
		{NodeID: "assess-risk", Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}), Configuration: datatypes.NewJSONType(map[string]any{"environment": []any{map[string]any{"name": "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID", "value": order.ID.String()}}})},
	}, nil)
	require.NoError(t, db.Model(assessmentCanvas).Update("factory_id", factory.ID).Error)
	root := &models.CanvasEvent{WorkflowID: assessmentCanvas.ID, NodeID: "on-pr-risk", Data: models.NewJSONValue(map[string]any{"type": "github.pullRequest", "data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "current-head"}}}})}
	require.NoError(t, db.Create(root).Error)
	require.NoError(t, pr.LinkRun(db, root.RunID, "Merge confidence"))
	execution := support.CreateCanvasNodeExecution(t, assessmentCanvas.ID, "assess-risk", root.ID, root.ID)
	score := 5.0
	report := mergeconfidence.ReportData{
		PullRequestID: pr.ID.String(), Complete: true, ReviewedCommit: "current-head", OverallScore: &score, OverallLevel: "positive",
		Checks: []mergeconfidence.ReportCheck{{ID: "security", Score: 5, MaxScore: 5, Summary: "No security issue found.", Level: "positive"}},
	}
	require.NoError(t, db.Create(&models.CanvasEvent{
		WorkflowID: assessmentCanvas.ID, NodeID: "assess-risk", ExecutionID: &execution.ID, Channel: "passed",
		Data: models.NewJSONValue(map[string]any{"type": "runnerClaudeCode.finished", "data": map[string]any{"mergeConfidence": report}}),
	}).Error)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", root.RunID).Updates(map[string]any{"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultPassed}).Error)
	humanCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
		NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger,
		Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}),
		Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"}),
		Metadata:      datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 1)),
	}}, nil)
	require.NoError(t, db.Model(humanCanvas).Update("factory_id", factory.ID).Error)
	automation, err := findHumanReviewAutomation(db, factory)
	require.NoError(t, err)
	require.NotNil(t, automation)
	setupFailure := errors.New("changed-file lookup is temporarily unavailable")
	client := &humanReviewSetupFake{
		humanReviewLifecycleFake: humanReviewLifecycleFake{current: &github.PullRequest{State: github.Ptr("open"), User: reviewTestUser(1, "author", "User"), Head: &github.PullRequestBranch{SHA: github.Ptr("current-head")}}},
		setupFailure:             setupFailure,
	}
	require.ErrorIs(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client), setupFailure)
	failed, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.Equal(t, models.FactoryHumanReviewStateFailed, failed.State)
	require.Nil(t, failed.RunID)
	require.Contains(t, failed.BlockingReason, setupFailure.Error())
	require.Equal(t, humanReviewPublication{Status: "completed", Conclusion: "failure"}, client.publications[len(client.publications)-1])
	client.setupFailure = nil
	require.NoError(t, reconcileHumanReview(t.Context(), db, IntakeDependencies{}, factory, pr, automation, client))
	recovered, err := pr.CurrentHumanReview(db)
	require.NoError(t, err)
	require.Equal(t, failed.ID, recovered.ID)
	require.Equal(t, models.FactoryHumanReviewStateWaitingDecision, recovered.State)
	require.NotNil(t, recovered.RunID)
	require.Equal(t, humanReviewPublication{Status: "in_progress"}, client.publications[len(client.publications)-1])
}

type humanReviewSetupFake struct {
	humanReviewLifecycleFake
	setupFailure error
	publications []humanReviewPublication
}

func (f *humanReviewSetupFake) HumanReviewChangedFiles(context.Context, string, int) ([]*github.CommitFile, error) {
	return nil, f.setupFailure
}

func (f *humanReviewSetupFake) CreateHumanReviewCheck(ctx context.Context, repository string, options github.CreateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	f.publications = append(f.publications, humanReviewPublication{Status: options.GetStatus(), Conclusion: options.GetConclusion()})
	return f.humanReviewLifecycleFake.CreateHumanReviewCheck(ctx, repository, options)
}

func (f *humanReviewSetupFake) UpdateHumanReviewCheck(ctx context.Context, repository string, id int64, options github.UpdateCheckRunOptions) (*github.CheckRun, *github.Response, error) {
	f.publications = append(f.publications, humanReviewPublication{Status: options.GetStatus(), Conclusion: options.GetConclusion()})
	return f.humanReviewLifecycleFake.UpdateHumanReviewCheck(ctx, repository, id, options)
}
