package mcpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
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
	assert.Equal(t, "", row["source_url"])
	assert.Contains(t, row, "assignees")
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

func TestCreateTaskReturnsNewTaskWhenLaterWorkspaceReadsFail(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Create", "", "CRT")
	require.NoError(t, err)

	// Fail workspace reads after the task insert. The completed action must
	// still return its result and URL from the workspace already authorized.
	var inserted atomic.Bool
	createCallback := "mcp_test:task_inserted"
	queryCallback := "mcp_test:workspace_read_failure"
	require.NoError(t, db.Callback().Create().After("gorm:create").Register(createCallback, func(tx *gorm.DB) {
		if tx.Statement.Table == "factory_work_orders" && tx.Error == nil {
			inserted.Store(true)
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Create().Remove(createCallback)) })
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(queryCallback, func(tx *gorm.DB) {
		if tx.Statement.Table == "factories" && inserted.Load() {
			tx.AddError(errors.New("workspace read failed after task creation"))
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(queryCallback)) })

	runtime := &Runtime{Auth: r.AuthService, BaseURL: "https://app.example.test"}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "create_task", map[string]any{
		"title":       "Ship MCP",
		"description": "Expose the workspace tools.",
	})
	var saved models.FactoryWorkOrder
	require.NoError(t, db.Where("factory_id = ? AND title = ?", factoryModel.ID, "Ship MCP").First(&saved).Error)
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "Ship MCP", payload["title"])
	assert.Equal(t, "draft", payload["state"])
	assert.Equal(t, saved.ID.String(), payload["id"])
	assert.Equal(t, fmt.Sprintf("https://app.example.test/%s/workspaces/%s/task/%d",
		r.Organization.ID, factoryModel.RouteSegment(), saved.Number), payload["url"])
}

func TestHandOffTaskReturnsOpenImplementTask(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Hand Off", "", "HND")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	})
	require.NoError(t, err)

	claims := toolClaims(r, factoryModel.ID)
	claims.ClientID = LocalClientID
	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, claims, "hand_off_task", map[string]any{
		"title":  "Ship MCP hand-off",
		"plan":   "# Implement the MCP hand-off.",
		"column": "implement",
	})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "open", payload["state"])
	assert.Equal(t, "implement", payload["column"])
	require.NotEmpty(t, payload["id"])
	require.NotEmpty(t, payload["key"])
}

func TestTaskToolsReturnWebURL(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP URLs", "", "URL")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "Linked task", "", &r.User, nil, nil)
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService, BaseURL: "https://app.example.test/"}
	claims := toolClaims(r, factoryModel.ID)
	want := fmt.Sprintf("https://app.example.test/%s/workspaces/%s/task/%d",
		r.Organization.ID, factoryModel.RouteSegment(), order.Number)

	t.Run("get_task", func(t *testing.T) {
		assert.Equal(t, want, taskURLFromTool(t, runtime, claims, "get_task", map[string]any{"task": order.ID.String()}))
	})

	t.Run("list_tasks", func(t *testing.T) {
		payload := listTasks(t, runtime, claims, map[string]any{})
		rows, ok := payload["tasks"].([]any)
		require.True(t, ok)
		require.Len(t, rows, 1)
		row, ok := rows[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, want, row["url"])
	})

	t.Run("get_task_report", func(t *testing.T) {
		payload := decodeToolJSON(t, mustCallTool(t, runtime, claims, "get_task_report", map[string]any{"task": order.ID.String()}))
		task, ok := payload["task"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, want, task["url"])
	})

	t.Run("assign_task", func(t *testing.T) {
		assert.Equal(t, want, taskURLFromTool(t, runtime, claims, "assign_task", map[string]any{
			"task":     order.ID.String(),
			"unassign": true,
		}))
	})

	t.Run("create_task", func(t *testing.T) {
		created := decodeToolJSON(t, mustCallTool(t, runtime, claims, "create_task", map[string]any{"title": "Second"}))
		createdID, _ := created["id"].(string)
		require.NotEmpty(t, createdID)
		createdURL, _ := created["url"].(string)
		require.NotEmpty(t, createdURL)
		assert.Equal(t, taskURLFromTool(t, runtime, claims, "get_task", map[string]any{"task": createdID}), createdURL)
	})
}

