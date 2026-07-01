package compute

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteStaticIP__Setup(t *testing.T) {
	component := &DeleteStaticIP{}

	t.Run("missing address returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-central1"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "address is required")
	})

	t.Run("plain name is rejected", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"address": "web-ip"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.Error(t, err)
	})

	t.Run("expression is accepted without parsing", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"address": "{{ $.nodes.create.outputs.default[0].data.selfLink }}"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid selfLink is accepted", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"address": "regions/us-central1/addresses/web-ip"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__DeleteStaticIP__Execute(t *testing.T) {
	component := &DeleteStaticIP{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		var deletePath string
		mc := &mockStaticIPClient{
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
			Configuration:  map[string]any{"address": "regions/us-central1/addresses/web-ip"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.staticIP.deleted", state.Type)
		assert.True(t, strings.HasSuffix(deletePath, "/regions/us-central1/addresses/web-ip"))
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "web-ip", data["name"])
		assert.Equal(t, "us-central1", data["region"])
	})

	t.Run("cross-project selfLink -> fails before any API call", func(t *testing.T) {
		var called bool
		mc := &mockStaticIPClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return opDone("op-x"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"address": "https://www.googleapis.com/compute/v1/projects/other-project/regions/us-central1/addresses/web-ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})

	t.Run("not found (404) -> fails execution", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"address": "regions/us-central1/addresses/web-ip"},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to delete static IP")
	})
}
