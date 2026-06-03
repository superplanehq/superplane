package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	compute "google.golang.org/api/compute/v1"
)

func Test__UpdateImage__Setup(t *testing.T) {
	component := &UpdateImage{}

	t.Run("missing image returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "image is required")
	})

	t.Run("invalid deprecation state returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"image": "my-image", "deprecationState": "BOGUS"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid deprecation state")
	})

	t.Run("valid config stores image name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"image": "global/images/my-image", "deprecationState": "DEPRECATED"},
			Metadata:      meta,
		})
		require.NoError(t, err)
	})
}

func Test__UpdateImage__Execute(t *testing.T) {
	component := &UpdateImage{}

	t.Run("nothing to update -> fails", func(t *testing.T) {
		mc := &mockImageClient{projectID: "my-project"}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"image": "my-image"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "nothing to update")
	})

	t.Run("deprecate only -> calls deprecate and emits updated", func(t *testing.T) {
		var deprecatePath string
		mc := &mockImageClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-dep"), nil
				}
				return imageGetJSON("my-image", "READY", "my-app", nil, "fp-1", "DEPRECATED"), nil
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				deprecatePath = path
				return opDone("op-dep"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"image":            "my-image",
				"deprecationState": "DEPRECATED",
				"replacement":      "my-app-v2",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.image.updated", state.Type)
		assert.True(t, strings.HasSuffix(deprecatePath, "/global/images/my-image/deprecate"))
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "DEPRECATED", data["deprecationState"])
	})

	t.Run("labels merge with existing -> sends combined set and fingerprint", func(t *testing.T) {
		var setLabelsPath string
		var setLabelsReq *compute.GlobalSetLabelsRequest
		mc := &mockImageClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-lbl"), nil
				}
				return imageGetJSON("my-image", "READY", "", map[string]string{"env": "staging", "team": "core"}, "fp-1", ""), nil
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				setLabelsPath = path
				setLabelsReq, _ = body.(*compute.GlobalSetLabelsRequest)
				return opDone("op-lbl"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"image": "my-image",
				"labels": []any{
					map[string]any{"key": "env", "value": "prod"},
					map[string]any{"key": "owner", "value": "platform"},
				},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.True(t, strings.HasSuffix(setLabelsPath, "/global/images/my-image/setLabels"))
		require.NotNil(t, setLabelsReq)
		assert.Equal(t, "fp-1", setLabelsReq.LabelFingerprint)
		// env is overwritten, owner is added, team is preserved.
		assert.Equal(t, map[string]string{"env": "prod", "team": "core", "owner": "platform"}, setLabelsReq.Labels)
	})

	t.Run("active state drops a stale replacement", func(t *testing.T) {
		var deprecateReq *compute.DeprecationStatus
		mc := &mockImageClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if isOperationPath(path) {
					return opDone("op-act"), nil
				}
				return imageGetJSON("my-image", "READY", "", nil, "fp-1", "ACTIVE"), nil
			},
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				deprecateReq, _ = body.(*compute.DeprecationStatus)
				return opDone("op-act"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"image":            "my-image",
				"deprecationState": "ACTIVE",
				// Stale value left over from a previous Deprecated selection.
				"replacement": "my-app-v2",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.NotNil(t, deprecateReq)
		assert.Equal(t, "ACTIVE", deprecateReq.State)
		assert.Empty(t, deprecateReq.Replacement)
	})

	t.Run("cross-project selfLink -> fails before mutating", func(t *testing.T) {
		var called bool
		mc := &mockImageClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return imageGetJSON("my-image", "READY", "", nil, "fp", ""), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"image":            "https://www.googleapis.com/compute/v1/projects/other-project/global/images/my-image",
				"deprecationState": "OBSOLETE",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
