package public

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
	sentrytest "github.com/superplanehq/superplane/test/support/sentry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestWriteRunnerLiveLogSessionError(t *testing.T) {
	for _, tt := range []struct {
		name          string
		err           error
		cancelRequest bool
		expireRequest bool
		status        int
	}{
		{
			name: "plain cancellation on canceled request",
			err:  context.Canceled, cancelRequest: true, status: statusClientClosedRequest,
		},
		{
			name:          "wrapped cancellation on canceled request",
			err:           fmt.Errorf("timeout: context already done: %w", context.Canceled),
			cancelRequest: true, status: statusClientClosedRequest,
		},
		{
			name:          "wrapped deadline on canceled request",
			err:           fmt.Errorf("lookup: %w", context.DeadlineExceeded),
			cancelRequest: true, status: statusClientClosedRequest,
		},
		{
			name: "plain cancellation on live request",
			err:  context.Canceled, status: http.StatusInternalServerError,
		},
		{
			name: "wrapped cancellation on live request",
			err:  fmt.Errorf("lookup: %w", context.Canceled), status: http.StatusInternalServerError,
		},
		{
			name: "plain deadline on live request",
			err:  context.DeadlineExceeded, status: http.StatusInternalServerError,
		},
		{
			name: "wrapped deadline on live request",
			err:  fmt.Errorf("lookup: %w", context.DeadlineExceeded), status: http.StatusInternalServerError,
		},
		{
			name: "database failure on canceled request",
			err:  errors.New("database unavailable"), cancelRequest: true, status: http.StatusInternalServerError,
		},
		{
			name: "deadline on expired incoming request",
			err:  context.DeadlineExceeded, expireRequest: true, status: http.StatusInternalServerError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			transport := sentrytest.BindHub(t)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases/canvas/node-executions/execution/runner-live-logs/session", nil)
			if tt.cancelRequest {
				ctx, cancel := context.WithCancel(req.Context())
				cancel()
				req = req.WithContext(ctx)
			}
			if tt.expireRequest {
				ctx, cancel := context.WithDeadline(req.Context(), time.Now().Add(-time.Second))
				defer cancel()
				req = req.WithContext(ctx)
			}
			logger := log.New()
			logger.SetOutput(io.Discard)
			handler := middleware.LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeRunnerLiveLogSessionError(w, r, tt.err)
			}))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.status, rec.Code)
			events := transport.Events()
			if tt.status == statusClientClosedRequest {
				assert.Empty(t, events)
				return
			}
			require.Len(t, events, 1)
			assert.Equal(t, "500", events[0].Tags["status"])
			assert.Contains(t, sentrytest.ExceptionText(events[0]), tt.err.Error())
		})
	}
}

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
		r.AuthService, false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
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

	t.Run("hosted pricing follows execution credentials without charging for reads", func(t *testing.T) {
		markup := 2000
		require.NoError(t, billingmodels.UpsertOrganizationLLMMarkup(database.DB(t.Context()), r.Organization.ID, &markup))
		for _, item := range []struct {
			name, component, source string
			priced                  bool
		}{
			{"hosted", models.SuperPlaneRunnerComponent, "hosted", true},
			{"BYOK", "runnerCodex", "integration", false},
		} {
			t.Run(item.name, func(t *testing.T) {
				task := createSessionRunnerTask(t, r)
				canvasID, execID := createCanvasWithComponentExecution(t, r, item.component, "pricing-"+item.name, map[string]any{
					runneraction.ExecutionMetadataBrokerTaskID: task.ID.String(),
					runneraction.ExecutionMetadataTaskBackend:  "integrated",
				})
				require.NoError(t, database.Conn().Model(&models.CanvasNodeExecution{}).Where("id = ?", execID).
					Update("configuration", datatypes.NewJSONType(map[string]any{
						"model": "gpt-4o", "hostedProvider": "openai", "credentials": map[string]any{"source": item.source},
					})).Error)
				var before, after int64
				require.NoError(t, database.Conn().Model(&models.WorkspaceUsageEvent{}).Count(&before).Error)
				rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
				require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
				var session runneraction.LiveLogSession
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
				assert.Equal(t, models.RunnerTaskBackendIntegrated, session.Backend)
				assert.Equal(t, fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-logs", canvasID, execID), session.StreamURL)
				if item.priced {
					require.NotNil(t, session.HostedPricing)
					assert.Equal(t, int64(250), session.HostedPricing.Rate.Input)
					assert.Equal(t, 2000, session.HostedPricing.MarkupBPS)
				} else {
					assert.Nil(t, session.HostedPricing)
				}
				require.NoError(t, database.Conn().Model(&models.WorkspaceUsageEvent{}).Count(&after).Error)
				assert.Equal(t, before, after, "opening live logs must not record or charge usage")
			})
		}
	})

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
		assert.Equal(
			t,
			runneraction.LiveLogSessionNotReadyErrorCode,
			rec.Header().Get(runneraction.LiveLogErrorCodeHeader),
		)
		assert.Contains(t, rec.Body.String(), "not available for this execution")
	})
}

func createSessionRunnerTask(t *testing.T, r *support.ResourceRegistry) models.RunnerTask {
	t.Helper()
	fleet := models.RunnerFleet{
		ID: uuid.New(), Slug: "session-" + uuid.NewString(), ScopeType: models.RunnerFleetScopeInstallation,
		Enabled: true, Spec: datatypes.NewJSONType(models.RunnerFleetSpec{}), RunnerVersion: "0.1.0",
	}
	require.NoError(t, fleet.Create(database.DB(t.Context())))
	return createRunnerTaskForLogTest(t, r, fleet.ID, models.RunnerTaskStateSucceeded)
}
