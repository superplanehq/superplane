package public

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
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
		r.AuthService, false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	return server, signer
}

func TestRunnerLiveLogSessionDiagnostics(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	for _, item := range []struct {
		name, operation, schema, message string
		backend                          string
		failBeforeWrite                  bool
		status                           int
	}{
		{name: "authorization", operation: "authorization", message: "Authorization check failed", status: 500},
		{name: "access lookup", operation: "access_lookup", schema: "Canvas", message: "Lookup failed", status: 500},
		{name: "pricing lookup", operation: "pricing_lookup", schema: "OrganizationLLMSettings", message: "Lookup failed", status: 500},
		{name: "missing broker URL", operation: "session_creation", message: "Live logs are not configured on this installation", status: 503},
		{name: "missing broker secret", operation: "session_creation", message: "Live logs are not configured on this installation", status: 503},
		{name: "session signing", operation: "session_creation", message: "Could not create live log session", status: 500},
		{name: "integrated write", operation: "response_encoding", backend: "integrated", status: 200},
		{name: "legacy write", operation: "response_encoding", backend: "legacy", message: "Failed to encode response", status: 200},
		{name: "legacy write before headers", operation: "response_encoding", backend: "legacy", failBeforeWrite: true, message: "Failed to encode response", status: 500},
	} {
		t.Run(item.name, func(t *testing.T) {
			t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
			t.Setenv("TASK_BROKER_PUBLIC_URL", "")
			t.Setenv("TASK_BROKER_AUTH_TOKEN", "diagnostic-test-secret")
			canvasID, executionID := createCanvasWithComponentExecution(t, r, models.SuperPlaneRunnerComponent, "diagnostics", map[string]any{
				runneraction.ExecutionMetadataBrokerTaskID: "diagnostic-task",
				runneraction.ExecutionMetadataTaskBackend:  item.backend,
			})
			require.NoError(t, database.Conn().Model(&models.CanvasNodeExecution{}).Where("id = ?", executionID).
				Update("configuration", datatypes.NewJSONType(map[string]any{
					"model": "gpt-4o", "hostedProvider": "openai", "credentials": map[string]any{"source": "hosted"},
				})).Error)

			cause := errors.New("controlled " + item.name + " failure")
			server := &Server{authService: r.AuthService}
			switch item.name {
			case "authorization":
				server.authService = liveLogFailingAuthorization{Authorization: r.AuthService, cause: cause}
			case "missing broker URL":
				t.Setenv("TASK_BROKER_BASE_URL", "")
				cause = fmt.Errorf("%w: TASK_BROKER_BASE_URL is not set", runneraction.ErrLiveLogNotConfigured)
			case "missing broker secret":
				t.Setenv("TASK_BROKER_AUTH_TOKEN", "")
				cause = fmt.Errorf("%w: TASK_BROKER_AUTH_TOKEN is not set", runneraction.ErrLiveLogNotConfigured)
			case "session signing":
				previous := gojwt.SigningMethodHS256.Hash
				gojwt.SigningMethodHS256.Hash = crypto.Hash(0)
				t.Cleanup(func() { gojwt.SigningMethodHS256.Hash = previous })
				cause = fmt.Errorf("sign live log token: %w", gojwt.ErrHashUnavailable)
			}
			if item.schema != "" {
				queries := database.Conn().Callback().Query()
				const callback = "test:live_log_failure"
				require.NoError(t, queries.Before("gorm:query").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Schema != nil && tx.Statement.Schema.Name == item.schema {
						_ = tx.AddError(cause)
					}
				}))
				t.Cleanup(func() { require.NoError(t, queries.Remove(callback)) })
			}

			transport := &liveLogCaptureTransport{}
			client, err := sentry.NewClient(sentry.ClientOptions{Dsn: "https://public@localhost/1", Transport: transport})
			require.NoError(t, err)
			hub := sentry.CurrentHub()
			previous := hub.Client()
			hub.BindClient(client)
			t.Cleanup(func() { hub.BindClient(previous) })
			var buffer bytes.Buffer
			logger := log.New()
			logger.SetFormatter(&log.JSONFormatter{})
			logger.SetOutput(&buffer)

			path := fmt.Sprintf("/api/v1/canvases/%s/node-executions/%s/runner-live-logs/session", canvasID, executionID)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-Request-ID", "live-log-request")
			req.Header.Set("X-Cloud-Trace-Context", "live-log-trace/1;o=1")
			user, err := models.FindActiveUserByID(r.Organization.ID.String(), r.User.String())
			require.NoError(t, err)
			req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
			req = mux.SetURLVars(req, map[string]string{"canvas_id": canvasID.String(), "execution_id": executionID.String()})
			rec := httptest.NewRecorder()
			var writer http.ResponseWriter = rec
			var failingWriter *liveLogFailingWriter
			if item.operation == "response_encoding" {
				failingWriter = &liveLogFailingWriter{ResponseRecorder: rec, cause: cause, failBeforeWrite: item.failBeforeWrite}
				writer = failingWriter
			}
			var noted grpc.NotedServerError
			var hasCause bool
			handler := middleware.LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				middleware.SetRequestLogUser(request.Context(), user)
				server.handleRunnerLiveLogSession(w, request)
				noted, hasCause = grpc.NotedServerErrorFrom(request.Context())
			}))
			handler.ServeHTTP(writer, req)

			require.Equal(t, item.status, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), item.message)
			assert.NotContains(t, rec.Body.String(), cause.Error())
			require.True(t, hasCause, "handler must retain the underlying error")
			assert.Equal(t, cause.Error(), noted.Cause.Error())
			if item.operation == "authorization" || item.schema != "" || item.operation == "response_encoding" {
				assert.Same(t, cause, noted.Cause)
			}
			assert.Equal(t, map[string]string{
				"operation": item.operation, "organization_id": r.Organization.ID.String(),
				"canvas_id": canvasID.String(), "execution_id": executionID.String(),
			}, noted.Tags)
			if item.operation == "response_encoding" {
				assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
			}

			var entry map[string]any
			require.NoError(t, json.Unmarshal(buffer.Bytes(), &entry))
			assert.Equal(t, path, entry["path"])
			assert.Equal(t, "live-log-request", entry["request_id"])
			assert.Equal(t, "live-log-trace", entry["trace_id"])
			assert.Equal(t, user.OrganizationID.String(), entry["organization_id"])
			assert.Equal(t, user.ID.String(), entry["user_id"])
			reportedStatus := item.status
			if item.backend == "legacy" && item.operation == "response_encoding" {
				// The existing legacy error response attempts a 500 even after bytes were sent.
				reportedStatus = http.StatusInternalServerError
			}
			assert.Equal(t, float64(reportedStatus), entry["status"])
			if reportedStatus >= 500 {
				assert.Equal(t, cause.Error(), entry["error"])
				events := transport.Events()
				require.Len(t, events, 1)
				require.NotEmpty(t, events[0].Exception)
				assert.Equal(t, cause.Error(), events[0].Exception[len(events[0].Exception)-1].Value)
				assert.Equal(t, fmt.Sprint(reportedStatus), events[0].Tags["status"])
				assert.Equal(t, item.operation, events[0].Tags["operation"])
				require.NotNil(t, events[0].Request)
				assert.Equal(t, "http://example.com"+path, events[0].Request.URL)
			} else {
				assert.Empty(t, transport.Events(), "a committed successful response keeps its reporting behavior")
			}
			diagnostics, err := json.Marshal(transport.Events())
			require.NoError(t, err)
			assert.NotContains(t, string(diagnostics), "diagnostic-test-secret")
			assert.NotContains(t, string(diagnostics), "\"token\"")
			assert.NotContains(t, buffer.String(), "diagnostic-test-secret")
			if failingWriter != nil && item.backend == "legacy" {
				require.NotEmpty(t, failingWriter.sessionToken)
				assert.NotContains(t, string(diagnostics), failingWriter.sessionToken)
				assert.NotContains(t, buffer.String(), failingWriter.sessionToken)
			}
		})
	}
}

