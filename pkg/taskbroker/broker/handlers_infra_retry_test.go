package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestCompleteTaskRequeuesInfraFailureOverHTTP(t *testing.T) {
	st := openStore(t)

	ctx := context.Background()
	taskID := uuid.NewString()
	require.NoError(t, st.CreateTask(ctx, &models.Task{
		ID:         taskID,
		FleetID:    "fleet-http-retry",
		Status:     models.StatusQueued,
		CreatedAt:  time.Now().UTC(),
		WebhookURL: "https://example.com/hook",
		Commands:   models.CommandList{{Command: "echo hi"}},
	}))
	claimed, err := st.ClaimTask(ctx, "runner-http", "fleet-http-retry", 5*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimed)

	srv := &Server{Store: st, TaskNotify: NewWaitHub()}
	ts := httptest.NewServer(NewRouter(srv, RouterOptions{AuthToken: "token"}))
	t.Cleanup(ts.Close)

	body, err := json.Marshal(api.CompleteTaskRequest{
		RunnerID:    "runner-http",
		ExitCode:    1,
		Error:       "context canceled",
		FailureKind: api.FailureKindRunnerInfra,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/v1/tasks/"+taskID+"/complete", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	got, err := st.GetTask(ctx, taskID)
	require.NoError(t, err)
	require.Equal(t, models.StatusQueued, got.Status)
	require.Equal(t, 1, got.InfraRetryCount)
	require.Empty(t, got.RunnerID)
	require.Nil(t, got.ClaimedAt)
	require.Nil(t, got.LeaseUntil)
}
