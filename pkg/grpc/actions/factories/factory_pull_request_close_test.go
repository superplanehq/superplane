package factories

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestCloseWorkOrderAfterLineStepWaitsForImplement(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	newTask := func(t *testing.T, number int64, mergedAfterStep time.Duration) (*models.FactoryWorkOrder, *models.FactoryWorkOrderExecution) {
		t.Helper()
		order, err := factory.CreateWorkOrder(db, "Ship refunds in API and web", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen, Actor: &r.User, SkipSame: true})
		require.NoError(t, err)
		dispatch := support.CreateFactoryLineDispatch(t, factory.OrganizationID, factory.ID, order.ID, line.ID, line.Name, nil)
		execution := &models.FactoryWorkOrderExecution{
			ID: uuid.New(), OrganizationID: factory.OrganizationID, FactoryID: factory.ID, WorkOrderID: order.ID,
			LineID: line.ID, LineDispatchID: dispatch.ID, StepName: "Implement", Status: models.FactoryWorkOrderExecutionStatusRunning,
		}
		require.NoError(t, db.Create(execution).Error)
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider: models.FactoryPullRequestProviderGitHub, Repository: "acme/api", Number: number,
			URL: fmt.Sprintf("https://github.com/acme/api/pull/%d", number), State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		merged := models.FactoryPullRequestStateMerged
		mergedAt := dispatch.CreatedAt.Add(mergedAfterStep)
		require.NoError(t, pullRequest.Update(db, models.FactoryPullRequestPatch{State: &merged, MergedAt: &mergedAt}))
		return order, execution
	}
	finishStep := func(t *testing.T, execution *models.FactoryWorkOrderExecution) {
		t.Helper()
		require.NoError(t, db.Model(execution).Update("status", models.FactoryWorkOrderExecutionStatusFinished).Error)
	}
	reload := func(t *testing.T, order *models.FactoryWorkOrder) *models.FactoryWorkOrder {
		t.Helper()
		loaded, err := factory.FindWorkOrder(db, order.ID)
		require.NoError(t, err)
		return loaded
	}

	t.Run("a merge during Implement closes the task when the step ends", func(t *testing.T) {
		order, execution := newTask(t, 1, time.Minute)

		_, done, err := order.PullRequestCloseOutcome(db)
		require.NoError(t, err)
		assert.False(t, done, "Implement can still open a pull request in another repository")
		require.NoError(t, CloseWorkOrderAfterLineStep(db, factory, execution))
		assert.True(t, reload(t, order).IsOpen())

		finishStep(t, execution)
		review := &models.FactoryWorkOrderExecution{
			ID: uuid.New(), OrganizationID: factory.OrganizationID, FactoryID: factory.ID, WorkOrderID: order.ID,
			LineID: line.ID, LineDispatchID: execution.LineDispatchID, StepIndex: 1, StepName: "Review",
			Status: models.FactoryWorkOrderExecutionStatusRunning,
		}
		require.NoError(t, db.Create(review).Error)
		require.NoError(t, CloseWorkOrderAfterLineStep(db, factory, execution))
		assert.True(t, reload(t, order).IsOpen(), "the next line step is still running")

		finishStep(t, review)
		require.NoError(t, CloseWorkOrderAfterLineStep(db, factory, review))
		closed := reload(t, order)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, closed.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, closed.Result)
	})

	t.Run("a pull request merged before the step keeps a reopened task open", func(t *testing.T) {
		order, execution := newTask(t, 2, -time.Hour)
		finishStep(t, execution)

		require.NoError(t, CloseWorkOrderAfterLineStep(db, factory, execution))
		assert.True(t, reload(t, order).IsOpen())
	})
}
