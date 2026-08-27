package models_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RecordUsage__FactoryLinkedRunPersistsAndRollsUp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()

	err := models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		OutputTokens:    0,
		TotalTokens:     1_000_000,
	})
	require.NoError(t, err)

	assertInProgressExecutionUsage(t, db, execution.ID, 1_000_000, 300)

	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
		FactoryID:      &execution.FactoryID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), totals.TotalTokens)
	assert.Equal(t, int64(300), totals.CostCents())
	require.Len(t, byModel, 1)
	assert.Equal(t, "anthropic", byModel[0].Provider)
	assert.Equal(t, "claude-sonnet-4-6", byModel[0].Model)
}

func Test__RecordUsage__SameNodeExecutionRecordsEachBilledCall(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()

	first := models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}
	require.NoError(t, models.RecordUsage(db, first))
	require.NoError(t, models.RecordUsage(db, first))

	assertInProgressExecutionUsage(t, db, execution.ID, 2_000_000, 600)
}

func Test__RecordUsage__ConcurrentCallsKeepFullSpend(t *testing.T) {
	r := support.Setup(t)
	execution := dispatchWorkOrderExecution(t, r)

	const calls = 20
	var wg sync.WaitGroup
	errs := make(chan error, calls)
	for range calls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- models.RecordUsage(database.Conn(), models.LLMUsageEventInput{
				OrganizationID:  r.Organization.ID,
				CanvasRunID:     requireExecutionRunID(t, execution),
				NodeExecutionID: uuid.New(),
				NodeID:          "prompt",
				Provider:        models.UsageProviderAnthropic,
				Model:           "claude-sonnet-4-6",
				InputTokens:     1_000_000,
				TotalTokens:     1_000_000,
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	assertInProgressExecutionUsage(t, database.Conn(), execution.ID, int64(calls)*1_000_000, int64(calls)*300)
}

func Test__RecordUsage__ChildRunUsesParentFactoryExecution(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	parentRun, err := models.FindUnscopedCanvasRun(db, requireExecutionRunID(t, execution))
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
	require.NoError(t, db.Create(&childRun).Error)

	err = models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     childRun.ID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     100,
		OutputTokens:    20,
		TotalTokens:     120,
	})
	require.NoError(t, err)

	assertInProgressExecutionUsage(t, db, execution.ID, 120, 0)
}

func Test__RecordUsage__SkipsRunsWithoutWorkOrderExecution(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	err := models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     uuid.New(),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     100,
		OutputTokens:    20,
		TotalTokens:     120,
	})
	require.NoError(t, err)

	totals, byModel, err := models.SummarizeUsage(db, models.UsageReportFilter{
		OrganizationID: r.Organization.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), totals.TotalTokens)
	assert.Empty(t, byModel)
}

func Test__RecordUsage__SameIdempotencyKeyRecordsOnce(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	nodeExecutionID := uuid.New()
	key := models.UsageIdempotencyKeyRunner + ":" + nodeExecutionID.String()

	input := models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: nodeExecutionID,
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
		IdempotencyKey:  key,
	}
	require.NoError(t, models.RecordUsage(db, input))
	require.NoError(t, models.RecordUsage(db, input))

	var count int64
	require.NoError(t, db.Model(&models.LLMUsageEvent{}).Where("idempotency_key = ?", key).Count(&count).Error)
	assert.Equal(t, int64(1), count)
	assertInProgressExecutionUsage(t, db, execution.ID, 1_000_000, 300)
}

func Test__RecordUsage__HostedUnknownModelFailsClosed(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)

	err := models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     requireExecutionRunID(t, execution),
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenRouter,
		Model:           "unknown-hosted-model",
		InputTokens:     100,
		TotalTokens:     100,
		FundingSource:   models.UsageFundingSourceHosted,
	})
	require.ErrorIs(t, err, models.ErrHostedUsageUnpriced)
}

func assertInProgressExecutionUsage(t *testing.T, db *gorm.DB, executionID uuid.UUID, tokens, cents int64) {
	t.Helper()

	var updated models.FactoryWorkOrderExecution
	require.NoError(t, db.First(&updated, "id = ?", executionID).Error)
	assert.NotEqual(t, models.FactoryWorkOrderExecutionStatusFinished, updated.Status)
	assert.Equal(t, tokens, updated.TotalTokens)
	assert.Equal(t, cents, updated.CostCents)
}

func requireExecutionRunID(t *testing.T, execution *models.FactoryWorkOrderExecution) uuid.UUID {
	t.Helper()
	require.NotNil(t, execution.RunID)
	return *execution.RunID
}

func dispatchWorkOrderExecution(t *testing.T, r *support.ResourceRegistry) *models.FactoryWorkOrderExecution {
	t.Helper()
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
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
