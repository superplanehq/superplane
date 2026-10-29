package contexts

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__Build__OverlaysLineDispatchModelOnMatchingRunner(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6", "claude-opus-4-6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	require.NoError(t, db.Model(dispatch).Update("model", "claude-opus-4-6").Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model": "claude-sonnet-4-6",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-6", resolved["model"])
}

func TestBuildLineDispatchSubscriptionModel(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	connection, err := models.CreateFactoryCodexConnection(db, r.Organization.ID, factoryModel.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, connection.StoreUsage(db, nil, []models.CodexModel{{Model: "gpt-5.4"}}, time.Now()))
	require.NoError(t, db.Model(connection).Updates(map[string]any{"status": models.CodexConnectionStatusReady, "credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(db, r.Organization.ID, models.UsageProviderOpenAI, datatypes.JSONSlice[string]{"gpt-4.1"})
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerCodex")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factoryModel, order.ID, run.ID)
	execution, err := models.FindWorkOrderExecutionByRunID(db, run.ID)
	require.NoError(t, err)
	dispatch, err := models.FindWorkOrderLineDispatch(db, execution.LineDispatchID)
	require.NoError(t, err)

	for _, tc := range []struct {
		name     string
		model    string
		source   string
		expected string
	}{
		{"subscription catalog model", "gpt-5.4", "subscription", "gpt-5.4"},
		{"API key model excluded from subscription catalog", "gpt-4.1", "subscription", "gpt-5-codex"},
		{"subscription model excluded from API key allowlist", "gpt-5.4", "integration", "gpt-5-codex"},
		{"automatic model", "", "subscription", "gpt-5-codex"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, db.Model(dispatch).Update("model", tc.model).Error)
			resolved, err := NewNodeConfigurationBuilder(db, canvas.ID).
				WithNodeID("agent").WithRootEvent(&rootEvent.ID).
				Build(map[string]any{"model": "gpt-5-codex", "credentials": map[string]any{"source": tc.source}})
			if tc.source == "subscription" && tc.model == "gpt-4.1" {
				require.ErrorIs(t, err, models.ErrSubscriptionModelUnavailable)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, resolved["model"])
		})
	}
	t.Run("a locked task retries queue admission", func(t *testing.T) {
		require.NoError(t, db.Model(dispatch).Update("model", "gpt-5.4").Error)
		node, err := models.FindCanvasNode(db, canvas.ID, "agent")
		require.NoError(t, err)
		node.Configuration = datatypes.NewJSONType(map[string]any{"model": "gpt-5-codex", "credentials": map[string]any{"source": "subscription"}})
		locked := db.Begin()
		require.NoError(t, locked.Error)
		t.Cleanup(func() { locked.Rollback() })
		require.NoError(t, order.LockForUpdate(locked))
		_, err = BuildProcessQueueContext(nil, db, node, &models.CanvasNodeQueueItem{WorkflowID: canvas.ID, NodeID: "agent", RunID: run.ID, RootEventID: rootEvent.ID, EventID: rootEvent.ID}, nil, nil)
		var busy *models.CodexConnectionBusyError
		require.ErrorAs(t, err, &busy)
		var invalidConfiguration *ConfigurationBuildError
		assert.False(t, errors.As(err, &invalidConfiguration), "task contention must retry rather than create a failed execution")
	})

}

func Test__Build__OverlaysLineDispatchThinkingOnMatchingRunner(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	require.NoError(t, db.Model(dispatch).Update("thinking_level", "high").Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model":         "claude-sonnet-4-6",
		"thinkingLevel": "low",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "high", resolved["thinkingLevel"])
}

func Test__Build__KeepsCanvasThinkingWhenDispatchIsAuto(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factoryModel, order.ID, run.ID)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model":         "claude-sonnet-4-6",
		"thinkingLevel": "medium",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "medium", resolved["thinkingLevel"])
}

func Test__Build__OverlaysDispatchDefaultThinkingOntoEmpty(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	require.NoError(t, db.Model(dispatch).Update("thinking_level", "default").Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model":         "claude-sonnet-4-6",
		"thinkingLevel": "high",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "", resolved["thinkingLevel"])
}

func Test__Build__KeepsCanvasModelWhenDispatchIsAuto(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factoryModel, order.ID, run.ID)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model": "claude-sonnet-4-6",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", resolved["model"])
}

func Test__Build__SkipsOverrideWhenModelIsNotOnNodeProvider(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
	})

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	require.NoError(t, db.Model(dispatch).Update("model", "gpt-5").Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model": "claude-sonnet-4-6",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", resolved["model"])
}

