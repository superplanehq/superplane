package runner

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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

func TestNormalizeCommandsPreservesShellBlocks(t *testing.T) {
	t.Parallel()

	input := `
echo before
if ! command -v aws >/dev/null 2>&1; then
  apt-get update
  for package in curl unzip; do
    apt-get install -y "$package"
  done
fi
echo ready
`
	want := []string{
		"echo before",
		`if ! command -v aws >/dev/null 2>&1; then
apt-get update
for package in curl unzip; do
apt-get install -y "$package"
done
fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsIgnoresQuotedShellClosers(t *testing.T) {
	t.Parallel()

	input := `
if true; then
  echo "; fi"
  echo "}"
fi
echo ready
`
	want := []string{
		`if true; then
echo "; fi"
echo "}"
fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsIgnoresHeredocBodyShellWords(t *testing.T) {
	t.Parallel()

	input := `
if true; then
  cat <<'EOF'
fi
done
  indented

EOF
fi
echo ready
`
	want := []string{
		`if true; then
cat <<'EOF'
fi
done
  indented

EOF
fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsDoesNotTreatHereStringAsHeredoc(t *testing.T) {
	t.Parallel()

	input := `
cat <<< "foo"
echo ready
`
	want := []string{
		`cat <<< "foo"`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsIgnoresArgumentShellClosers(t *testing.T) {
	t.Parallel()

	input := `
if true; then
  echo fi
  apt-get install -y done
fi
echo ready
`
	want := []string{
		`if true; then
echo fi
apt-get install -y done
fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsCountsMultipleCommandPositionClosers(t *testing.T) {
	t.Parallel()

	input := `
if true; then
  for package in curl; do
    echo "$package"
  done; fi
echo ready
`
	want := []string{
		`if true; then
for package in curl; do
echo "$package"
done; fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsPreservesChainedShellBlocks(t *testing.T) {
	t.Parallel()

	input := `
command -v aws >/dev/null 2>&1 || if true; then
  echo install
fi
echo ready
`
	want := []string{
		`command -v aws >/dev/null 2>&1 || if true; then
echo install
fi`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsIgnoresBacktickCommandSubstitution(t *testing.T) {
	t.Parallel()

	input := `
if true; then
  echo ` + "`echo fi`" + `
fi
echo ready
`
	want := []string{
		"if true; then\necho `echo fi`\nfi",
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
}

func TestNormalizeCommandsPreservesFunctionBlocks(t *testing.T) {
	t.Parallel()

	input := `
function install_tools {
  echo install
}
echo ready
`
	want := []string{
		`function install_tools {
echo install
}`,
		"echo ready",
	}

	assert.Equal(t, want, normalizeCommands(input))
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

func TestRunnerExecuteSendsEnvironmentToBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-123"}`))},
		},
	}

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requests := &contexts.RequestContext{}
	component := &Runner{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"commands":     "echo hello",
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

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, testRunnerMachineType, req.FleetID)
	assert.Equal(t, []string{"echo hello"}, req.Commands)
}

func TestRunnerExecuteUsesConfiguredMachineType(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-456"}`))},
		},
	}

	err := (&Runner{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"commands":     "echo hi",
			"machine_type": "aws-arm64-1",
		},
		HTTP:           httpContext,
		Webhook:        &contexts.NodeWebhookContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))
	assert.Equal(t, "aws-arm64-1", req.FleetID)
}

func TestRunnerExecuteOmitsEmptyEnvironment(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-123"}`))},
		},
	}

	err := (&Runner{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"commands":     "echo hello",
		},
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
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{}

	err := (&Runner{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"machine_type": testRunnerMachineType,
			"commands":     "echo hello",
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
	task := &Task{
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
	task := &Task{
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

func TestRunnerProcessTaskStatusCanceledUsesFailedChannel(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 130
	task := &Task{
		Status:   "canceled",
		ExitCode: &exit,
	}
	require.NoError(t, (&Runner{}).processTaskStatus(state, task))
	require.Equal(t, FailedOutputChannel, state.Channel)
}

func TestBrokerCancelTaskSuccess(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"fleet-1","state":"canceled","status":"canceled"}`))},
		},
	}

	broker, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	require.NoError(t, broker.CancelTask("broker-task-99"))
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "/v1/tasks/broker-task-99/cancel", httpContext.Requests[0].URL.Path)
	assert.Equal(t, "Bearer token-1", httpContext.Requests[0].Header.Get("Authorization"))
}

func TestBrokerCancelTask404Noop(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"task not found"}`))},
		},
	}

	broker, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	require.NoError(t, broker.CancelTask("missing"))
}

func TestBrokerCancelTask409RetriesThenSucceeds(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader(`{"error":"task not yet assigned upstream"}`))},
			{StatusCode: http.StatusConflict, Body: io.NopCloser(strings.NewReader(`{"error":"task not yet assigned upstream"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"f1","state":"cancel_requested","status":"claimed"}`))},
		},
	}

	broker, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	require.NoError(t, broker.CancelTask("t1"))
	require.Len(t, httpContext.Requests, 3)
}

func TestRunnerCancelCallsBroker(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"up-1","state":"already_terminal","status":"succeeded"}`))},
		},
	}

	state := &contexts.ExecutionStateContext{KVs: map[string]string{"task_id": "broker-42"}}
	err := (&Runner{}).Cancel(core.ExecutionContext{
		HTTP:           httpContext,
		ExecutionState: state,
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "/v1/tasks/broker-42/cancel", httpContext.Requests[0].URL.Path)
}

func TestBrokerListActiveTasks(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"tasks": [
						{"id":"task-1","status":"queued","fleet_id":"fleet-1","created_at":"2026-05-24T12:00:00Z"},
						{"id":"task-2","status":"claimed","fleet_id":"fleet-1","created_at":"2026-05-24T12:01:00Z","runner_id":"runner-a"}
					]
				}`)),
			},
		},
	}

	broker, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	tasks, err := broker.ListActiveTasks()
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, "task-1", tasks[0].ID)
	assert.Equal(t, "queued", tasks[0].Status)
	assert.Equal(t, "task-2", tasks[1].ID)
	assert.Equal(t, "runner-a", tasks[1].RunnerID)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, "/v1/tasks", httpContext.Requests[0].URL.Path)
	assert.Equal(t, "Bearer token-1", httpContext.Requests[0].Header.Get("Authorization"))
}
