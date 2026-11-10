package contexts

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// These cases exercise the runner result boundary against durable decisions.
// They catch a late completion restoring enforcement after revision or policy changes.
func TestHumanReviewCompletionAuthority(t *testing.T) {
	for _, scenario := range []string{"valid decision", "missing output", "new revision", "disabled policy", "changed configuration"} {
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
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
				{NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger, Ref: datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}}), Metadata: datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 1)), Configuration: datatypes.NewJSONType(map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"})},
				{NodeID: models.HumanReviewAgentNodeID, Type: models.NodeTypeComponent, Ref: datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: "runnerClaudeCode"}})},
			}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", workspace.ID).Error)
			event := support.EmitCanvasEventForNode(t, canvas.ID, models.HumanReviewTriggerNodeID, "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, models.HumanReviewAgentNodeID, event.ID, event.ID)
			require.NoError(t, db.Model(execution).Update("state", models.CanvasNodeExecutionStateStarted).Error)
			execution.State = models.CanvasNodeExecutionStateStarted
			decision, err := pr.NewHumanReviewDecision(db, models.FactoryHumanReviewDecision{RevisionID: observed.Revision.ID, RunID: &event.RunID, AssessedCommit: "head", State: models.FactoryHumanReviewStateWaitingDecision, ConfigurationVersion: canvas.LiveVersionID.String()})
			require.NoError(t, err)
			version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
			require.NoError(t, err)
			switch scenario {
			case "new revision":
				_, err = pr.ObserveRevision(db, "next")
				require.NoError(t, err)
			case "disabled policy":
				version.Nodes[0].Configuration["enabled"] = false
				require.NoError(t, db.Model(version).Update("nodes", version.Nodes).Error)
			case "changed configuration":
				decision.ConfigurationVersion = "old-policy"
				require.NoError(t, decision.Save(db))
			}
			payload := map[string]any{"result": map[string]any{"humanReview": map[string]any{"assessedCommit": "head", "reviewRequired": false, "rationale": "The complete assessment is favorable.", "selectedReviewers": []any{}}}}
			if scenario == "missing output" {
				payload = map[string]any{"result": map[string]any{}}
			}
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return NewExecutionStateContext(tx, execution, nil).Emit("passed", "runnerClaudeCode.finished", []any{payload})
			}))
			saved, err := models.FindFactoryHumanReviewDecision(db, decision.ID)
			require.NoError(t, err)
			switch scenario {
			case "valid decision":
				require.Equal(t, models.FactoryHumanReviewStateDecided, saved.State)
				require.NotEmpty(t, saved.Result)
			case "missing output":
				require.Equal(t, models.FactoryHumanReviewStateFailed, saved.State)
				require.Empty(t, saved.Result)
			case "new revision":
				require.Equal(t, models.FactoryHumanReviewStateSuperseded, saved.State)
				require.Empty(t, saved.Result)
			default:
				require.Empty(t, saved.Result)
			}
		})
	}
}