func Test__Build__DoesNotOverlaySiblingCanvasModel(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	sibling := createFactoryRunnerApp(t, r, factoryModel.ID, "runnerClaudeCode", "sonnet")
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: canvas.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: sibling.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, line.Steps)
	require.NoError(t, db.Model(dispatch).Update("model", "sonnet").Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"model": "opus",
		"credentials": map[string]any{
			"source": "integration",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-5-5", resolved["model"])
}

func Test__Build__OverlaysLineDispatchModelOnSuperPlaneWithoutCanvasModel(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"x-ai/grok-4.6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, models.SuperPlaneRunnerComponent)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	override := models.FormatSelectableLLMModelKey(models.UsageFundingSourceHosted, models.UsageProviderOpenRouter, "x-ai/grok-4.6")
	require.NoError(t, db.Model(dispatch).Update("model", override).Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"machineType": "e1-large-amd64",
	})
	require.NoError(t, err)
	assert.Equal(t, override, resolved["model"])
}

func Test__Build__SkipsSuperPlaneOverrideWhenModelIsNotAllowlisted(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderOpenRouter,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"x-ai/grok-4.6"},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderOpenRouter).Delete(&models.HostedLLMProvider{})
	})

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, models.SuperPlaneRunnerComponent)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "ship", nil)
	require.NoError(t, err)
	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, order.ID, line.ID, line.Name, nil)
	require.NoError(t, db.Model(dispatch).Update("model", models.FormatSelectableLLMModelKey(
		models.UsageFundingSourceHosted,
		models.UsageProviderOpenRouter,
		"openai/gpt-5",
	)).Error)

	now := time.Now()
	require.NoError(t, db.Create(&models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "agent",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{})
	require.NoError(t, err)
	assert.NotContains(t, resolved, "model")
}

func Test__Build__KeepsSuperPlaneModelUnsetWhenDispatchIsAuto(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, rootEvent, run := setupRunnerAppExecution(t, r, factoryModel.ID, models.SuperPlaneRunnerComponent)
	order, err := factoryModel.CreateWorkOrder(db, "Ship it", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factoryModel, order.ID, run.ID)

	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	resolved, err := builder.Build(map[string]any{
		"machineType": "e1-large-amd64",
	})
	require.NoError(t, err)
	assert.NotContains(t, resolved, "model")
}

func setupRunnerAppExecution(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	component string,
) (*models.Canvas, *models.CanvasEvent, *models.CanvasRun) {
	t.Helper()
	const nodeID = "agent"
	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: nodeID,
				Name:   "agent",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: component},
				}),
			},
		},
		nil,
	)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryID).Error)
	canvas.FactoryID = &factoryID

	triggerEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, nodeID, "default", nil, map[string]any{"key": "value"})
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)
	return canvas, triggerEvent, run
}

func Test__Build__ResolvesClaudeAliasFromTheNodeSource(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.UpsertHostedLLMProvider(db, models.HostedLLMProvider{
		Provider:      models.UsageProviderAnthropic,
		Enabled:       true,
		APIKey:        []byte("encrypted"),
		AllowedModels: datatypes.JSONSlice[string]{"claude-sonnet-5"},
	})
	require.NoError(t, err)
	_, err = models.UpsertOrganizationBYOKModelAllowlist(
		db,
		r.Organization.ID,
		models.UsageProviderAnthropic,
		datatypes.JSONSlice[string]{"claude-sonnet-4-6"},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Where("provider = ?", models.UsageProviderAnthropic).Delete(&models.HostedLLMProvider{})
		_ = db.Where("organization_id = ? AND provider = ?", r.Organization.ID, models.UsageProviderAnthropic).
			Delete(&models.OrganizationBYOKModelAllowlist{})
	})

	canvas, rootEvent, _ := setupRunnerAppExecution(t, r, factoryModel.ID, "runnerClaudeCode")
	builder := NewNodeConfigurationBuilder(db, canvas.ID).
		WithNodeID("agent").
		WithRootEvent(&rootEvent.ID)

	byok, err := builder.Build(map[string]any{
		"model": "sonnet",
		"credentials": map[string]any{
			"source": "integration",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-6", byok["model"])

	hosted, err := builder.Build(map[string]any{
		"model": "sonnet",
		"credentials": map[string]any{
			"source": "hosted",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-5", hosted["model"])
}

func createFactoryRunnerApp(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	component string,
	model string,
) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           support.RandomName("app"),
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		return tx.Create(&models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{
				{
					ID:   "agent",
					Name: "agent",
					Type: models.NodeTypeComponent,
					Ref:  models.NodeRef{Component: &models.ComponentRef{Name: component}},
					Configuration: map[string]any{
						"model": model,
						"credentials": map[string]any{
							"source": "integration",
						},
					},
				},
			}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}).Error
	}))
	return canvas
}