func TestHandOffTaskReturnsWebURL(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Hand Off URL", "", "HOU")
	require.NoError(t, err)
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start-impl")
	_, err = factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	})
	require.NoError(t, err)

	claims := toolClaims(r, factoryModel.ID)
	claims.ClientID = LocalClientID
	runtime := &Runtime{Auth: r.AuthService, BaseURL: "https://app.example.test"}
	handedOff := decodeToolJSON(t, mustCallTool(t, runtime, claims, "hand_off_task", map[string]any{
		"title":  "Hand-off URL",
		"plan":   "# Check the URL.",
		"column": "implement",
	}))
	handedOffURL, _ := handedOff["url"].(string)
	require.NotEmpty(t, handedOffURL)
	assert.Equal(t, taskURLFromTool(t, runtime, claims, "get_task", map[string]any{"task": handedOff["id"]}), handedOffURL)
}

// taskURLFromTool calls a flat task tool and returns the task url field.
func taskURLFromTool(t *testing.T, runtime *Runtime, claims *AccessClaims, tool string, args map[string]any) string {
	t.Helper()
	payload := decodeToolJSON(t, mustCallTool(t, runtime, claims, tool, args))
	url, ok := payload["url"].(string)
	require.True(t, ok, "%s response has no url", tool)
	return url
}

func mustCallTool(t *testing.T, runtime *Runtime, claims *AccessClaims, tool string, args map[string]any) ToolResult {
	t.Helper()
	result, err := runtime.CallTool(t.Context(), claims, tool, args)
	require.NoError(t, err)
	return result
}

func TestCallToolSucceedsWithoutMCPServerFlag(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "No MCP", "", "NMC")
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "list_tasks", map[string]any{})
	require.NoError(t, err)
	require.False(t, result.IsError)
}

func TestCallToolRejectsWithoutFactories(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.DisableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "No factories", "", "NF")
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
	support.AssignOpenSystemTrial(t, r.Organization.ID)
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

func TestResolveAssigneeMemberMatchesEmailBeforeGitHubLogin(t *testing.T) {
	ada := models.FactoryVelocityMember{
		UserID:      uuid.New(),
		Name:        "Ada Lovelace",
		Email:       "ada@example.com",
		GitHubLogin: "ada",
	}
	grace := models.FactoryVelocityMember{
		UserID:      uuid.New(),
		Name:        "Grace Hopper",
		Email:       "grace@example.com",
		GitHubLogin: "ada@example.com",
	}
	twin := models.FactoryVelocityMember{
		UserID:      uuid.New(),
		Name:        "Other Ada",
		Email:       "ada@example.com",
		GitHubLogin: "other",
	}

	member, err := resolveAssigneeMember([]models.FactoryVelocityMember{ada, grace}, " ADA@example.com ")
	require.NoError(t, err)
	assert.Equal(t, ada.UserID, member.UserID)

	member, err = resolveAssigneeMember([]models.FactoryVelocityMember{ada, grace}, ada.UserID.String())
	require.NoError(t, err)
	assert.Equal(t, ada.UserID, member.UserID)

	_, err = resolveAssigneeMember([]models.FactoryVelocityMember{ada, grace}, "Ada Lovelace")
	require.EqualError(t, err, "The member was not found.")

	_, err = resolveAssigneeMember([]models.FactoryVelocityMember{ada, twin}, "ada@example.com")
	require.EqualError(t, err, "More than one member matches. Use a user id.")
}

func TestResolveAssigneeMemberMatchesUUIDShapedLogin(t *testing.T) {
	login := "123e4567-e89b-12d3-a456-426614174000"
	ownerID := uuid.MustParse(login)
	ada := models.FactoryVelocityMember{
		UserID:      uuid.New(),
		Name:        "Ada Lovelace",
		Email:       "ada@example.com",
		GitHubLogin: login,
	}
	grace := models.FactoryVelocityMember{
		UserID:      ownerID,
		Name:        "Grace Hopper",
		Email:       "grace@example.com",
		GitHubLogin: "grace",
	}

	member, err := resolveAssigneeMember([]models.FactoryVelocityMember{ada}, login)
	require.NoError(t, err)
	assert.Equal(t, ada.UserID, member.UserID)

	member, err = resolveAssigneeMember([]models.FactoryVelocityMember{ada, grace}, login)
	require.NoError(t, err)
	assert.Equal(t, grace.UserID, member.UserID)
}

