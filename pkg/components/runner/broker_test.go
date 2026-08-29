package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateTaskUsesFleetIDOverride(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "local")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-local"}`))},
		},
	}

	client, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	id, err := client.CreateTask(CreateTaskParams{
		MachineType:    MachineTypeE1LargeAMD64,
		Commands:       []BrokerCommand{{Command: "echo hello"}},
		WebhookURL:     "https://example.com/hook",
		ExecutionMode:  ExecutionModeHost,
		TimeoutSeconds: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, "task-local", id)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Equal(t, "local", req.FleetID)
}

func TestCreateTaskUsesMachineTypeWhenOverrideEmpty(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	t.Setenv("TASK_BROKER_FLEET_ID", "")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-1"}`))},
		},
	}

	client, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	_, err = client.CreateTask(CreateTaskParams{
		MachineType:   MachineTypeE1TinyAMD64,
		Commands:      []BrokerCommand{{Command: "echo hello"}},
		WebhookURL:    "https://example.com/hook",
		ExecutionMode: ExecutionModeHost,
	})
	require.NoError(t, err)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Equal(t, MachineTypeE1TinyAMD64, req.FleetID)
}

func TestBrokerUsesUnrestrictedHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		baseURL string
		want    bool
	}{
		{name: "docker desktop host", baseURL: "http://host.docker.internal:8091", want: true},
		{name: "localhost", baseURL: "http://localhost:8091", want: true},
		{name: "loopback", baseURL: "http://127.0.0.1:8091", want: true},
		{name: "rfc1918", baseURL: "http://192.168.65.254:8091", want: true},
		{name: "public broker", baseURL: "https://broker.example", want: false},
		{name: "invalid", baseURL: ":", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, brokerUsesUnrestrictedHTTP(test.baseURL))
		})
	}
}

type ssrfBlockingHTTP struct{}

func (ssrfBlockingHTTP) Do(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("connection blocked: access to private IP address 127.0.0.1 is not allowed")
}

func TestCreateTaskReachesLoopbackBrokerWhenSSRFBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/tasks", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"task-local"}`))
	}))
	t.Cleanup(server.Close)

	t.Setenv("TASK_BROKER_BASE_URL", server.URL)
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	client, err := NewBrokerClient(ssrfBlockingHTTP{})
	require.NoError(t, err)

	id, err := client.CreateTask(CreateTaskParams{
		MachineType:   MachineTypeE1TinyAMD64,
		Commands:      []BrokerCommand{{Command: "echo hello"}},
		WebhookURL:    "https://example.com/hook",
		ExecutionMode: ExecutionModeHost,
	})
	require.NoError(t, err)
	assert.Equal(t, "task-local", id)
}
