package mcpserver

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func enableFactories(t *testing.T, orgID uuid.UUID) {
	t.Helper()
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureFactories))
	require.NoError(t, models.EnableExperimentalFeature(orgID, features.FeatureSuperPlaneMCPServer))
}

func toolClaims(r *support.ResourceRegistry, factoryID uuid.UUID) *AccessClaims {
	return &AccessClaims{
		UserID:    r.User,
		OrgID:     r.Organization.ID,
		FactoryID: factoryID,
		Resource:  "http://localhost:8000/mcp",
		Scopes:    append([]string{}, GrantedScopes...),
	}
}

func decodeToolJSON(t *testing.T, result ToolResult) map[string]any {
	t.Helper()
	require.False(t, result.IsError)
	require.NotEmpty(t, result.Content)
	text, _ := result.Content[0]["text"].(string)
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &payload))
	return payload
}

func TestListTasksAppliesMineAndStates(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP List", "", "LST")
	require.NoError(t, err)
	other := support.CreateUser(t, r, r.Organization.ID)

	draftMine, err := factoryModel.CreateWorkOrder(db, "Draft mine", "", &r.User, nil, nil)
	require.NoError(t, err)
	openMine, err := factoryModel.CreateWorkOrder(db, "Open mine", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = openMine.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, err = factoryModel.CreateWorkOrder(db, "Draft other", "", &other.ID, nil, nil)
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "list_tasks", map[string]any{
		"mine":   true,
		"states": []any{"draft"},
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	tasks, ok := payload["tasks"].([]any)
	require.True(t, ok)
	require.Len(t, tasks, 1)
	row, ok := tasks[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, draftMine.ID.String(), row["id"])
	assert.Equal(t, "draft", row["state"])
}

func TestGetTaskRejectsOtherWorkspace(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryA, err := models.CreateFactory(db, r.Organization.ID, "Workspace A", "", "WSA")
	require.NoError(t, err)
	factoryB, err := models.CreateFactory(db, r.Organization.ID, "Workspace B", "", "WSB")
	require.NoError(t, err)
	orderB, err := factoryB.CreateWorkOrder(db, "Secret", "", &r.User, nil, nil)
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryA.ID), "get_task", map[string]any{"task": orderB.ID.String()})
	require.Error(t, err)
	assert.Equal(t, "Not found", err.Error())
}

func TestCreateTaskReturnsNewTask(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Create", "", "CRT")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":       "Ship MCP",
		"description": "Expose the workspace tools.",
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "Ship MCP", payload["title"])
	assert.Equal(t, "draft", payload["state"])
	require.NotEmpty(t, payload["id"])
}

func TestCallToolRejectsWithoutMCPServerFlag(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "No MCP", "", "NMC")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "list_tasks", map[string]any{})
	require.Error(t, err)
	assert.Equal(t, "Not found", err.Error())
}

func TestListTaskArtifactsIncludesDownloadURL(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Artifacts", "", "ART")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Has file", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.CreateArtifact(db, models.FactoryWorkOrderArtifactParams{
		Type:      models.FactoryWorkOrderArtifactTypeFile,
		CreatedBy: &r.User,
		Data: map[string]any{
			"fileId":      uuid.New().String(),
			"filename":    "notes.png",
			"contentType": "image/png",
			"title":       "Notes",
			"sizeBytes":   12,
			"url":         "http://localhost:8000/api/v1/public/artifacts/" + uuid.New().String() + "/notes.png",
		},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "list_task_artifacts", map[string]any{
		"task": order.ID.String(),
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	artifacts, ok := payload["artifacts"].([]any)
	require.True(t, ok)
	require.Len(t, artifacts, 1)
	row, ok := artifacts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "file", row["type"])
	assert.Equal(t, "notes.png", row["filename"])
	assert.Contains(t, row["url"], "/api/v1/public/artifacts/")
}

func TestGetTaskAgentWithoutSessionReturnsToolError(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Agent", "", "AGT")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "No agent", "", &r.User, nil, nil)
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "get_task_agent", map[string]any{"task": order.ID.String()})
	require.Error(t, err)
	assert.Equal(t, "This task has no agent session.", err.Error())

	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "send_task_message", map[string]any{
		"task":    order.ID.String(),
		"message": "Hello",
	})
	require.Error(t, err)
	assert.Equal(t, "This task has no agent session.", err.Error())
}

func TestSendTaskMessageStoresMessage(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)
	session := openAnalysisSessionForMCP(t, r, db)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, session.FactoryID), "send_task_message", map[string]any{
		"task":    session.DraftWorkOrderID.String(),
		"message": "Use the current retry form.",
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "stored", payload["status"])
	assert.Equal(t, session.ID.String(), payload["session_id"])
}

func openAnalysisSessionForMCP(t *testing.T, r *support.ResourceRegistry, db *gorm.DB) *models.FactoryPlanningSession {
	t.Helper()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Retry refunds", "Stop double charges.", &r.User, nil, nil)
	require.NoError(t, err)
	canvas := createOnWorkOrderCanvasForMCP(t, r, factoryModel.ID)
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, "start", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	session, err := factoryModel.AttachAnalysisSession(db, models.AttachAnalysisSessionParams{
		Repository:  "acme/payments",
		CanvasID:    canvas.ID,
		CanvasRunID: run.ID,
		WorkOrderID: order.ID,
	})
	require.NoError(t, err)
	return session
}

func createOnWorkOrderCanvasForMCP(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           "Backlog",
		CreatedBy:      &r.User,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	require.NoError(t, database.DB(t.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}
		node := models.CanvasNode{
			WorkflowID: canvas.ID,
			NodeID:     "start",
			Name:       "On Task",
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName},
			}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}
		version := models.CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			OwnerID:    &r.User,
			Nodes: datatypes.NewJSONSlice([]models.Node{{
				ID:   "start",
				Name: "On Task",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas
}
