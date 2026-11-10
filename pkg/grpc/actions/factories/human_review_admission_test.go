package factories

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestPrepareHumanReviewRejectsMissingSubscriptionOwnerBeforeAgentAdmission(t *testing.T) {
	f := newHumanReviewReconciliationFixture(t)
	order, err := f.factory.FindWorkOrder(f.db, f.pr.WorkOrderID)
	require.NoError(t, err)
	_, err = order.UpdateStatus(f.db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	subscription := true
	require.NoError(t, f.factory.UpdateOnboarding(f.db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	root := &models.CanvasEvent{
		WorkflowID: f.automation.canvas.ID, NodeID: models.HumanReviewTriggerNodeID,
		Data: models.NewJSONValue(map[string]any{
			"type": "app.invocation", "data": map[string]any{"parameters": map[string]any{"pullRequestId": f.pr.ID.String()}},
		}),
	}
	require.NoError(t, f.db.Create(root).Error)
	run, err := models.FindCanvasRunInTransaction(f.db, f.automation.canvas.ID, root.RunID)
	require.NoError(t, err)

	prepared, err := prepareHumanReview(t.Context(), f.db, f.factory, f.pr, f.automation, f.client, run, root)
	require.NoError(t, err)
	require.False(t, prepared.AgentNeeded)
	decision, err := f.pr.CurrentHumanReview(f.db)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.Equal(t, models.FactoryHumanReviewStateFailed, decision.State)
	require.Equal(t, models.ErrSubscriptionOwnerRequired.Error(), decision.BlockingReason)
	require.Nil(t, decision.RunID)
	_, err = models.FindPullRequestActivityByRunID(f.db, run.ID)
	require.ErrorIs(t, err, models.ErrFactoryPullRequestActivityNotFound)
	order, err = f.factory.FindWorkOrder(f.db, f.pr.WorkOrderID)
	require.NoError(t, err)
	notes, err := order.StatusNotes()
	require.NoError(t, err)
	require.Len(t, notes, 1)
	require.Contains(t, notes[0].Body, models.ErrSubscriptionOwnerRequired.Error())
}
