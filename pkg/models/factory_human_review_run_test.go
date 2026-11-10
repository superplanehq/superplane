package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Completion must keep the authority granted by preparation. A late result
// cannot restore enforcement after its revision, assessment, or policy changes.
func TestHumanReviewCompletionAuthority(t *testing.T) {
	for _, scenario := range []string{
		"valid decision", "missing output", "wrong assessed commit", "failed agent",
		"new revision", "new assessment", "disabled policy", "changed configuration",
		"cancelled run", "cancelled agent", "wrong agent", "wrong completion component",
		"foreign completion", "tampered completion", "multiple completions", "incomplete assessment",
	} {
		t.Run(scenario, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			workspace, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("review"), "", "")
			require.NoError(t, err)
			order, err := workspace.CreateWorkOrder(db, "Review task", "", &r.User, nil, nil)
			require.NoError(t, err)
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
			require.NoError(t, err)
			observed, err := pr.ObserveRevision(db, "head")
			require.NoError(t, err)
			assessmentCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: "on-assessment", Type: models.NodeTypeTrigger, Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata("risk-score", 1))},
				{NodeID: "assess-risk", Type: models.NodeTypeComponent,
					Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}}),
					Configuration: datatypes.NewJSONType(map[string]any{"environment": []any{map[string]any{"name": "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID", "value": order.ID.String()}}})},
			}, nil)
			require.NoError(t, db.Model(assessmentCanvas).Update("factory_id", workspace.ID).Error)
			source := emitHumanReviewAssessment(t, db, pr, assessmentCanvas)
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger,
					Ref:      datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}),
					Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 2))},
				{NodeID: models.HumanReviewPrepareNodeID, Type: models.NodeTypeComponent,
					Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "prepareHumanReview"}}),
					Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})},
				{NodeID: models.HumanReviewAgentNodeID, Type: models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
				{NodeID: "unrelated-agent", Type: models.NodeTypeComponent,
					Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
			}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
			canvas.FactoryID = &workspace.ID
			root := support.EmitCanvasEventForNode(t, canvas.ID, models.HumanReviewTriggerNodeID, "default", nil)
			agentNodeID := models.HumanReviewAgentNodeID
			if scenario == "wrong agent" {
				agentNodeID = "unrelated-agent"
			}
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, agentNodeID, root.ID, root.ID)
			require.NoError(t, execution.StartInTransaction(db))
			decision, err := pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{
				RevisionID: observed.Revision.ID, RunID: &root.RunID, SourceRunID: &source.ID,
				AssessedCommit: "head", State: models.FactoryHumanReviewStateWaitingDecision,
				ConfigurationVersion: canvas.LiveVersionID.String(),
			})
			require.NoError(t, err)
			switch scenario {
			case "new revision":
				_, err = pr.ObserveRevision(db, "next")
				require.NoError(t, err)
			case "new assessment":
				emitHumanReviewAssessment(t, db, pr, assessmentCanvas)
			case "disabled policy":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.Nodes[1].Configuration["enabled"] = false
				require.NoError(t, db.Model(version).Update("nodes", version.Nodes).Error)
			case "changed configuration":
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
				require.NoError(t, err)
				version.ID = uuid.New()
				require.NoError(t, db.Create(version).Error)
				require.NoError(t, db.Model(canvas).Update("live_version_id", version.ID).Error)
			case "cancelled run":
				require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", root.RunID).Update("cancelled_at", time.Now()).Error)
			case "incomplete assessment":
				require.NoError(t, db.Model(&models.CanvasEvent{}).Where("run_id = ? AND execution_id IS NOT NULL", source.ID).
					Update("data", gorm.Expr("jsonb_set(data, '{data,mergeConfidence,complete}', 'false'::jsonb)")).Error)
			}
			output := map[string]any{"assessedCommit": "head", "reviewRequired": false, "rationale": "The complete assessment is favorable.", "selectedReviewers": []any{}}
			payload := map[string]any{"result": map[string]any{"humanReview": output}}
			channel, payloadType := "passed", "runnerClaudeCode.finished"
			switch scenario {
			case "missing output", "tampered completion":
				payload = map[string]any{"result": map[string]any{}}
			case "wrong assessed commit":
				output["assessedCommit"] = "next"
			case "failed agent":
				channel = "failed"
			case "wrong completion component":
				payloadType = "runnerCodex.finished"
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				completion := map[string]any{"type": payloadType, "data": payload}
				outputs := []any{completion}
				if scenario == "multiple completions" {
					outputs = append(outputs, completion)
				}
				events, err := execution.PassInTransaction(tx, map[string][]any{channel: outputs})
				if err != nil {
					return err
				}
				require.Len(t, events, len(outputs))
				if scenario == "cancelled agent" {
					// The caller still holds the successful completion snapshot.
					if err := tx.Model(&models.CanvasNodeExecution{}).Where("id = ?", execution.ID).Update("result", models.CanvasNodeExecutionResultCancelled).Error; err != nil {
						return err
					}
				}
				if scenario == "foreign completion" {
					foreign, err := models.HumanReviewCompletionEvents(tx, source.ID, "assess-risk")
					if err != nil {
						return err
					}
					require.Len(t, foreign, 1)
					events[0] = foreign[0]
				}
				if scenario == "tampered completion" {
					events[0].Data = models.NewJSONValue(map[string]any{"type": payloadType, "data": map[string]any{"result": map[string]any{"humanReview": output}}})
				}
				_, err = models.RecordHumanReviewDecision(tx, canvas, execution, &events[0])
				return err
			}))
			saved, err := models.FindFactoryHumanReviewDecision(db, decision.ID)
			require.NoError(t, err)
			switch scenario {
			case "valid decision":
				require.Equal(t, models.FactoryHumanReviewStateDecided, saved.State)
				require.False(t, saved.ReviewRequired)
				require.NotEmpty(t, saved.Result)
			case "missing output", "wrong assessed commit", "failed agent", "tampered completion", "multiple completions":
				require.Equal(t, models.FactoryHumanReviewStateFailed, saved.State)
				require.Empty(t, saved.Result)
				run, err := models.FindCanvasRunInTransaction(db, canvas.ID, root.RunID)
				require.NoError(t, err)
				require.Len(t, run.Errors, 1)
			case "new revision":
				require.Equal(t, models.FactoryHumanReviewStateSuperseded, saved.State)
				require.Empty(t, saved.Result)
			default:
				require.Equal(t, models.FactoryHumanReviewStateWaitingDecision, saved.State)
				require.Empty(t, saved.Result)
			}
		})
	}
}