func TestAssignTaskSetsOneOwner(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Assign", "", "ASN")
	require.NoError(t, err)
	owner := createHumanMember(t, r.Organization.ID, "Ada Lovelace", "ada-"+uuid.NewString()+"@example.com")
	linkGitHub(t, owner, "OctoCat")
	other := createHumanMember(t, r.Organization.ID, "Grace Hopper", "grace-"+uuid.NewString()+"@example.com")
	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, factoryModel.ID)

	t.Run("assigns by email, GitHub login, and user id", func(t *testing.T) {
		byEmail := createDraftTask(t, factoryModel, r.User)
		result, err := runtime.CallTool(ctx, claims, "assign_task", map[string]any{
			"task":     factoryModel.WorkOrderKey(byEmail.Number),
			"assignee": strings.ToUpper(*owner.Email),
		})
		require.NoError(t, err)
		assert.Equal(t, []string{owner.ID.String()}, assigneeIDs(t, decodeToolJSON(t, result)["assignees"]))

		byLogin := createDraftTask(t, factoryModel, r.User)
		result, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
			"task":     strconv.FormatInt(byLogin.Number, 10),
			"assignee": " octocat ",
		})
		require.NoError(t, err)
		assert.Equal(t, []string{owner.ID.String()}, assigneeIDs(t, decodeToolJSON(t, result)["assignees"]))

		byID := createDraftTask(t, factoryModel, r.User)
		result, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
			"task":     byID.ID.String(),
			"assignee": strings.ToUpper(owner.ID.String()),
		})
		require.NoError(t, err)
		payload := decodeToolJSON(t, result)
		assert.Equal(t, byID.ID.String(), payload["id"])
		assert.Equal(t, factoryModel.WorkOrderKey(byID.Number), payload["key"])
		assert.Equal(t, []string{owner.ID.String()}, assigneeIDs(t, payload["assignees"]))
		assert.Equal(t, []string{owner.ID.String()}, storedAssigneeIDs(t, factoryModel, byID.ID))
	})

	t.Run("replaces the owner on a draft, an open task, and a closed task", func(t *testing.T) {
		for _, state := range []string{
			models.FactoryWorkOrderStateDraft,
			models.FactoryWorkOrderStateOpen,
			models.FactoryWorkOrderStateClosed,
		} {
			order := createDraftTask(t, factoryModel, r.User)
			setTaskState(t, order, state, r.User)
			_, err := runtime.CallTool(ctx, claims, "assign_task", map[string]any{
				"task":     order.ID.String(),
				"assignee": owner.ID.String(),
			})
			require.NoError(t, err)
			_, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
				"task":     order.ID.String(),
				"assignee": other.ID.String(),
			})
			require.NoError(t, err)
			assert.Equal(t, []string{other.ID.String()}, storedAssigneeIDs(t, factoryModel, order.ID))
		}
	})

	t.Run("clears every owner", func(t *testing.T) {
		order := createDraftTask(t, factoryModel, r.User)
		require.NoError(t, order.ReplaceAssignees(db, []uuid.UUID{owner.ID, other.ID}))

		result, err := runtime.CallTool(ctx, claims, "assign_task", map[string]any{
			"task":     order.ID.String(),
			"unassign": true,
		})
		require.NoError(t, err)
		assert.Empty(t, assigneeIDs(t, decodeToolJSON(t, result)["assignees"]))
		assert.Empty(t, storedAssigneeIDs(t, factoryModel, order.ID))
	})
}