func TestRunnerLiveLogSessionExpectedErrorsDoNotRecordCauses(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	user, err := models.FindActiveUserByID(r.Organization.ID.String(), r.User.String())
	require.NoError(t, err)
	canvasID, executionID := createCanvasWithComponentExecution(t, r, "runner", "expected-errors", nil)
	nonRunnerCanvasID, nonRunnerExecutionID := createCanvasWithComponentExecution(t, r, "noop", "non-runner", nil)
	for _, item := range []struct {
		name, canvas, execution string
		status                  int
	}{
		{"unauthenticated", canvasID.String(), executionID.String(), http.StatusUnauthorized},
		{"permission denied", canvasID.String(), executionID.String(), http.StatusForbidden},
		{"invalid canvas", "invalid", executionID.String(), http.StatusBadRequest},
		{"invalid execution", canvasID.String(), "invalid", http.StatusBadRequest},
		{"missing canvas", uuid.NewString(), executionID.String(), http.StatusNotFound},
		{"missing execution", canvasID.String(), uuid.NewString(), http.StatusNotFound},
		{"non-runner", nonRunnerCanvasID.String(), nonRunnerExecutionID.String(), http.StatusBadRequest},
		{"not ready", canvasID.String(), executionID.String(), http.StatusNotFound},
	} {
		t.Run(item.name, func(t *testing.T) {
			server := &Server{authService: r.AuthService}
			if item.name == "permission denied" {
				server.authService = liveLogFailingAuthorization{Authorization: r.AuthService}
			}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/canvases/"+item.canvas+"/node-executions/"+item.execution+"/runner-live-logs/session", nil)
			if item.name != "unauthenticated" {
				req = req.WithContext(context.WithValue(req.Context(), middleware.UserContextKey, user))
			}
			req = grpc.WithServerErrorReport(mux.SetURLVars(req, map[string]string{"canvas_id": item.canvas, "execution_id": item.execution}))
			rec := httptest.NewRecorder()
			server.handleRunnerLiveLogSession(rec, req)
			assert.Equal(t, item.status, rec.Code)
			_, noted := grpc.NotedServerErrorFrom(req.Context())
			assert.False(t, noted)
			if item.name == "not ready" {
				assert.Equal(t, runneraction.LiveLogSessionNotReadyErrorCode, rec.Header().Get(runneraction.LiveLogErrorCodeHeader))
			}
		})
	}
}

