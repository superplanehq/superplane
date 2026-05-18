package runner

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/runners"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// mockStore is a test-only in-memory implementation of fleetStore.
type mockStore struct {
	fleet *runners.RunnerFleet
	tasks []*runners.RunnerTask
}

func (m *mockStore) FindFleet(id uuid.UUID) (*runners.RunnerFleet, error) {
	return m.fleet, nil
}

func (m *mockStore) CreateTask(id uuid.UUID, fleetID uuid.UUID, fleetTaskID string, executionID uuid.UUID) (*runners.RunnerTask, error) {
	t := &runners.RunnerTask{ID: id, FleetID: fleetID, FleetTaskID: fleetTaskID, ExecutionID: executionID}
	m.tasks = append(m.tasks, t)
	return t, nil
}

func testFleet() *runners.RunnerFleet {
	return &runners.RunnerFleet{
		ID:        uuid.New(),
		Name:      "fleet-1",
		FleetURL:  "https://broker.example",
		AuthToken: "token-1",
	}
}

func testRunner(fleet *runners.RunnerFleet) *Runner {
	return &Runner{store: &mockStore{fleet: fleet}}
}

func TestNormalizeCommands(t *testing.T) {
	t.Parallel()
	got := normalizeCommands("echo a\n\n  echo b  \n")
	want := []string{"echo a", "echo b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCommands: got %#v want %#v", got, want)
	}
	if len(normalizeCommands("")) != 0 {
		t.Fatal("empty input should yield empty slice")
	}
	if len(normalizeCommands("\n \n")) != 0 {
		t.Fatal("blank-only lines should yield empty slice")
	}
}

func TestValidateEnvironment(t *testing.T) {
	t.Parallel()

	value := func(v string) *string { return &v }

	tests := []struct {
		name        string
		environment []EnvironmentVariable
		errContains string
	}{
		{
			name: "valid literal",
			environment: []EnvironmentVariable{
				{Name: "COMMIT_AUTHOR", ValueSource: EnvironmentValueSourceLiteral, Value: value("alice@example.com")},
			},
		},
		{
			name: "valid secret",
			environment: []EnvironmentVariable{
				{Name: "API_TOKEN", ValueSource: EnvironmentValueSourceSecret, Secret: secretRef("api", "token")},
			},
		},
		{
			name: "missing name",
			environment: []EnvironmentVariable{
				{ValueSource: EnvironmentValueSourceLiteral, Value: value("alice@example.com")},
			},
			errContains: "environment[0].name is required",
		},
		{
			name: "invalid name",
			environment: []EnvironmentVariable{
				{Name: "BAD-NAME", ValueSource: EnvironmentValueSourceLiteral, Value: value("x")},
			},
			errContains: "invalid environment variable name",
		},
		{
			name: "duplicate name",
			environment: []EnvironmentVariable{
				{Name: "API_TOKEN", ValueSource: EnvironmentValueSourceLiteral, Value: value("a")},
				{Name: "API_TOKEN", ValueSource: EnvironmentValueSourceLiteral, Value: value("b")},
			},
			errContains: "duplicate environment variable name",
		},
		{
			name: "invalid value source",
			environment: []EnvironmentVariable{
				{Name: "API_TOKEN", ValueSource: "vault", Value: value("x")},
			},
			errContains: "invalid environment variable value source",
		},
		{
			name: "missing literal value",
			environment: []EnvironmentVariable{
				{Name: "API_TOKEN", ValueSource: EnvironmentValueSourceLiteral},
			},
			errContains: "value is required",
		},
		{
			name: "missing secret ref",
			environment: []EnvironmentVariable{
				{Name: "API_TOKEN", ValueSource: EnvironmentValueSourceSecret},
			},
			errContains: "secret.secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateEnvironment(tt.environment)
			if tt.errContains == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.errContains)
		})
	}
}

func TestRunnerExecuteSendsEnvironmentToFleet(t *testing.T) {
	fleet := testFleet()
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-123"}`))},
		},
	}

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requests := &contexts.RequestContext{}
	component := testRunner(fleet)

	err := component.Execute(core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"fleet_id": fleet.ID.String(),
			"commands": "echo hello",
			"environment": []map[string]any{
				{
					"name":        "COMMIT_AUTHOR",
					"valueSource": EnvironmentValueSourceLiteral,
					"value":       "alice@example.com",
				},
				{
					"name":        "API_TOKEN",
					"valueSource": EnvironmentValueSourceSecret,
					"secret": map[string]any{
						"secret": "api",
						"key":    "token",
					},
				},
			},
		},
		HTTP:           httpContext,
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"api/token": []byte("secret'value;$PATH")}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: state,
		Requests:       requests,
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req runners.FleetCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, "fleet-1", req.FleetID)
	assert.Equal(t, []string{"echo hello"}, req.Commands)
	assert.Equal(t, "host", req.ExecutionMode)
	assert.Equal(t, []runners.FleetEnvironmentVariable{
		{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
		{Name: "API_TOKEN", Value: "secret'value;$PATH"},
	}, req.Environment)
	assert.Equal(t, "task-123", state.KVs["task_id"])
	assert.Equal(t, hookActionPoll, requests.Action)
}

func TestRunnerExecuteOmitsEmptyEnvironment(t *testing.T) {
	fleet := testFleet()
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-123"}`))},
		},
	}

	err := testRunner(fleet).Execute(core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  map[string]any{"fleet_id": fleet.ID.String(), "commands": "echo hello"},
		HTTP:           httpContext,
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.NotContains(t, string(body), "environment")
}

func TestRunnerExecuteFailsWhenSecretCannotBeResolved(t *testing.T) {
	fleet := testFleet()
	httpContext := &contexts.HTTPContext{}

	err := testRunner(fleet).Execute(core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"fleet_id": fleet.ID.String(),
			"commands": "echo hello",
			"environment": []map[string]any{
				{
					"name":        "API_TOKEN",
					"valueSource": EnvironmentValueSourceSecret,
					"secret": map[string]any{
						"secret": "api",
						"key":    "token",
					},
				},
			},
		},
		HTTP:           httpContext,
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})

	require.ErrorContains(t, err, "failed to resolve environment variable API_TOKEN")
	assert.Empty(t, httpContext.Requests)
}

func secretRef(secret, key string) configuration.SecretKeyRef {
	return configuration.SecretKeyRef{Secret: secret, Key: key}
}

func TestRunnerProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &runners.FleetTask{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`{"items":[1,2],"ok":true}`),
	}
	require.NoError(t, (&Runner{}).processTaskStatus(state, task))
	require.Equal(t, PassedOutputChannel, state.Channel)

	wrapped := state.Payloads[0].(map[string]any)
	data := wrapped["data"].(map[string]any)
	assert.Equal(t, "succeeded", data["status"])
	assert.Equal(t, 0, data["exit_code"])
	result := data["result"].(map[string]any)
	assert.Equal(t, true, result["ok"])
	assert.Equal(t, []any{float64(1), float64(2)}, result["items"])
}

func TestRunnerProcessTaskStatusOmitsInvalidResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &runners.FleetTask{
		Status:   "succeeded",
		ExitCode: &exit,
		Result:   json.RawMessage(`not-json`),
	}
	require.NoError(t, (&Runner{}).processTaskStatus(state, task))
	wrapped := state.Payloads[0].(map[string]any)
	data := wrapped["data"].(map[string]any)
	_, ok := data["result"]
	assert.False(t, ok)
}