func TestAssignTaskRejectsInvalidChanges(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Assign Reject", "", "ARJ")
	require.NoError(t, err)
	owner := createHumanMember(t, r.Organization.ID, "Visible Owner", "owner-"+uuid.NewString()+"@example.com")
	first := createHumanMember(t, r.Organization.ID, "First Twin", "first-"+uuid.NewString()+"@example.com")
	second := createHumanMember(t, r.Organization.ID, "Second Twin", "second-"+uuid.NewString()+"@example.com")
	linkGitHub(t, first, "shared-login")
	linkGitHub(t, second, "Shared-Login")
	apiKey, err := models.CreateAPIKey(db, r.Organization.ID, "bot-key", nil, r.User, nil, nil)
	require.NoError(t, err)

	order := createDraftTask(t, factoryModel, r.User)
	require.NoError(t, order.ReplaceAssignees(db, []uuid.UUID{owner.ID}))
	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, factoryModel.ID)

	reject := func(args map[string]any, message string) {
		t.Helper()
		_, err := runtime.CallTool(ctx, claims, "assign_task", args)
		require.EqualError(t, err, message)
		assert.Equal(t, []string{owner.ID.String()}, storedAssigneeIDs(t, factoryModel, order.ID))
	}

	reject(map[string]any{"task": order.ID.String()}, "Set assignee or set unassign to true.")
	reject(map[string]any{"task": order.ID.String(), "assignee": "   "}, "Set assignee or set unassign to true.")
	reject(map[string]any{
		"task":     order.ID.String(),
		"assignee": *owner.Email,
		"unassign": true,
	}, "Do not set assignee and unassign.")
	reject(map[string]any{"task": order.ID.String(), "assignee": "missing@example.com"}, "The member was not found.")
	reject(map[string]any{"task": order.ID.String(), "assignee": "Visible Owner"}, "The member was not found.")
	reject(map[string]any{"task": order.ID.String(), "assignee": apiKey.ID.String()}, "The member was not found.")
	reject(map[string]any{"task": order.ID.String(), "assignee": "shared-login"}, "More than one member matches. Use a user id.")
	reject(map[string]any{
		"task":     order.ID.String(),
		"assignee": float64(123),
		"unassign": true,
	}, "assignee must be text.")
	reject(map[string]any{
		"task":     order.ID.String(),
		"assignee": *owner.Email,
		"unassign": "true",
	}, "unassign must be true or false.")

	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "Other workspace", "", "OTH")
	require.NoError(t, err)
	secret, err := otherFactory.CreateWorkOrder(db, "Secret", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, secret.ReplaceAssignees(db, []uuid.UUID{owner.ID}))
	_, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
		"task":     secret.ID.String(),
		"assignee": *first.Email,
	})
	require.EqualError(t, err, "Not found")
	assert.Equal(t, []string{owner.ID.String()}, storedAssigneeIDs(t, otherFactory, secret.ID))

	denied := toolClaims(r, factoryModel.ID)
	denied.Scopes = []string{"work_orders:read"}
	_, err = runtime.CallTool(ctx, denied, "assign_task", map[string]any{
		"task":     order.ID.String(),
		"assignee": *owner.Email,
	})
	require.EqualError(t, err, "Not found")
	assert.Equal(t, []string{owner.ID.String()}, storedAssigneeIDs(t, factoryModel, order.ID))
}

func TestTaskReadsIncludeAssignees(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Owners", "", "OWN")
	require.NoError(t, err)
	owner := createHumanMember(t, r.Organization.ID, "Ada Lovelace", "ada-"+uuid.NewString()+"@example.com")
	order := createDraftTask(t, factoryModel, r.User)
	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, factoryModel.ID)

	_, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
		"task":     order.ID.String(),
		"assignee": *owner.Email,
	})
	require.NoError(t, err)

	got, err := runtime.CallTool(ctx, claims, "get_task", map[string]any{"task": order.ID.String()})
	require.NoError(t, err)
	payload := decodeToolJSON(t, got)
	assert.Equal(t, []string{owner.ID.String()}, assigneeIDs(t, payload["assignees"]))
	assert.Equal(t, "Ada Lovelace", payload["assignees"].([]any)[0].(map[string]any)["name"])

	listed, err := runtime.CallTool(ctx, claims, "list_tasks", map[string]any{})
	require.NoError(t, err)
	row := taskRowByID(t, decodeToolJSON(t, listed), order.ID.String())
	assert.Equal(t, []string{owner.ID.String()}, assigneeIDs(t, row["assignees"]))

	_, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
		"task":     order.ID.String(),
		"unassign": true,
	})
	require.NoError(t, err)

	got, err = runtime.CallTool(ctx, claims, "get_task", map[string]any{"task": order.ID.String()})
	require.NoError(t, err)
	assert.Empty(t, assigneeIDs(t, decodeToolJSON(t, got)["assignees"]))

	listed, err = runtime.CallTool(ctx, claims, "list_tasks", map[string]any{})
	require.NoError(t, err)
	row = taskRowByID(t, decodeToolJSON(t, listed), order.ID.String())
	assert.Empty(t, assigneeIDs(t, row["assignees"]))
}

