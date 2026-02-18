package jsruntime

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func TestParseDefinition_ValidComponent(t *testing.T) {
	source := `
		superplane.component({
			label: "Transform",
			description: "Transforms data",
			icon: "shuffle",
			color: "green",
			configuration: [
				{ name: "field", label: "Field", type: "string", required: true },
			],
			outputChannels: [
				{ name: "default", label: "Default" },
				{ name: "error", label: "Error" },
			],
			setup(ctx) {},
			execute(ctx) {},
		});
	`

	rt := NewRuntime(0)
	def, err := rt.ParseDefinition(source)

	require.NoError(t, err)
	assert.Equal(t, "Transform", def.Label)
	assert.Equal(t, "Transforms data", def.Description)
	assert.Equal(t, "shuffle", def.Icon)
	assert.Equal(t, "green", def.Color)
	assert.True(t, def.HasExecute)
	assert.True(t, def.HasSetup)

	config, err := def.ParseConfiguration()
	require.NoError(t, err)
	require.Len(t, config, 1)
	assert.Equal(t, "field", config[0].Name)
	assert.True(t, config[0].Required)

	channels, err := def.ParseOutputChannels()
	require.NoError(t, err)
	require.Len(t, channels, 2)
	assert.Equal(t, "default", channels[0].Name)
	assert.Equal(t, "error", channels[1].Name)
}

func TestParseDefinition_MinimalComponent(t *testing.T) {
	source := `
		superplane.component({
			label: "Noop",
			description: "Does nothing",
			execute(ctx) {},
		});
	`

	rt := NewRuntime(0)
	def, err := rt.ParseDefinition(source)

	require.NoError(t, err)
	assert.Equal(t, "Noop", def.Label)
	assert.Equal(t, "code", def.Icon)
	assert.Equal(t, "blue", def.Color)
	assert.True(t, def.HasExecute)
	assert.False(t, def.HasSetup)

	channels, err := def.ParseOutputChannels()
	require.NoError(t, err)
	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func TestParseDefinition_NoComponentCall(t *testing.T) {
	source := `var x = 1;`

	rt := NewRuntime(0)
	_, err := rt.ParseDefinition(source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "did not call superplane.component()")
}

func TestParseDefinition_NoExecuteFunction(t *testing.T) {
	source := `
		superplane.component({
			label: "Bad",
			description: "Missing execute",
			setup(ctx) {},
		});
	`

	rt := NewRuntime(0)
	_, err := rt.ParseDefinition(source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must include an execute function")
}

func TestParseDefinition_SyntaxError(t *testing.T) {
	source := `superplane.component({{{`

	rt := NewRuntime(0)
	_, err := rt.ParseDefinition(source)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse component script")
}

func TestExecute_EmitsPayload(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				ctx.emit("default", "test.result", { count: 42 });
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)

	err := rt.Execute(source, ctx)

	require.NoError(t, err)
	require.Len(t, state.emitted, 1)
	assert.Equal(t, "default", state.emitted[0].channel)
	assert.Equal(t, "test.result", state.emitted[0].payloadType)
}

func TestExecute_Pass(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				ctx.pass();
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)

	err := rt.Execute(source, ctx)

	require.NoError(t, err)
	assert.True(t, state.passed)
}

func TestExecute_Fail(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				ctx.fail("error", "something went wrong");
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)

	err := rt.Execute(source, ctx)

	require.NoError(t, err)
	assert.Equal(t, "error", state.failReason)
	assert.Equal(t, "something went wrong", state.failMessage)
}

func TestExecute_ReadsInputAndConfiguration(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				var total = ctx.input.items.length;
				var field = ctx.configuration.filterField;
				ctx.emit("default", "test.result", { total: total, field: field });
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)
	ctx.Data = map[string]any{
		"items": []any{"a", "b", "c"},
	}
	ctx.Configuration = map[string]any{
		"filterField": "status",
	}

	err := rt.Execute(source, ctx)

	require.NoError(t, err)
	require.Len(t, state.emitted, 1)

	data := state.emitted[0].data.(map[string]any)
	assert.Equal(t, int64(3), data["total"])
	assert.Equal(t, "status", data["field"])
}

