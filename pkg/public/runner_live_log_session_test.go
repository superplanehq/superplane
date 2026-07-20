package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func mustRunnerLiveLogServer(t *testing.T, r *support.ResourceRegistry) (*Server, *jwt.Signer) {
	t.Helper()
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		r.GitProvider,
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		nil,
		false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider(), r.GitProvider, nil)
	return server, signer
}

func runnerLiveLogSessionGET(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	r *support.ResourceRegistry,
	canvasID, executionID string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs/session", canvasID, executionID),
		nil,
	)
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	token, err := authentication.GenerateAccountToken(signer, r.Account.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}

func createExecutionForCanvasRun(
	t *testing.T,
	run *models.CanvasRun,
	rootEventID uuid.UUID,
	nodeID string,
) *models.CanvasNodeExecution {
	t.Helper()
	now := time.Now()
	execution := models.CanvasNodeExecution{
		ID:            uuid.New(),
		WorkflowID:    run.WorkflowID,
		NodeID:        nodeID,
		RootEventID:   rootEventID,
		RunID:         run.ID,
		EventID:       rootEventID,
		State:         models.CanvasNodeExecutionStatePending,
		Configuration: datatypes.NewJSONType(map[string]any{}),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return &execution
}

func createCanvasWithComponentExecution(
	t *testing.T,
	r *support.ResourceRegistry,
	componentName string,
	nodeID string,
	metadata map[string]any,
) (canvasID uuid.UUID, executionID uuid.UUID) {
	t.Helper()
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "trigger-1",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "start"},
			}),
		},
		{
			NodeID: nodeID,
			Type:   models.NodeTypeComponent,
			Name:   componentName,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Component: &models.ComponentRef{Name: componentName},
			}),
		},
	}, nil)

	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger-1", "default", nil)

	var run *models.CanvasRun
	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = models.FindOrCreateCanvasRunForRootEventInTransaction(tx, rootEvent)
		if err != nil {
			return err
		}
		return rootEvent.RoutedInTransaction(tx)
	}))

	exec := createExecutionForCanvasRun(t, run, rootEvent.ID, nodeID)
	if len(metadata) > 0 {
		exec.Metadata = datatypes.NewJSONType(metadata)
		require.NoError(t, database.Conn().Save(exec).Error)
	}
	return canvas.ID, exec.ID
}

func TestHandleRunnerLiveLogSession(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	server, signer := mustRunnerLiveLogServer(t, r)

	t.Run("no session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs/session", uuid.New(), uuid.New()),
			nil,
		)
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid canvas id", func(t *testing.T) {
		rec := runnerLiveLogSessionGET(t, server, signer, r, "not-a-uuid", uuid.New().String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid canvas id")
	})

	t.Run("invalid execution id", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), "bad-id")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid execution id")
	})

	t.Run("canvas not found", func(t *testing.T) {
		rec := runnerLiveLogSessionGET(t, server, signer, r, uuid.New().String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Canvas not found")
	})

	t.Run("execution not found", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Execution not found")
	})

	t.Run("non-runner node", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "noop", "noop-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "tb-1",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Runner components")
	})

	t.Run("broker task id missing", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not available for this execution")
	})

	t.Run("live logs not configured", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "")
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "tb-x",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "not configured")
	})

	t.Run("returns stream session for runnerBash", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runnerBash", "runner-runbash-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-runbash-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "https://broker.example/v1/tasks/task-runbash-ok/live-logs", session.StreamURL)
	})

	t.Run("returns stream session for runnerJS", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runnerJS", "runner-runjs-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-runjs-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "https://broker.example/v1/tasks/task-runjs-ok/live-logs", session.StreamURL)
	})

	t.Run("returns stream session for runnerPython", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runnerPython", "runner-runpy-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-runpy-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "https://broker.example/v1/tasks/task-runpy-ok/live-logs", session.StreamURL)
	})

	t.Run("returns stream session", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "https://broker.example/v1/tasks/task-ok/live-logs", session.StreamURL)
		assert.NotEmpty(t, session.Token)
		assert.False(t, session.ExpiresAt.IsZero())

		err := runneraction.ValidateLiveLogStreamToken(session.Token, "task-ok", "live-log-secret")
		require.NoError(t, err)
	})
}
