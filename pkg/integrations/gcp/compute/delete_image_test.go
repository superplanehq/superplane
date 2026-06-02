package compute

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteImage__Setup(t *testing.T) {
	component := &DeleteImage{}

	t.Run("missing image returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "image is required")
	})

	t.Run("stores parsed image name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"image": "https://www.googleapis.com/compute/v1/projects/my-project/global/images/my-image",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored ImageNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "my-image", stored.ImageName)
	})

	t.Run("expression image stored verbatim", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"image": "{{ $.nodes.create.outputs.default[0].data.selfLink }}",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored ImageNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Contains(t, stored.ImageName, "{{")
	})
}

func Test__DeleteImage__Execute(t *testing.T) {
	component := &DeleteImage{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		var deletePath string
		mc := &mockImageClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				deletePath = path
				return opDone("op-del"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				return opDone("op-del"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"image": "my-image"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.image.deleted", state.Type)
		assert.True(t, strings.HasSuffix(deletePath, "/global/images/my-image"))
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "my-image", data["imageName"])
	})

	t.Run("not found (404) -> fails execution", func(t *testing.T) {
		mc := &mockImageClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "Image not found"}
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"image": "my-image"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to delete image")
	})

	t.Run("cross-project selfLink -> fails before delete", func(t *testing.T) {
		var called bool
		mc := &mockImageClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"image": "https://www.googleapis.com/compute/v1/projects/other-project/global/images/my-image",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
