package workers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type monitorRequestHTTP func(*http.Request) (*http.Response, error)

func (f monitorRequestHTTP) Do(r *http.Request) (*http.Response, error) { return f(r) }

type monitorRequestIntegration struct {
	core.Integration
	http core.HTTPContext
}

func (integration monitorRequestIntegration) Triggers() []core.Trigger {
	triggers := integration.Integration.Triggers()
	for i := range triggers {
		triggers[i] = monitorRequestTrigger{Trigger: triggers[i], http: integration.http}
	}
	return triggers
}

type monitorRequestTrigger struct {
	core.Trigger
	http core.HTTPContext
}

func (trigger monitorRequestTrigger) HandleHook(ctx core.TriggerHookContext) (map[string]any, error) {
	ctx.HTTP = trigger.http
	return trigger.Trigger.HandleHook(ctx)
}

type monitorRequestFixture struct {
	executor *NodeRequestExecutor
	request  models.CanvasNodeRequest
	node     models.CanvasNode
	r        *support.ResourceRegistry
}

func newMonitorRequestFixture(t *testing.T, httpContext core.HTTPContext) *monitorRequestFixture {
	t.Helper()
	r := support.Setup(t)
	db := database.DB(t.Context())
	id := uuid.New()
	configuration := map[string]any{"site": "datadoghq.eu"}
	for _, key := range []string{"apiKey", "appKey"} {
		value, err := r.Encryptor.Encrypt(t.Context(), []byte("test-key"), []byte(id.String()))
		require.NoError(t, err)
		configuration[key] = base64.StdEncoding.EncodeToString(value)
	}
	integration, err := models.CreateIntegration(id, r.Organization.ID, "datadog", "Datadog", configuration)
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	node := datadogIntakeNode("intake")
	node.Metadata = datatypes.NewJSONType(map[string]any{"subscriptionId": "subscription", "unrelated": "original"})
	node.AppInstallationID = &id
	canvas, nodes := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{node}, nil)
	require.NoError(t, db.Model(&nodes[0]).Update("app_installation_id", id).Error)
	now := time.Now().Add(-time.Second)
	require.NoError(t, nodes[0].CreateRequest(db, models.NodeRequestTypeInvokeAction, models.NodeExecutionRequestSpec{InvokeAction: &models.InvokeAction{ActionName: "reconcileIntakeMonitor", Parameters: map[string]any{}}}, &now))
	request, err := models.FindPendingRequestForNode(db, canvas.ID, node.NodeID)
	require.NoError(t, err)
	implementation := r.Registry.Integrations["datadog"]
	require.NotNil(t, implementation)
	r.Registry.Integrations["datadog"] = monitorRequestIntegration{Integration: implementation, http: httpContext}
	return &monitorRequestFixture{executor: NewNodeRequestExecutor(r.Encryptor, r.Registry, "", r.AuthService), request: *request, node: nodes[0], r: r}
}
func monitorResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}
}

func TestDatadogMonitorRequestReleasesLocksAndMergesMetadata(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	fixture := newMonitorRequestFixture(t, monitorRequestHTTP(func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
		}
		deadline, ok := req.Context().Deadline()
		assert.True(t, ok, "all monitor HTTP must have a deadline")
		assert.LessOrEqual(t, time.Until(deadline), time.Minute)
		if req.Method == http.MethodGet {
			return monitorResponse(200, `{"monitors":[]}`), nil
		}
		return monitorResponse(200, `{"id":42}`), nil
	}))
	done := make(chan error, 1)
	go func() { done <- fixture.executor.Execute(fixture.request) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("monitor HTTP did not start")
	}
	// Always release HTTP, including when a regression assertion fails.
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
		<-done
	}()
	db := database.DB(t.Context())
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := models.LockNodeRequestNow(tx, fixture.request.ID); err != nil {
			return err
		}
		if _, err := models.LockUnscopedCanvas(tx, fixture.node.WorkflowID); err != nil {
			return err
		}
		_, err := models.LockUnscopedCanvasNode(tx, fixture.node.WorkflowID, fixture.node.NodeID)
		return err
	}), "HTTP must release the request lock")
	require.NoError(t, fixture.executor.RunNow(fixture.request))
	require.NoError(t, fixture.executor.Execute(fixture.request))
	require.Equal(t, int32(1), calls.Load(), "duplicate execution must respect the lease")
	latest, err := models.FindCanvasNode(db, fixture.node.WorkflowID, fixture.node.NodeID)
	require.NoError(t, err)
	metadata := latest.Metadata.Data()
	metadata["unrelated"] = "concurrent"
	metadata["factoryTemplate"] = map[string]any{"id": "intake:datadog"}
	require.NoError(t, db.Model(latest).Update("metadata", datatypes.NewJSONType(metadata)).Error)
	close(release)
	// Receive once here and replace the channel for deferred cleanup.
	require.NoError(t, <-done)
	done <- nil
	saved, err := models.FindCanvasNode(db, fixture.node.WorkflowID, fixture.node.NodeID)
	require.NoError(t, err)
	require.Equal(t, "42", saved.Metadata.Data()["monitorId"])
	require.Equal(t, "concurrent", saved.Metadata.Data()["unrelated"])
	require.Equal(t, "subscription", saved.Metadata.Data()["subscriptionId"])
	require.Equal(t, map[string]any{"id": "intake:datadog"}, saved.Metadata.Data()["factoryTemplate"])
	request, err := models.FindCanvasNodeRequest(db, fixture.request.ID)
	require.NoError(t, err)
	require.Equal(t, models.NodeExecutionRequestStateCompleted, request.State)
}

