package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/webhook"
	brokermetrics "github.com/superplane/runner/task-broker/internal/metrics"
	brokermodels "github.com/superplane/runner/task-broker/internal/models"
	"github.com/superplane/runner/task-broker/internal/store/testdb"
)

func testBrokerMetrics(t *testing.T) (*brokermetrics.BrokerMetrics, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	m, err := brokermetrics.New(provider.Meter("test"))
	require.NoError(t, err)
	return m, reader
}

func metricCounterTotal(t *testing.T, reader *metric.ManualReader, name string) int64 {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum := m.Data.(metricdata.Sum[int64])
			var total int64
			for _, dp := range sum.DataPoints {
				total += dp.Value
			}
			return total
		}
	}
	return 0
}

func TestCreateTaskRecordsTasksCreatedMetric(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	require.NoError(t, st.CreateFleet(context.Background(), &brokermodels.Fleet{
		ID:          "fleet-1",
		Provisioner: "local",
		Arch:        "amd64",
		Size:        "local",
		CreatedAt:   time.Now().UTC(),
	}))

	m, reader := testBrokerMetrics(t)
	srv := &Server{Store: st, Metrics: m}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body, err := json.Marshal(api.BrokerCreateTaskRequest{
		CreateTaskRequest: api.CreateTaskRequest{
			Commands:   []string{"echo hi"},
			WebhookURL: "https://example.com/hook",
		},
		FleetID: "fleet-1",
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, strings.TrimSpace(string(respBody)))

	require.Equal(t, int64(1), metricCounterTotal(t, reader, "tasks.created"))
}

func TestCompleteTaskRecordsTasksCompletedMetric(t *testing.T) {
	st, cleanup := testdb.Open(t)
	defer cleanup()
	ctx := context.Background()
	require.NoError(t, st.CreateFleet(ctx, &brokermodels.Fleet{
		ID: "fleet-1", Provisioner: "local", Arch: "amd64", Size: "local", CreatedAt: time.Now().UTC(),
	}))
	require.NoError(t, st.CreateTask(ctx, &models.Task{
		ID: "task-1", FleetID: "fleet-1", WebhookURL: "https://example.com/hook",
		Status: models.StatusQueued, CreatedAt: time.Now().UTC(), Command: []string{"echo"},
	}))
	_, err := st.ClaimTask(ctx, "runner-1", "fleet-1", time.Minute)
	require.NoError(t, err)

	m, reader := testBrokerMetrics(t)
	srv := &Server{Store: st, Metrics: m}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	defer ts.Close()

	body, err := json.Marshal(api.CompleteTaskRequest{RunnerID: "runner-1", ExitCode: 0})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/task-1/complete", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	require.Equal(t, int64(1), metricCounterTotal(t, reader, "tasks.completed"))
}

func TestDeliverWebhookRecordsWebhookMetrics(t *testing.T) {
	m, reader := testBrokerMetrics(t)

	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer okSrv.Close()

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()

	sender := &webhook.Sender{
		Client:  okSrv.Client(),
		Retries: 1,
	}
	srv := &Server{
		Metrics: m,
		Webhook: sender,
	}

	srv.DeliverWebhook(&models.Task{
		ID: "task-ok", FleetID: "fleet-1", WebhookURL: okSrv.URL, Status: models.StatusSucceeded,
	})
	require.Equal(t, int64(1), metricCounterTotal(t, reader, "webhook.deliveries"))

	sender.Client = failSrv.Client()
	srv.DeliverWebhook(&models.Task{
		ID: "task-fail", FleetID: "fleet-1", WebhookURL: failSrv.URL, Status: models.StatusSucceeded,
	})
	require.Equal(t, int64(2), metricCounterTotal(t, reader, "webhook.deliveries"))
}
