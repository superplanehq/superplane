package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/superplanehq/superplane/pkg/database"
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

func Test__Build__OverlaysCanvasModelFromSameProviderOnTheLine(t *testing.T) {
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
	assert.Equal(t, "sonnet", resolved["model"])
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