func emitHumanReviewAssessment(t *testing.T, db *gorm.DB, pr *models.FactoryPullRequest, canvas *models.Canvas) *models.CanvasRun {
	t.Helper()
	root := support.EmitCanvasEventForNodeWithData(t, canvas.ID, "on-assessment", "default", nil, map[string]any{
		"type": "github.pullRequest", "data": map[string]any{"pull_request": map[string]any{"head": map[string]any{"sha": "head"}}},
	})
	require.NoError(t, pr.LinkRun(db, root.RunID, "Merge confidence"))
	execution := support.CreateCanvasNodeExecution(t, canvas.ID, "assess-risk", root.ID, root.ID)
	_, err := execution.PassInTransaction(db, map[string][]any{"passed": {map[string]any{
		"type": "runnerClaudeCode.finished", "data": map[string]any{"mergeConfidence": map[string]any{
			"pullRequestId": pr.ID.String(), "complete": true, "reviewedCommit": "head",
			"overallScore": 5, "overallLevel": "positive", "checks": []any{map[string]any{
				"id": "security", "score": 5, "maxScore": 5, "summary": "No security issue found.", "level": "positive",
			}},
		}},
	}}})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", root.RunID).Updates(map[string]any{
		"state": models.CanvasRunStateFinished, "result": models.CanvasRunResultPassed,
	}).Error)
	source, err := models.FindCanvasRunInTransaction(db, canvas.ID, root.RunID)
	require.NoError(t, err)
	return source
}