func TestListMembersReturnsHumanMembers(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Members", "", "MBR")
	require.NoError(t, err)
	amy := createHumanMember(t, r.Organization.ID, "Amy", "amy-"+uuid.NewString()+"@example.com")
	zoe := createHumanMember(t, r.Organization.ID, "Zoe", "zoe-"+uuid.NewString()+"@example.com")
	linkGitHub(t, zoe, "zoe-gh")
	_, err = models.CreateAPIKey(db, r.Organization.ID, "Key Bot", nil, r.User, nil, nil)
	require.NoError(t, err)

	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(ctx, toolClaims(r, factoryModel.ID), "list_members", map[string]any{})
	require.NoError(t, err)
	members, ok := decodeToolJSON(t, result)["members"].([]any)
	require.True(t, ok)

	amyRow := memberByEmail(t, members, *amy.Email)
	assert.Equal(t, amy.ID.String(), amyRow["id"])
	assert.Equal(t, "Amy", amyRow["name"])
	assert.Equal(t, "", amyRow["github_login"])

	zoeRow := memberByEmail(t, members, *zoe.Email)
	assert.Equal(t, "zoe-gh", zoeRow["github_login"])
	assert.Less(t, memberIndex(t, members, *amy.Email), memberIndex(t, members, *zoe.Email))

	setup := memberByID(t, members, r.User.String())
	assert.Equal(t, "testuser", setup["github_login"])
	for _, member := range members {
		row, ok := member.(map[string]any)
		require.True(t, ok)
		assert.NotEqual(t, "Key Bot", row["name"])
	}
}

func TestListMembersRequiresMembersRead(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Member Deny", "", "MDN")
	require.NoError(t, err)
	reader := createHumanMember(t, r.Organization.ID, "Reader", "reader-"+uuid.NewString()+"@example.com")
	roleName := "task-reader-" + uuid.NewString()
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:        roleName,
		DisplayName: "Task reader",
		DomainType:  models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{{
			Resource:   "work_orders",
			Action:     "read",
			DomainType: models.DomainTypeOrganization,
		}},
	}))
	require.NoError(t, r.AuthService.AssignRole(reader.ID.String(), roleName, r.Organization.ID.String(), models.DomainTypeOrganization))

	claims := toolClaims(r, factoryModel.ID)
	claims.UserID = reader.ID
	runtime := &Runtime{Auth: r.AuthService}
	_, err = runtime.CallTool(ctx, claims, "list_members", map[string]any{})
	require.EqualError(t, err, "Not found")
}

