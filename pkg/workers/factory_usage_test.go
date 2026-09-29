package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__RollUpFactoryUsage__FillsFinishedExecutionCache(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	execution := dispatchFactoryExecutionForUsageTest(t, r)
	runID := requireFactoryExecutionRunID(t, execution)
	recordFactoryLLMUsage(t, r.Organization.ID, runID)
	require.NoError(t, execution.MarkFinished(database.Conn(), models.CanvasRunResultPassed))

	require.NoError(t, rollUpFactoryUsage(database.Conn(), runID))

	updated, err := models.FindWorkOrderExecutionByRunID(database.Conn(), runID)
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), updated.TotalTokens)
	assert.Equal(t, int64(300), updated.CostCents)
}

func Test__RollUpFactoryUsage__FillsCacheFromChildRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	execution := dispatchFactoryExecutionForUsageTest(t, r)
	parentRun, err := models.FindUnscopedCanvasRun(database.Conn(), requireFactoryExecutionRunID(t, execution))
	require.NoError(t, err)

	now := parentRun.CreatedAt
	if now == nil {
		created := time.Now()
		now = &created
	}
	childRun := models.CanvasRun{
		ID:               uuid.New(),
		WorkflowID:       parentRun.WorkflowID,
		NodeID:           parentRun.NodeID,
		VersionID:        parentRun.VersionID,
		ParentRunID:      &parentRun.ID,
		ParentWorkflowID: &parentRun.WorkflowID,
		State:            models.CanvasRunStateStarted,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	require.NoError(t, database.Conn().Create(&childRun).Error)

	recordFactoryLLMUsage(t, r.Organization.ID, childRun.ID)
	require.NoError(t, execution.MarkFinished(database.Conn(), models.CanvasRunResultPassed))

	require.NoError(t, rollUpFactoryUsage(database.Conn(), childRun.ID))

	updated, err := models.FindWorkOrderExecutionByRunID(database.Conn(), requireFactoryExecutionRunID(t, execution))
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), updated.TotalTokens)
	assert.Equal(t, int64(300), updated.CostCents)
}

func Test__RollUpFactoryUsage__IgnoresRunsWithoutFactoryExecution(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	require.NoError(t, rollUpFactoryUsage(database.Conn(), uuid.New()))
}

func requireFactoryExecutionRunID(t *testing.T, execution *models.FactoryWorkOrderExecution) uuid.UUID {
	t.Helper()
	require.NotNil(t, execution.RunID)
	return *execution.RunID
}

func recordFactoryLLMUsage(t *testing.T, organizationID, runID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.RecordUsage(database.Conn(), models.WorkspaceUsageEventInput{
		OrganizationID:  organizationID,
		CanvasRunID:     runID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		OutputTokens:    0,
		TotalTokens:     1_000_000,
	}))
}

func clearFactoryExecutionUsageCache(t *testing.T, executionID uuid.UUID) {
	t.Helper()
	require.NoError(t, database.Conn().Model(&models.FactoryWorkOrderExecution{}).
		Where("id = ?", executionID).
		Updates(map[string]any{
			"total_tokens": 0,
			"cost_cents":   0,
		}).Error)
}

func dispatchFactoryExecutionForUsageTest(t *testing.T, r *support.ResourceRegistry) *models.FactoryWorkOrderExecution {
	t.Helper()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(database.Conn(), "Ship feature", "", &r.User, nil, nil)
	require.NoError(t, err)
	dispatchWorkOrderForTest(t, order)

	line, err := factory.CreateLine(database.Conn(), "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(database.Conn(), nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))
	require.NotNil(t, execution)
	return execution
}