func TestDatadogMonitorRequestRetriesThenReportsFailure(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls atomic.Int32
			f := newMonitorRequestFixture(t, monitorRequestHTTP(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return monitorResponse(status, `{"errors":["provider unavailable"]}`), nil
			}))
			db := database.DB(t.Context())
			for attempt := 1; attempt <= 10; attempt++ {
				before := time.Now()
				require.NoError(t, f.executor.RunNow(f.request))
				request, err := models.FindCanvasNodeRequest(db, f.request.ID)
				require.NoError(t, err)
				if status == http.StatusForbidden || attempt == 10 {
					require.Equal(t, models.NodeExecutionRequestStateCompleted, request.State)
					node, err := models.FindCanvasNode(db, f.node.WorkflowID, f.node.NodeID)
					require.NoError(t, err)
					require.Equal(t, models.CanvasNodeStateError, node.State)
					require.NotNil(t, node.StateReason)
					break
				}
				require.Equal(t, models.NodeExecutionRequestStatePending, request.State)
				require.True(t, request.RunAt.After(before))
				delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second, time.Minute, time.Minute, time.Minute}
				require.WithinDuration(t, before.Add(delays[attempt-1]), request.RunAt, time.Second)
			}
			expected := int32(10)
			if status == http.StatusForbidden {
				expected = 1
			}
			require.Equal(t, expected, calls.Load())
		})
	}
}

func TestDatadogMonitorRequestFencesExpiredLease(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var searches, creates atomic.Int32
	f := newMonitorRequestFixture(t, monitorRequestHTTP(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodGet {
			if searches.Add(1) == 1 {
				close(entered)
				<-release
			}
			return monitorResponse(200, `{"monitors":[]}`), nil
		}
		id := 43
		if creates.Add(1) > 1 {
			id = 42
		}
		return monitorResponse(200, fmt.Sprintf(`{"id":%d}`, id)), nil
	}))
	done := make(chan error, 1)
	go func() { done <- f.executor.Execute(f.request) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("HTTP did not start")
	}
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	db := database.DB(t.Context())
	// A second queued hook must share the first request's durable lease.
	now := time.Now()
	require.NoError(t, f.node.CreateRequest(db, models.NodeRequestTypeInvokeAction, f.request.Spec.Data(), &now))
	var duplicate models.CanvasNodeRequest
	require.NoError(t, db.Where("workflow_id = ? AND id <> ?", f.node.WorkflowID, f.request.ID).First(&duplicate).Error)
	require.NoError(t, f.executor.RunNow(duplicate))
	require.Equal(t, int32(1), searches.Load())
	current, err := models.FindCanvasNodeRequest(db, f.request.ID)
	require.NoError(t, err)
	spec := current.Spec.Data()
	expired := time.Now().Add(-time.Minute)
	spec.DatadogMonitor.LeaseUntil = &expired
	current.Spec = datatypes.NewJSONType(spec)
	require.NoError(t, current.Reschedule(db, expired))
	require.NoError(t, f.executor.Execute(f.request))
	saved, err := models.FindCanvasNode(db, f.node.WorkflowID, f.node.NodeID)
	require.NoError(t, err)
	require.Equal(t, "43", saved.Metadata.Data()["monitorId"])
	close(release)
	require.NoError(t, <-done)
	saved, err = models.FindCanvasNode(db, f.node.WorkflowID, f.node.NodeID)
	require.NoError(t, err)
	require.Equal(t, "43", saved.Metadata.Data()["monitorId"], "expired attempt must not overwrite its successor")
	var cleanups []models.CanvasNodeRequest
	require.NoError(t, db.Where("workflow_id = ? AND state = ?", f.node.WorkflowID, models.NodeExecutionRequestStatePending).Find(&cleanups).Error)
	found := false
	for _, request := range cleanups {
		if state := request.Spec.Data().DatadogMonitor; state != nil && state.RetiredMonitor != nil && state.RetiredMonitor.MonitorID == "42" {
			found = true
		}
	}
	require.True(t, found, "stale create must schedule monitor cleanup")
}

