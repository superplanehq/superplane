package ansible

import (
	"context"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
)

// --- test doubles -----------------------------------------------------------

type testMetadataContext struct {
	value any
}

func (m *testMetadataContext) Get() any        { return m.value }
func (m *testMetadataContext) Set(v any) error { m.value = v; return nil }

type emittedPayload struct {
	channel     string
	payloadType string
	payloads    []any
}

type testExecutionState struct {
	finished bool
	emitted  []emittedPayload
}

func (s *testExecutionState) IsFinished() bool                  { return s.finished }
func (s *testExecutionState) SetKV(key, value string) error     { return nil }
func (s *testExecutionState) GetKV(key string) (string, error)  { return "", nil }
func (s *testExecutionState) Pass() error                       { return nil }
func (s *testExecutionState) Fail(reason, message string) error { return nil }

func (s *testExecutionState) Emit(channel, payloadType string, payloads []any) error {
	s.emitted = append(s.emitted, emittedPayload{channel, payloadType, payloads})
	return nil
}

type stubRunner struct {
	result *RunResult
	err    error
	called bool
}

func (r *stubRunner) Run(ctx context.Context, spec Spec, logger *log.Entry) (*RunResult, error) {
	r.called = true
	return r.result, r.err
}

func strPtr(s string) *string { return &s }

// --- Setup / validation -----------------------------------------------------

