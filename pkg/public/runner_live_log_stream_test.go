package public

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, server.RegisterGRPCGateway("localhost:50051"))
	return server, signer
}

func runnerLiveLogGET(
	t *testing.T,
	server *Server,
	signer *jwt.Signer,
	r *support.ResourceRegistry,
	canvasID, executionID string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs", canvasID, executionID),
		nil,
	)
	req.Header.Set("x-organization-id", r.Organization.ID.String())
	token, err := signer.Generate(r.Account.ID.String(), time.Hour)
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

func TestHandleRunnerLiveLogStream(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	server, signer := mustRunnerLiveLogServer(t, r)

	t.Run("no session cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs", uuid.New(), uuid.New()),
			nil,
		)
		req.Header.Set("x-organization-id", r.Organization.ID.String())
		rec := httptest.NewRecorder()
		server.Router.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("invalid canvas id", func(t *testing.T) {
		rec := runnerLiveLogGET(t, server, signer, r, "not-a-uuid", uuid.New().String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid canvas id")
	})

	t.Run("invalid execution id", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), "bad-id")
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid execution id")
	})

	t.Run("canvas not found", func(t *testing.T) {
		rec := runnerLiveLogGET(t, server, signer, r, uuid.New().String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Canvas not found")
	})

	t.Run("execution not found", func(t *testing.T) {
		canvasID, _ := createCanvasWithComponentExecution(t, r, "runner", "runner-1", nil)
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), uuid.New().String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Execution not found")
	})

	t.Run("non-runner node", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "noop", "noop-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "tb-1",
		})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Runner components")
	})

	t.Run("broker task id missing", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "not available for this execution")
	})

	t.Run("broker task id from non-string metadata", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: 99,
		})
		t.Setenv("TASK_BROKER_BASE_URL", "http://127.0.0.1:1")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "token")
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusBadGateway, rec.Code)
	})

	t.Run("task broker not configured", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "")
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "tb-x",
		})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Contains(t, rec.Body.String(), "not configured")
	})

	t.Run("upstream error response is proxied", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTeapot)
			_, _ = w.Write([]byte("upstream"))
		}))
		t.Cleanup(upstream.Close)
		t.Setenv("TASK_BROKER_BASE_URL", upstream.URL)
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-418",
		})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusTeapot, rec.Code)
		assert.Contains(t, rec.Body.String(), "upstream")
	})

	t.Run("upstream success streams body", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/tasks/task-ok/live-logs", r.URL.Path)
			assert.Equal(t, "Bearer broker-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/x-ndjson", r.Header.Get("Accept"))
			assert.Equal(t, "identity", r.Header.Get("Accept-Encoding"))
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"type":"line","text":"log"}` + "\n"))
		}))
		t.Cleanup(upstream.Close)
		t.Setenv("TASK_BROKER_BASE_URL", upstream.URL)
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "broker-token")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-ok",
		})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		assert.Contains(t, rec.Body.String(), "log")
	})

	t.Run("upstream success forwards upstream content-type", func(t *testing.T) {
		// httptest.Server sets Content-Type to text/plain when the handler does not set one.
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("x"))
		}))
		t.Cleanup(upstream.Close)
		t.Setenv("TASK_BROKER_BASE_URL", upstream.URL)
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "t")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-ct",
		})
		rec := runnerLiveLogGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "x", rec.Body.String())
		assert.NotEmpty(t, rec.Header().Get("Content-Type"))
	})
}