func TestAssignTaskRequiresMembersRead(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Assign Deny", "", "ADN")
	require.NoError(t, err)
	owner := createHumanMember(t, r.Organization.ID, "Visible Owner", "owner-"+uuid.NewString()+"@example.com")
	linkGitHub(t, owner, "visible-owner")
	editor := createHumanMember(t, r.Organization.ID, "Task Editor", "editor-"+uuid.NewString()+"@example.com")
	roleName := "task-editor-" + uuid.NewString()
	require.NoError(t, r.AuthService.CreateCustomRole(r.Organization.ID.String(), &authorization.RoleDefinition{
		Name:        roleName,
		DisplayName: "Task editor",
		DomainType:  models.DomainTypeOrganization,
		Permissions: []*authorization.Permission{{
			Resource:   "work_orders",
			Action:     "update",
			DomainType: models.DomainTypeOrganization,
		}},
	}))
	require.NoError(t, r.AuthService.AssignRole(editor.ID.String(), roleName, r.Organization.ID.String(), models.DomainTypeOrganization))

	order := createDraftTask(t, factoryModel, r.User)
	require.NoError(t, order.ReplaceAssignees(db, []uuid.UUID{owner.ID}))
	claims := toolClaims(r, factoryModel.ID)
	claims.UserID = editor.ID
	runtime := &Runtime{Auth: r.AuthService}

	for _, assignee := range []string{*owner.Email, "visible-owner", owner.ID.String(), "missing@example.com"} {
		_, err = runtime.CallTool(ctx, claims, "assign_task", map[string]any{
			"task":     order.ID.String(),
			"assignee": assignee,
		})
		require.EqualError(t, err, "Not found")
		assert.Equal(t, []string{owner.ID.String()}, storedAssigneeIDs(t, factoryModel, order.ID))
	}

	result, err := runtime.CallTool(ctx, claims, "assign_task", map[string]any{
		"task":     order.ID.String(),
		"unassign": true,
	})
	require.NoError(t, err)
	assert.Empty(t, assigneeIDs(t, decodeToolJSON(t, result)["assignees"]))
	assert.Empty(t, storedAssigneeIDs(t, factoryModel, order.ID))
}

func createDraftTask(t *testing.T, factoryModel *models.Factory, creator uuid.UUID) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), support.RandomName("task"), "", &creator, nil, nil)
	require.NoError(t, err)
	return order
}

func setTaskState(t *testing.T, order *models.FactoryWorkOrder, state string, actor uuid.UUID) {
	t.Helper()
	if state == models.FactoryWorkOrderStateDraft {
		return
	}
	if state == models.FactoryWorkOrderStateClosed {
		_, err := order.UpdateStatus(database.DB(t.Context()), models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
			Actor:   &actor,
		})
		require.NoError(t, err)
		_, err = order.UpdateStatus(database.DB(t.Context()), models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateClosed,
			Result:  models.FactoryWorkOrderResultCompleted,
			Actor:   &actor,
		})
		require.NoError(t, err)
		return
	}
	_, err := order.UpdateStatus(database.DB(t.Context()), models.FactoryWorkOrderStatusUpdate{
		ToState: state,
		Actor:   &actor,
	})
	require.NoError(t, err)
}

func createHumanMember(t *testing.T, orgID uuid.UUID, name, email string) *models.User {
	t.Helper()
	account, err := models.CreateAccount(name, email)
	require.NoError(t, err)
	user, err := models.CreateUser(orgID, account.ID, email, name)
	require.NoError(t, err)
	return user
}

func linkGitHub(t *testing.T, user *models.User, login string) {
	t.Helper()
	require.NotNil(t, user.AccountID)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(*user.AccountID, models.ProviderGitHub, uuid.NewString(), login, "", ""),
	))
}

func assigneeIDs(t *testing.T, value any) []string {
	t.Helper()
	rows, ok := value.([]any)
	require.True(t, ok)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		require.True(t, ok)
		id, _ := item["id"].(string)
		require.NotEmpty(t, id)
		_, hasEmail := item["email"]
		assert.False(t, hasEmail)
		ids = append(ids, id)
	}
	return ids
}

func storedAssigneeIDs(t *testing.T, factoryModel *models.Factory, orderID uuid.UUID) []string {
	t.Helper()
	order, err := factoryModel.FindWorkOrder(database.DB(t.Context()), orderID)
	require.NoError(t, err)
	ids := make([]string, 0, len(order.Assignees))
	for _, assignee := range order.Assignees {
		ids = append(ids, assignee.UserID.String())
	}
	return ids
}