type liveLogFailingAuthorization struct {
	authorization.Authorization
	cause error
}

func (a liveLogFailingAuthorization) CheckOrganizationPermission(context.Context, string, string, string, string) (bool, error) {
	return false, a.cause
}

type liveLogFailingWriter struct {
	*httptest.ResponseRecorder
	cause           error
	failed          bool
	failBeforeWrite bool
	sessionToken    string
}

func (w *liveLogFailingWriter) Write(payload []byte) (int, error) {
	if w.failed {
		return w.ResponseRecorder.Write(payload)
	}
	w.failed = true
	var session runneraction.LiveLogSession
	if json.Unmarshal(payload, &session) == nil {
		w.sessionToken = session.Token
	}
	if w.failBeforeWrite {
		return 0, w.cause
	}
	// Commit a partial response before the transport fails.
	n, _ := w.ResponseRecorder.Write(payload[:1])
	return n, w.cause
}

type liveLogCaptureTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (*liveLogCaptureTransport) Configure(sentry.ClientOptions) {}
func (*liveLogCaptureTransport) Flush(time.Duration) bool       { return true }
func (t *liveLogCaptureTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}
func (t *liveLogCaptureTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]*sentry.Event(nil), t.events...)
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
				canvasID, execID := createCanvasWithComponentExecution(t, r, item.component, "pricing-"+item.name, map[string]any{
					runneraction.ExecutionMetadataBrokerTaskID: "task-pricing",
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

	t.Run("returns stream session with public broker URL", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "http://task-broker:8081")
		t.Setenv("TASK_BROKER_PUBLIC_URL", "http://localhost:8091")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-public-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-public-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "http://localhost:8091/v1/tasks/task-public-ok/live-logs", session.StreamURL)
	})

	t.Run("returns organization-authorized integrated stream", func(t *testing.T) {
		canvasID, execID := createCanvasWithComponentExecution(t, r, "runner", "runner-integrated-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-integrated",
			runneraction.ExecutionMetadataTaskBackend:  "integrated",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "integrated", session.Backend)
		assert.Equal(
			t,
			fmt.Sprintf(
				"/api/v1/canvases/%s/node-executions/%s/runner-logs",
				canvasID,
				execID,
			),
			session.StreamURL,
		)
		assert.Empty(t, session.Token)
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

	t.Run("returns stream session for runnerClaudeCode", func(t *testing.T) {
		t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
		t.Setenv("TASK_BROKER_AUTH_TOKEN", "live-log-secret")

		canvasID, execID := createCanvasWithComponentExecution(t, r, "runnerClaudeCode", "runner-claude-1", map[string]any{
			runneraction.ExecutionMetadataBrokerTaskID: "task-claude-ok",
		})
		rec := runnerLiveLogSessionGET(t, server, signer, r, canvasID.String(), execID.String())
		assert.Equal(t, http.StatusOK, rec.Code)

		var session runneraction.LiveLogSession
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &session))
		assert.Equal(t, "https://broker.example/v1/tasks/task-claude-ok/live-logs", session.StreamURL)
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
