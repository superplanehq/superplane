package runner

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	runnermodels "github.com/superplanehq/superplane/pkg/runners/models"
	"github.com/superplanehq/superplane/test/support/contexts"
	"gorm.io/datatypes"
)

// mockStore is a test-only in-memory implementation of fleetStore.
type mockStore struct {
	fleet *runnermodels.RunnerFleet
	tasks []*runnermodels.RunnerTask
}

func (m *mockStore) FindFleet(id uuid.UUID) (*runnermodels.RunnerFleet, error) {
	return m.fleet, nil
}

func (m *mockStore) EnqueueJob(fleetID, executionID uuid.UUID, spec runnermodels.JobSpec) (*runnermodels.RunnerTask, error) {
	id := uuid.New()
	t := &runnermodels.RunnerTask{
		ID:          id,
		FleetID:     fleetID,
		FleetTaskID: id.String(),
		ExecutionID: executionID,
		Status:      runnermodels.TaskStatusQueued,
		Spec:        datatypes.NewJSONType(spec),
	}
	m.tasks = append(m.tasks, t)
	return t, nil
}

func testFleet() *runnermodels.RunnerFleet {
	return &runnermodels.RunnerFleet{
		ID:        uuid.New(),
		Name:      "fleet-1",
		AuthToken: "token-1",
	}
}

func testRunner(fleet *runnermodels.RunnerFleet) *Runner {
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

func TestRunnerExecuteEnqueuesJob(t *testing.T) {
	fleet := testFleet()
	store := &mockStore{fleet: fleet}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&Runner{store: store}).Execute(core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  map[string]any{"fleet_id": fleet.ID.String(), "commands": "echo hello"},
		HTTP:           &contexts.HTTPContext{},
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
	})

	require.NoError(t, err)
	require.Len(t, store.tasks, 1)
	assert.Equal(t, []string{"echo hello"}, store.tasks[0].Spec.Data().Commands)
	assert.Equal(t, runnermodels.TaskStatusQueued, store.tasks[0].Status)
	assert.NotEmpty(t, state.KVs["task_id"])
}

func TestRunnerExecuteSendsEnvironmentInJobSpec(t *testing.T) {
	fleet := testFleet()
	store := &mockStore{fleet: fleet}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&Runner{store: store}).Execute(core.ExecutionContext{
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
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"api/token": []byte("secret'value;$PATH")}},
		ExecutionState: state,
		Requests:       &contexts.RequestContext{},
	})

	require.NoError(t, err)
	require.Len(t, store.tasks, 1)
	assert.Equal(t, []runnermodels.FleetEnvironmentVariable{
		{Name: "COMMIT_AUTHOR", Value: "alice@example.com"},
		{Name: "API_TOKEN", Value: "secret'value;$PATH"},
	}, store.tasks[0].Spec.Data().Environment)
}

func TestRunnerExecuteFailsWhenSecretCannotBeResolved(t *testing.T) {
	fleet := testFleet()

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
		HTTP:           &contexts.HTTPContext{},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{}},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
	})

	require.ErrorContains(t, err, "failed to resolve environment variable API_TOKEN")
}

func secretRef(secret, key string) configuration.SecretKeyRef {
	return configuration.SecretKeyRef{Secret: secret, Key: key}
}

func TestRunnerProcessTaskStatusIncludesResult(t *testing.T) {
	t.Parallel()

	state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	exit := 0
	task := &runnermodels.FleetTask{
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
	task := &runnermodels.FleetTask{
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
	task := &runnermodels.FleetTask{
		Status:   "canceled",
		ExitCode: &exit,
	}
	require.NoError(t, (&Runner{}).processTaskStatus(state, task))
	require.Equal(t, FailedOutputChannel, state.Channel)
}