func taskRowByID(t *testing.T, payload map[string]any, id string) map[string]any {
	t.Helper()
	tasks, ok := payload["tasks"].([]any)
	require.True(t, ok)
	for _, task := range tasks {
		row, ok := task.(map[string]any)
		require.True(t, ok)
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("task %s not found", id)
	return nil
}

func memberByEmail(t *testing.T, members []any, email string) map[string]any {
	t.Helper()
	for _, member := range members {
		row, ok := member.(map[string]any)
		require.True(t, ok)
		if row["email"] == email {
			return row
		}
	}
	t.Fatalf("member %s not found", email)
	return nil
}

func memberByID(t *testing.T, members []any, id string) map[string]any {
	t.Helper()
	for _, member := range members {
		row, ok := member.(map[string]any)
		require.True(t, ok)
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("member %s not found", id)
	return nil
}

func memberIndex(t *testing.T, members []any, email string) int {
	t.Helper()
	for index, member := range members {
		row, ok := member.(map[string]any)
		require.True(t, ok)
		if row["email"] == email {
			return index
		}
	}
	t.Fatalf("member %s not found", email)
	return -1
}

func TestListWorkspacesReturnsGrantedWorkspaces(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)
	first, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "ALP")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, "Beta", "", "BET")
	require.NoError(t, err)
	_, err = models.CreateFactory(db, r.Organization.ID, "Hidden", "", "HID")
	require.NoError(t, err)

	claims := toolClaims(r, first.ID)
	claims.FactoryIDs = []uuid.UUID{second.ID, first.ID}
	runtime := &Runtime{Auth: r.AuthService, BaseURL: "https://app.example.test"}
	result, err := runtime.CallTool(ctx, claims, "list_workspaces", nil)
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	rows, ok := payload["workspaces"].([]any)
	require.True(t, ok)
	require.Len(t, rows, 2)
	names := map[string]string{}
	slugs := map[string]string{}
	urls := map[string]string{}
	for _, row := range rows {
		item, ok := row.(map[string]any)
		require.True(t, ok)
		id := item["id"].(string)
		names[id] = item["name"].(string)
		slugs[id] = item["slug"].(string)
		urls[id] = item["url"].(string)
	}
	assert.Equal(t, "Alpha", names[first.ID.String()])
	assert.Equal(t, "Beta", names[second.ID.String()])
	assert.NotContains(t, names, "Hidden")
	assert.Equal(t, first.RouteSegment(), slugs[first.ID.String()])
	assert.Equal(t, fmt.Sprintf("https://app.example.test/%s/workspaces/%s", r.Organization.ID, first.RouteSegment()), urls[first.ID.String()])
}

func TestWorkspaceArgumentSelectsGrantedWorkspace(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)
	first, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "ALP")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, "Beta", "", "BET")
	require.NoError(t, err)
	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, first.ID)
	claims.FactoryIDs = []uuid.UUID{first.ID, second.ID}

	_, err = runtime.CallTool(ctx, claims, "list_tasks", map[string]any{})
	require.EqualError(t, err, "This connection can use more than one workspace. Pass workspace.")
	_, err = runtime.CallTool(ctx, claims, "get_task_report", map[string]any{"task": "1"})
	require.EqualError(t, err, "This connection can use more than one workspace. Pass workspace.")

	result, err := runtime.CallTool(ctx, claims, "list_tasks", map[string]any{"workspace": second.ID.String()})
	require.NoError(t, err)
	assert.False(t, result.IsError)

	_, err = runtime.CallTool(ctx, claims, "list_tasks", map[string]any{"workspace": uuid.NewString()})
	require.EqualError(t, err, "Not found")
}

func TestHandOffTaskReturnsPendingBacklogTarget(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	db := database.DB(t.Context())
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	factory, err := models.CreateFactory(db, r.Organization.ID, "Personal subscriptions", "", "PERS")
	require.NoError(t, err)
	subscription := true
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "implement", "start")
	line, err := factory.CreateLine(db, "ship", []models.FactoryLineStep{{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry}})
	require.NoError(t, err)
	runtime := &Runtime{Auth: r.AuthService}
	result, err := runtime.CallTool(t.Context(), toolClaims(r, factory.ID), "hand_off_task", map[string]any{"title": "Unassigned handoff", "plan": "Keep the plan", "column": "verify", "pull_request_url": "https://github.com/acme/app/pull/42"})
	require.NoError(t, err)
	payload := decodeToolJSON(t, result)
	assert.Equal(t, "draft", payload["state"])
	assert.Equal(t, "backlog", payload["column"])
	pending, ok := payload["pending_handoff"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "verify", pending["column"])
	assert.Equal(t, line.ID.String(), pending["line_id"])
	assert.Equal(t, "https://github.com/acme/app/pull/42", pending["pull_request_url"])
}
