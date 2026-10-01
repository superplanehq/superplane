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

// createOnRunCanvasForMCP creates a factory-owned canvas with a single
// onRun-triggered node, usable as a factory line step's entrypoint.
// StartStep (and so create_task's handoff) only dispatches to onRun
// entrypoints; createOnWorkOrderCanvasForMCP's onWorkOrder trigger is not
// dispatchable this way.
func createOnRunCanvasForMCP(t *testing.T, r *support.ResourceRegistry, factoryID uuid.UUID, name string) *models.Canvas {
	t.Helper()
	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryID,
		Name:           name,
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
			Name:       name,
			Type:       models.NodeTypeTrigger,
			State:      models.CanvasNodeStateReady,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "onRun"},
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
				Name: name,
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			}}),
			Edges:     datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt: &now,
			UpdatedAt: &now,
		}
		return tx.Create(&version).Error
	}))
	return canvas
}

func TestCreateTaskWithStartStepByName(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Handoff", "", "HND")
	require.NoError(t, err)

	// Create two canvases for the line steps. The second step is the
	// handoff target, so a resolver that always returns the first step
	// would fail this test.
	verifyCanvas := createOnRunCanvasForMCP(t, r, factoryModel.ID, "Verify")
	doneCanvas := createOnRunCanvasForMCP(t, r, factoryModel.ID, "Done")

	// Create a line with two steps
	_, err = factoryModel.CreateLine(db, "main", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: verifyCanvas.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: doneCanvas.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":       "Hand off to Done",
		"description": "Skip backlog, plan, and verify",
		"line":        "main",
		"start_step":  "Done",
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "Hand off to Done", payload["title"])
	assert.Equal(t, "open", payload["state"])
	require.NotEmpty(t, payload["id"])

	// Verify the task is dispatched to the correct step
	orderID := payload["id"].(string)
	workOrder, err := models.FindUnscopedWorkOrder(db, parseUUID(orderID))
	require.NoError(t, err)
	activeDispatch, err := workOrder.FindActiveLineDispatch(db)
	require.NoError(t, err)
	assert.Equal(t, "main", activeDispatch.LineName)
	assert.Equal(t, 2, len(activeDispatch.Steps))
	executions, err := models.ListFactoryWorkOrderExecutionsByLineDispatchIDs(db, []uuid.UUID{activeDispatch.ID})
	require.NoError(t, err)
	require.Len(t, executions[activeDispatch.ID], 1)
	assert.Equal(t, 1, executions[activeDispatch.ID][0].StepIndex)
}

func TestCreateTaskWithStartStepByIndex(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Index", "", "IDX")
	require.NoError(t, err)

	// Create two canvases for the line steps
	backlogCanvas := createOnRunCanvasForMCP(t, r, factoryModel.ID, "Backlog")
	verifyCanvas := createOnRunCanvasForMCP(t, r, factoryModel.ID, "Verify")

	// Create a line with two steps
	_, err = factoryModel.CreateLine(db, "main", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: backlogCanvas.ID, Entrypoint: "start"},
		{Type: models.FactoryLineStepTypeRunApp, AppID: verifyCanvas.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":      "Hand off at index",
		"line":       "main",
		"start_step": "1",
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "Hand off at index", payload["title"])

	// Verify the task is dispatched to step index 1
	orderID := payload["id"].(string)
	workOrder, err := models.FindUnscopedWorkOrder(db, parseUUID(orderID))
	require.NoError(t, err)
	activeDispatch, err := workOrder.FindActiveLineDispatch(db)
	require.NoError(t, err)
	executions, err := models.ListFactoryWorkOrderExecutionsByLineDispatchIDs(db, []uuid.UUID{activeDispatch.ID})
	require.NoError(t, err)
	require.Len(t, executions[activeDispatch.ID], 1)
	assert.Equal(t, 1, executions[activeDispatch.ID][0].StepIndex)
}