func TestDatadogMonitorRequestDiscardsChangedAndDeletedState(t *testing.T) {
	for _, scenario := range []string{"configuration", "node deletion", "canvas deletion"} {
		t.Run(scenario, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			var deletes atomic.Int32
			var created []byte
			f := newMonitorRequestFixture(t, monitorRequestHTTP(func(req *http.Request) (*http.Response, error) {
				switch req.Method {
				case http.MethodGet:
					if strings.HasSuffix(req.URL.Path, "/search") {
						select {
						case <-entered:
						default:
							close(entered)
							<-release
						}
						return monitorResponse(200, `{"monitors":[]}`), nil
					}
					return monitorResponse(200, string(created)), nil
				case http.MethodPost:
					body, err := io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					var monitor map[string]any
					if err := json.Unmarshal(body, &monitor); err != nil {
						return nil, err
					}
					monitor["id"] = 42
					created, err = json.Marshal(monitor)
					return monitorResponse(200, string(created)), err
				case http.MethodDelete:
					deletes.Add(1)
					return monitorResponse(200, `{}`), nil
				}
				return nil, fmt.Errorf("unexpected method %s", req.Method)
			}))
			done := make(chan error, 1)
			go func() { done <- f.executor.Execute(f.request) }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("HTTP did not start")
			}
			defer func() {
				select {
				case <-release:
				default:
					close(release)
				}
			}()
			db := database.DB(t.Context())
			switch scenario {
			case "configuration":
				require.NoError(t, db.Model(&f.node).Update("configuration", datatypes.NewJSONType(map[string]any{"service": "billing"})).Error)
				require.NoError(t, contexts.NewNodeRequestContext(db, &f.node).ScheduleActionCall("reconcileIntakeMonitor", map[string]any{}, time.Second))
				pending, err := models.FindPendingRequestForNode(db, f.node.WorkflowID, f.node.NodeID)
				require.NoError(t, err)
				require.NoError(t, f.executor.RunNow(*pending))
				waiting, err := models.FindCanvasNodeRequest(db, pending.ID)
				require.NoError(t, err)
				require.True(t, waiting.RunAt.After(time.Now().Add(4*time.Minute)), "a superseded request still owns its active HTTP lease")
			case "node deletion":
				require.NoError(t, models.DeleteCanvasNode(db, f.node))
			case "canvas deletion":
				canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, f.node.WorkflowID)
				require.NoError(t, err)
				require.NoError(t, canvas.SoftDelete())
			}
			close(release)
			require.NoError(t, <-done)
			saved, err := models.FindUnscopedCanvasNode(db, f.node.WorkflowID, f.node.NodeID)
			require.NoError(t, err)
			require.Empty(t, saved.Metadata.Data()["monitorId"])
			var requests []models.CanvasNodeRequest
			require.NoError(t, db.Where("workflow_id = ? AND state = ?", f.node.WorkflowID, models.NodeExecutionRequestStatePending).Find(&requests).Error)
			if scenario != "configuration" {
				past := time.Now().Add(-time.Second)
				for i := range requests {
					require.NoError(t, requests[i].Reschedule(db, past))
				}
				due, err := models.ListDueRetiredDatadogMonitorRequests(db)
				require.NoError(t, err)
				require.NotEmpty(t, due, "cleanup must remain eligible for a deleted node or canvas")
			}
			found := false
			for _, request := range requests {
				state := request.Spec.Data().DatadogMonitor
				if state == nil || state.RetiredMonitor == nil {
					if scenario == "configuration" {
						require.True(t, request.RunAt.Before(time.Now().Add(2*time.Second)), "finishing a stale lease must wake current-state reconciliation")
					}
					continue
				}
				found = true
				require.NoError(t, f.executor.RunNow(request))
			}
			require.True(t, found)
			require.Equal(t, int32(1), deletes.Load(), "retired owned monitor must be removed after committed changes")
		})
	}
}

func TestDatadogMonitorRequestRejectsOtherOrganizationBeforeHTTP(t *testing.T) {
	var calls atomic.Int32
	f := newMonitorRequestFixture(t, monitorRequestHTTP(func(*http.Request) (*http.Response, error) { calls.Add(1); return monitorResponse(200, `{}`), nil }))
	other := support.CreateOrganization(t, f.r, f.r.User)
	integration, err := models.CreateIntegration(uuid.New(), other.ID, "datadog", "Other", map[string]any{})
	require.NoError(t, err)
	db := database.DB(t.Context())
	require.NoError(t, db.Model(&f.node).Update("app_installation_id", integration.ID).Error)
	require.NoError(t, f.executor.RunNow(f.request))
	require.Zero(t, calls.Load())
	saved, err := models.FindCanvasNode(db, f.node.WorkflowID, f.node.NodeID)
	require.NoError(t, err)
	require.Equal(t, models.CanvasNodeStateError, saved.State)
	request, err := models.FindCanvasNodeRequest(db, f.request.ID)
	require.NoError(t, err)
	require.Equal(t, models.NodeExecutionRequestStateCompleted, request.State)
}
