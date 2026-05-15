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

func TestCommandForExecution(t *testing.T) {
	t.Parallel()

	got := commandForExecution("echo a\n echo 'b' \n")
	want := []string{
		"printf '$ %s\\n' 'echo a'\necho a",
		"printf '$ %s\\n' 'echo '\\''b'\\'''\necho 'b'",
	}
	require.Equal(t, want, got)
	require.Nil(t, commandForExecution("\n \n"))
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
	t.Setenv("TASK_BROKER_FLEET_ID", "fleet-1")
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

	var req brokerCreateTaskRequest
	require.NoError(t, json.Unmarshal(body, &req))

	assert.Equal(t, "fleet-1", req.FleetID)
	assert.Equal(t, []string{"printf '$ %s\\n' 'echo hello'\necho hello"}, req.Commands)
	assert.Equal(t, "host", req.ExecutionMode)
	assert.Equal(t, []BrokerEnvironmentVariable{
		{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
		{Name: "API_TOKEN", Value: "secret'value;$PATH"},
	}, req.Environment)
	assert.Equal(t, "task-123", state.KVs["task_id"])
	assert.Equal(t, hookActionPoll, requests.Action)
}

func TestRunnerExecuteOmitsEmptyEnvironment(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_FLEET_ID", "fleet-1")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":"task-123"}`))},
		},
	}

	err := (&Runner{}).Execute(core.ExecutionContext{
		Configuration:  map[string]any{"commands": "echo hello"},
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
	t.Setenv("TASK_BROKER_FLEET_ID", "fleet-1")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{}

	err := (&Runner{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
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