func TestCreateTaskWithInvalidStartStep(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Bad Step", "", "BAD")
	require.NoError(t, err)

	verifyCanvas := createOnWorkOrderCanvasForMCP(t, r, factoryModel.ID)
	verifyCanvas.Name = "Verify"
	require.NoError(t, db.Model(verifyCanvas).Update("name", "Verify").Error)

	_, err = factoryModel.CreateLine(db, "main", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: verifyCanvas.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":      "Hand off",
		"line":       "main",
		"start_step": "NonExistent",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// An invalid start_step must not leave an orphan task behind.
	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}

func TestCreateTaskWithMultipleLinesRequiresLineArg(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Multi Line", "", "MUL")
	require.NoError(t, err)

	// Create two lines
	canvas1 := createOnWorkOrderCanvasForMCP(t, r, factoryModel.ID)
	canvas1.Name = "Step1"
	require.NoError(t, db.Model(canvas1).Update("name", "Step1").Error)

	canvas2 := createOnWorkOrderCanvasForMCP(t, r, factoryModel.ID)
	canvas2.Name = "Step2"
	require.NoError(t, db.Model(canvas2).Update("name", "Step2").Error)

	_, err = factoryModel.CreateLine(db, "line1", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: canvas1.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	_, err = factoryModel.CreateLine(db, "line2", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: canvas2.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":      "Hand off",
		"start_step": "Step1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple lines exist")
	assert.Contains(t, err.Error(), "line1")
	assert.Contains(t, err.Error(), "line2")

	// An unresolved line must not leave an orphan task behind.
	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}

func TestCreateTaskWithPullRequest(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP PR", "", "PRR")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title": "Feature with PR",
		"pull_request": map[string]any{
			"repository": "octocat/Hello-World",
			"number":     float64(42),
			"url":        "https://github.com/octocat/Hello-World/pull/42",
			"title":      "Fix bug",
			"state":      "open",
			"provider":   "github",
		},
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "Feature with PR", payload["title"])

	// Verify the PR was attached
	orderID := payload["id"].(string)
	workOrder, err := models.FindUnscopedWorkOrder(db, parseUUID(orderID))
	require.NoError(t, err)
	prsByOrder, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{workOrder.ID})
	require.NoError(t, err)
	prs := prsByOrder[workOrder.ID]
	require.Len(t, prs, 1)
	assert.Equal(t, "octocat/Hello-World", prs[0].Repository)
	assert.Equal(t, int64(42), prs[0].Number)
}

func TestCreateTaskWithDraftPullRequestKeepsDraftState(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Draft PR", "", "DFT")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title": "Feature with draft PR",
		"pull_request": map[string]any{
			"repository": "octocat/Hello-World",
			"number":     float64(7),
			"url":        "https://github.com/octocat/Hello-World/pull/7",
			"title":      "Work in progress",
			"state":      "draft",
		},
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)

	orderID := payload["id"].(string)
	workOrder, err := models.FindUnscopedWorkOrder(db, parseUUID(orderID))
	require.NoError(t, err)
	prsByOrder, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{workOrder.ID})
	require.NoError(t, err)
	prs := prsByOrder[workOrder.ID]
	require.Len(t, prs, 1)
	assert.Equal(t, models.FactoryPullRequestStateDraft, prs[0].State)
}

func TestCreateTaskWithBitbucketPullRequest(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Bitbucket PR", "", "BBK")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title": "Feature with Bitbucket PR",
		"pull_request": map[string]any{
			"repository": "team/repo",
			"number":     float64(3),
			"url":        "https://bitbucket.org/team/repo/pull-requests/3",
			"title":      "Fix bug",
			"state":      "open",
			"provider":   "bitbucket",
		},
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)

	orderID := payload["id"].(string)
	workOrder, err := models.FindUnscopedWorkOrder(db, parseUUID(orderID))
	require.NoError(t, err)
	prsByOrder, err := models.ListPullRequestsByWorkOrderIDs(db, []uuid.UUID{workOrder.ID})
	require.NoError(t, err)
	prs := prsByOrder[workOrder.ID]
	require.Len(t, prs, 1)
	assert.Equal(t, models.FactoryPullRequestProviderBitbucket, prs[0].Provider)
}

func TestCreateTaskRejectsFractionalPullRequestNumber(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Fractional PR", "", "FRC")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title": "Feature with fractional PR number",
		"pull_request": map[string]any{
			"repository": "octocat/Hello-World",
			"number":     42.5,
			"url":        "https://github.com/octocat/Hello-World/pull/42",
			"title":      "Fix bug",
			"state":      "open",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "whole number")

	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}

func TestCreateTaskRejectsMistypedStartStepIndex(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Mistyped Step", "", "MST")
	require.NoError(t, err)

	verifyCanvas := createOnRunCanvasForMCP(t, r, factoryModel.ID, "Verify")
	_, err = factoryModel.CreateLine(db, "main", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: verifyCanvas.ID, Entrypoint: "start"},
	})
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":      "Hand off",
		"line":       "main",
		"start_step": "1abc",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}

func TestCreateTaskRejectsEmptyStartStep(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Empty Step", "", "EMP")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":      "Hand off",
		"start_step": "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start_step")

	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}

func TestCreateTaskWithMissingPullRequestField(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Bad PR", "", "BPR")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title": "Feature with bad PR",
		"pull_request": map[string]any{
			"repository": "octocat/Hello-World",
			"number":     float64(42),
			"url":        "https://github.com/octocat/Hello-World/pull/42",
			// Missing title and state
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	// An invalid pull request must not leave an orphan task behind.
	var taskCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factoryModel.ID).Count(&taskCount).Error)
	assert.Zero(t, taskCount)
}