func TestExecute_Timeout(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				while (true) {}
			},
		});
	`

	rt := NewRuntime(100 * time.Millisecond)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)

	err := rt.Execute(source, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestExecute_ThrowsError(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				throw new Error("custom error from JS");
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	ctx := newTestExecutionContext(state)

	err := rt.Execute(source, ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom error from JS")
}

func TestExecute_MetadataGetSet(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {
				ctx.metadata.set({ step: 1 });
				var m = ctx.metadata.get();
				ctx.emit("default", "test.result", { step: m.step });
			},
		});
	`

	rt := NewRuntime(0)
	state := &mockExecutionState{}
	meta := &mockMetadata{}
	ctx := newTestExecutionContext(state)
	ctx.Metadata = meta

	err := rt.Execute(source, ctx)

	require.NoError(t, err)
	require.Len(t, state.emitted, 1)
}

func TestSetup_ValidatesConfiguration(t *testing.T) {
	source := `
		superplane.component({
			setup(ctx) {
				if (!ctx.configuration.url) {
					throw new Error("url is required");
				}
			},
			execute(ctx) {},
		});
	`

	rt := NewRuntime(0)

	err := rt.Setup(source, core.SetupContext{
		Configuration: map[string]any{},
		Logger:        log.NewEntry(log.StandardLogger()),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestSetup_PassesWithValidConfig(t *testing.T) {
	source := `
		superplane.component({
			setup(ctx) {
				if (!ctx.configuration.url) {
					throw new Error("url is required");
				}
			},
			execute(ctx) {},
		});
	`

	rt := NewRuntime(0)

	err := rt.Setup(source, core.SetupContext{
		Configuration: map[string]any{"url": "https://example.com"},
		Logger:        log.NewEntry(log.StandardLogger()),
	})

	require.NoError(t, err)
}

func TestSetup_NoSetupHandler(t *testing.T) {
	source := `
		superplane.component({
			execute(ctx) {},
		});
	`

	rt := NewRuntime(0)

	err := rt.Setup(source, core.SetupContext{
		Configuration: map[string]any{},
		Logger:        log.NewEntry(log.StandardLogger()),
	})

	require.NoError(t, err)
}

// --- Test Helpers ---

func newTestExecutionContext(state *mockExecutionState) core.ExecutionContext {
	return core.ExecutionContext{
		ID:             uuid.New(),
		WorkflowID:     uuid.New().String(),
		NodeID:         "node-1",
		Data:           map[string]any{},
		Configuration:  map[string]any{},
		ExecutionState: state,
		Metadata:       &mockMetadata{},
		Logger:         log.NewEntry(log.StandardLogger()),
	}
}

type emittedPayload struct {
	channel     string
	payloadType string
	data        any
}

type mockExecutionState struct {
	emitted     []emittedPayload
	passed      bool
	failReason  string
	failMessage string
}

func (m *mockExecutionState) IsFinished() bool { return false }

func (m *mockExecutionState) SetKV(_, _ string) error { return nil }

func (m *mockExecutionState) Emit(channel, payloadType string, payloads []any) error {
	for _, p := range payloads {
		m.emitted = append(m.emitted, emittedPayload{
			channel:     channel,
			payloadType: payloadType,
			data:        p,
		})
	}

	return nil
}

func (m *mockExecutionState) Pass() error {
	m.passed = true
	return nil
}

func (m *mockExecutionState) Fail(reason, message string) error {
	m.failReason = reason
	m.failMessage = message
	return nil
}

type mockMetadata struct {
	value any
}

func (m *mockMetadata) Get() any {
	return m.value
}

func (m *mockMetadata) Set(v any) error {
	m.value = v
	return nil
}

type mockSecrets struct{}

func (m *mockSecrets) GetKey(secretName, keyName string) ([]byte, error) {
	if secretName == "test" && keyName == "api-key" {
		return []byte("secret-value"), nil
	}

	return nil, fmt.Errorf("secret not found")
}
