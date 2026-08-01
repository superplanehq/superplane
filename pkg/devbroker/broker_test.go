package devbroker

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAuthToken = "dev-token"

// createTask posts a task and returns the broker-assigned ID.
func createTask(t *testing.T, srv *httptest.Server, body map[string]any) string {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/tasks", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testAuthToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	// SuperPlane's broker client requires exactly 201 here (broker.go:254).
	require.Equal(t, http.StatusCreated, res.StatusCode)

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&created))
	require.NotEmpty(t, created.ID)

	return created.ID
}

func TestServer_RunsCommandsAndCallsWebhook(t *testing.T) {
	webhooks := make(chan map[string]any, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		webhooks <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	taskID := createTask(t, srv, map[string]any{
		"fleet_id":    "e1-large-amd64",
		"webhook_url": callback.URL,
		"commands": []map[string]string{
			{"command": "echo hello"},
		},
	})

	select {
	case payload := <-webhooks:
		assert.Equal(t, taskID, payload["task_id"])
		assert.Equal(t, "succeeded", payload["status"])
		assert.EqualValues(t, 0, payload["exit_code"])
		assert.Contains(t, payload["output"], "hello")
	case <-time.After(10 * time.Second):
		t.Fatal("no webhook received")
	}
}

func TestServer_ReportsFailedCommands(t *testing.T) {
	webhooks := make(chan map[string]any, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		webhooks <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	createTask(t, srv, map[string]any{
		"fleet_id":    "e1-large-amd64",
		"webhook_url": callback.URL,
		"commands": []map[string]string{
			{"command": "exit 3"},
		},
	})

	select {
	case payload := <-webhooks:
		assert.Equal(t, "failed", payload["status"])
		assert.EqualValues(t, 3, payload["exit_code"])
	case <-time.After(10 * time.Second):
		t.Fatal("no webhook received")
	}
}

func TestServer_ReturnsResultFileContents(t *testing.T) {
	webhooks := make(chan map[string]any, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		webhooks <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	createTask(t, srv, map[string]any{
		"fleet_id":    "e1-large-amd64",
		"webhook_url": callback.URL,
		"commands": []map[string]string{
			// Runner scripts write their structured result here; the runner is
			// responsible for creating the file and reporting its contents.
			{"command": `echo '{"pr":42}' > "$SUPERPLANE_RESULT_FILE"`},
		},
	})

	select {
	case payload := <-webhooks:
		require.Equal(t, "succeeded", payload["status"])
		result, ok := payload["result"].(map[string]any)
		require.True(t, ok, "result missing from webhook payload: %v", payload["result"])
		assert.EqualValues(t, 42, result["pr"])
	case <-time.After(10 * time.Second):
		t.Fatal("no webhook received")
	}
}

func TestServer_ProvidesPayloadFile(t *testing.T) {
	webhooks := make(chan map[string]any, 1)
	callback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		webhooks <- payload
		w.WriteHeader(http.StatusOK)
	}))
	defer callback.Close()

	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	createTask(t, srv, map[string]any{
		"fleet_id":    "e1-large-amd64",
		"webhook_url": callback.URL,
		"commands": []map[string]string{
			{"command": `test -f "$SUPERPLANE_PAYLOAD_FILE" && cat "$SUPERPLANE_PAYLOAD_FILE"`},
		},
	})

	select {
	case payload := <-webhooks:
		assert.Equal(t, "succeeded", payload["status"])
		assert.Contains(t, payload["output"], "{")
	case <-time.After(10 * time.Second):
		t.Fatal("no webhook received")
	}
}

func TestServer_RejectsWrongToken(t *testing.T) {
	srv := httptest.NewServer(New(Options{AuthToken: testAuthToken, WorkDir: t.TempDir()}).Handler())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v1/tasks", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer wrong")

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
}
