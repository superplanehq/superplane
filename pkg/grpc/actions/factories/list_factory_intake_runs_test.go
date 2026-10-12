package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func TestIntakeRunTitle_JiraUsesKeyWhenSummaryIsMissing(t *testing.T) {
	t.Run("key and summary", func(t *testing.T) {
		event := models.CanvasEvent{Data: models.NewJSONValue(map[string]any{
			"type": "jira.issue",
			"data": map[string]any{
				"issue": map[string]any{
					"key":    "ENG-42",
					"fields": map[string]any{"summary": "Login fails"},
				},
			},
		})}
		assert.Equal(t, "ENG-42: Login fails", intakeRunTitle(models.FactoryIntakeSourceJiraIssues, event))
	})

	t.Run("key only", func(t *testing.T) {
		event := models.CanvasEvent{Data: models.NewJSONValue(map[string]any{
			"type": "jira.issue",
			"data": map[string]any{
				"issue": map[string]any{"key": "ENG-42", "fields": map[string]any{}},
			},
		})}
		assert.Equal(t, "ENG-42", intakeRunTitle(models.FactoryIntakeSourceJiraIssues, event))
	})
}

func TestIntakeRunTitle_LinearUsesIdentifierAndTitle(t *testing.T) {
	event := models.CanvasEvent{Data: models.NewJSONValue(map[string]any{
		"type": "linear.issue",
		"data": map[string]any{
			"url": "https://linear.app/acme/issue/ENG-142/deploy-pipeline-fails",
			"data": map[string]any{
				"identifier": "ENG-142",
				"title":      "Deploy pipeline fails on retry",
			},
		},
	})}
	assert.Equal(t, "ENG-142: Deploy pipeline fails on retry", intakeRunTitle(models.FactoryIntakeSourceLinearIssues, event))
}

func Test__IntakeRunPlacement(t *testing.T) {
	runID := uuid.New()
	run := models.CanvasRun{ID: runID}

	t.Run("a create-only graph waits while the work order is still opening", func(t *testing.T) {
		placement := intakeRunPlacement(run, intakeRunContext{
			graph: intakeGraph{CreateNodeID: intakeCreateNodeID},
		})
		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_ANALYZING, placement)
	})

	t.Run("a create-only graph does not emit below threshold", func(t *testing.T) {
		placement := intakeRunPlacement(run, intakeRunContext{
			graph: intakeGraph{CreateNodeID: intakeCreateNodeID},
			creations: map[uuid.UUID]models.CanvasNodeExecution{
				runID: {
					State:  models.CanvasNodeExecutionStateFinished,
					Result: models.CanvasNodeExecutionResultFailed,
				},
			},
		})
		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_REJECTED, placement)
	})

	t.Run("a finished create lands in Backlog", func(t *testing.T) {
		placement := intakeRunPlacement(run, intakeRunContext{
			graph: intakeGraph{CreateNodeID: intakeCreateNodeID},
			creations: map[uuid.UUID]models.CanvasNodeExecution{
				runID: {
					State:  models.CanvasNodeExecutionStateFinished,
					Result: models.CanvasNodeExecutionResultPassed,
				},
			},
		})
		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_BACKLOG, placement)
	})

	t.Run("a legacy analysis graph still reports below threshold", func(t *testing.T) {
		placement := intakeRunPlacement(run, intakeRunContext{
			graph: intakeGraph{AnalysisNodeID: intakeAnalysisNodeID, CreateNodeID: intakeCreateNodeID},
			analyses: map[uuid.UUID]models.CanvasNodeExecution{
				runID: {
					State:  models.CanvasNodeExecutionStateFinished,
					Result: models.CanvasNodeExecutionResultPassed,
				},
			},
		})
		assert.Equal(t, pb.FactoryIntakeRun_PLACEMENT_BELOW_THRESHOLD, placement)
	})
}