func TestSetupValidation(t *testing.T) {
	a := &Ansible{}

	cases := []struct {
		name    string
		config  map[string]any
		wantErr string
	}{
		{
			name:    "missing inventory",
			config:  map[string]any{"mode": ModePlaybook, "playbook": "- hosts: all", "inventory": "", "timeout": 60},
			wantErr: "inventory is required",
		},
		{
			name:    "invalid mode",
			config:  map[string]any{"mode": "nope", "inventory": "localhost", "timeout": 60},
			wantErr: "invalid mode",
		},
		{
			name:    "playbook mode without playbook",
			config:  map[string]any{"mode": ModePlaybook, "inventory": "localhost", "timeout": 60},
			wantErr: "playbook is required",
		},
		{
			name:    "adhoc mode without host pattern",
			config:  map[string]any{"mode": ModeAdhoc, "inventory": "localhost", "timeout": 60},
			wantErr: "host pattern is required",
		},
		{
			name:    "host pattern starting with dash",
			config:  map[string]any{"mode": ModeAdhoc, "hostPattern": "--version", "inventory": "localhost", "timeout": 60},
			wantErr: "must not start with '-'",
		},
		{
			name:    "invalid module name",
			config:  map[string]any{"mode": ModeAdhoc, "hostPattern": "all", "module": "bad module!", "inventory": "localhost", "timeout": 60},
			wantErr: "invalid module name",
		},
		{
			name:    "invalid extra var name",
			config:  map[string]any{"mode": ModeAdhoc, "hostPattern": "all", "inventory": "localhost", "timeout": 60, "extraVars": []map[string]any{{"name": "BAD-NAME", "value": "x"}}},
			wantErr: "invalid extra variable name",
		},
		{
			name:    "timeout too small",
			config:  map[string]any{"mode": ModeAdhoc, "hostPattern": "all", "inventory": "localhost", "timeout": -1},
			wantErr: "timeout must be at least",
		},
		{
			name:    "verbosity out of range",
			config:  map[string]any{"mode": ModeAdhoc, "hostPattern": "all", "inventory": "localhost", "timeout": 60, "verbosity": 9},
			wantErr: "verbosity must be between",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := a.Setup(core.SetupContext{Configuration: tc.config})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	t.Run("valid playbook config", func(t *testing.T) {
		err := a.Setup(core.SetupContext{Configuration: map[string]any{
			"mode": ModePlaybook, "playbook": "- hosts: all\n  tasks:\n    - ping:", "inventory": defaultInventory, "timeout": 300,
		}})
		require.NoError(t, err)
	})

	t.Run("valid adhoc config", func(t *testing.T) {
		err := a.Setup(core.SetupContext{Configuration: map[string]any{
			"mode": ModeAdhoc, "hostPattern": "all", "module": "ansible.builtin.ping", "inventory": defaultInventory, "timeout": 300,
		}})
		require.NoError(t, err)
	})
}

func TestDecodeSpecDefaults(t *testing.T) {
	spec, err := decodeSpec(map[string]any{"inventory": "localhost", "playbook": "x"})
	require.NoError(t, err)
	assert.Equal(t, ModePlaybook, spec.Mode)
	assert.Equal(t, defaultTimeout, spec.Timeout)
}

// --- argv construction ------------------------------------------------------

func TestBuildPlaybookArgs(t *testing.T) {
	spec := Spec{
		Become:    true,
		Verbosity: 2,
		Limit:     strPtr("web"),
		ExtraVars: []ExtraVar{{Name: "env", Value: "prod"}, {Name: "skip", Value: ""}},
	}

	args := buildPlaybookArgs("/tmp/play.yml", "/tmp/inv", spec)

	assert.Equal(t, []string{
		"-i", "/tmp/inv",
		"--limit", "web",
		"--become",
		"-vv",
		"-e", "env=prod",
		"-e", "skip=",
		"/tmp/play.yml",
	}, args)
}

func TestBuildAdhocArgs(t *testing.T) {
	t.Run("defaults module to shell", func(t *testing.T) {
		args := buildAdhocArgs("/tmp/inv", Spec{HostPattern: strPtr("all"), ModuleArgs: strPtr("uptime")})
		assert.Equal(t, []string{"all", "-i", "/tmp/inv", "-m", "shell", "-a", "uptime"}, args)
	})

	t.Run("explicit module without args", func(t *testing.T) {
		args := buildAdhocArgs("/tmp/inv", Spec{HostPattern: strPtr("web"), Module: strPtr("ping")})
		assert.Equal(t, []string{"web", "-i", "/tmp/inv", "-m", "ping"}, args)
	})
}

func TestVerbosityFlag(t *testing.T) {
	assert.Equal(t, "", verbosityFlag(0))
	assert.Equal(t, "", verbosityFlag(-3))
	assert.Equal(t, "-v", verbosityFlag(1))
	assert.Equal(t, "-vvvv", verbosityFlag(4))
	assert.Equal(t, "-vvvv", verbosityFlag(10))
}

func TestExtraVarArgs(t *testing.T) {
	// A value containing shell metacharacters is passed verbatim as a single
	// argv element (no shell), so it cannot inject extra commands.
	args := extraVarArgs([]ExtraVar{{Name: "msg", Value: "a; rm -rf / $(whoami)"}})
	assert.Equal(t, []string{"-e", "msg=a; rm -rf / $(whoami)"}, args)
}

// --- recap parsing ----------------------------------------------------------

func TestParseRecap(t *testing.T) {
	t.Run("valid json stats", func(t *testing.T) {
		out := []byte(`{"stats":{"localhost":{"ok":3,"changed":1,"unreachable":0,"failures":0,"skipped":2,"rescued":0,"ignored":0}}}`)
		recap := parseRecap(out)
		require.NotNil(t, recap)
		assert.Equal(t, 3, recap["localhost"].Ok)
		assert.Equal(t, 1, recap["localhost"].Changed)
		assert.Equal(t, 2, recap["localhost"].Skipped)
	})

	t.Run("non-json returns nil", func(t *testing.T) {
		assert.Nil(t, parseRecap([]byte("PLAY RECAP *** localhost : ok=3")))
	})
}

// --- Execute channel routing ------------------------------------------------

func executeWith(t *testing.T, mode string, runner *stubRunner) *testExecutionState {
	t.Helper()
	state := &testExecutionState{}
	a := &Ansible{runner: runner}

	config := map[string]any{"mode": mode, "inventory": defaultInventory, "timeout": 60}
	if mode == ModeAdhoc {
		config["hostPattern"] = "all"
	} else {
		config["playbook"] = "- hosts: all\n  tasks:\n    - ping:"
	}

	err := a.Execute(core.ExecutionContext{
		Configuration:  config,
		Metadata:       &testMetadataContext{},
		ExecutionState: state,
		Logger:         log.NewEntry(log.New()),
	})
	require.NoError(t, err)
	require.True(t, runner.called)
	return state
}

func TestExecuteRouting(t *testing.T) {
	t.Run("exit 0 routes to success", func(t *testing.T) {
		state := executeWith(t, ModePlaybook, &stubRunner{result: &RunResult{ExitCode: 0}})
		require.Len(t, state.emitted, 1)
		assert.Equal(t, channelSuccess, state.emitted[0].channel)
		assert.Equal(t, payloadTypePlaybook, state.emitted[0].payloadType)
	})

	t.Run("non-zero exit routes to failed", func(t *testing.T) {
		state := executeWith(t, ModePlaybook, &stubRunner{result: &RunResult{ExitCode: 2}})
		require.Len(t, state.emitted, 1)
		assert.Equal(t, channelFailed, state.emitted[0].channel)
	})

	t.Run("adhoc uses adhoc payload type", func(t *testing.T) {
		state := executeWith(t, ModeAdhoc, &stubRunner{result: &RunResult{ExitCode: 0}})
		require.Len(t, state.emitted, 1)
		assert.Equal(t, payloadTypeAdhoc, state.emitted[0].payloadType)
	})
}

func TestExecuteRunnerErrorSurfacesAsError(t *testing.T) {
	state := &testExecutionState{}
	a := &Ansible{runner: &stubRunner{err: assertErr("ansible: command not found")}}

	err := a.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"mode": ModePlaybook, "inventory": defaultInventory, "playbook": "- hosts: all", "timeout": 60},
		Metadata:       &testMetadataContext{},
		ExecutionState: state,
		Logger:         log.NewEntry(log.New()),
	})

	require.Error(t, err)
	assert.Empty(t, state.emitted, "no channel should be emitted when the component errors")
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
