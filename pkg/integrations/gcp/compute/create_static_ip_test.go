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

func Test__CreateStaticIP__Setup(t *testing.T) {
	component := &CreateStaticIP{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"region": "us-central1"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing region returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "web-ip"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("invalid network tier returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "web-ip", "region": "us-central1", "networkTier": "GOLD"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid networkTier")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "web-ip", "region": "us-central1"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__CreateStaticIP__Execute(t *testing.T) {
	component := &CreateStaticIP{}

	t.Run("reserves IP -> emits created event", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				assert.True(t, strings.HasSuffix(path, "/regions/us-central1/addresses"))
				postBody, _ = body.(map[string]any)
				return opDone("op-create"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op-create"), nil
				}
				assert.True(t, strings.HasSuffix(path, "/regions/us-central1/addresses/web-prod-ip"))
				return addressJSON("web-prod-ip", "34.1.2.3", "us-central1", "RESERVED", "EXTERNAL", "PREMIUM"), nil
			},
		}

		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":        "web-prod-ip",
				"region":      "us-central1",
				"networkTier": "PREMIUM",
				"description": "prod ip",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.staticIP.created", state.Type)
		require.Len(t, state.Payloads, 1)
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "web-prod-ip", data["name"])
		assert.Equal(t, "34.1.2.3", data["address"])
		assert.Equal(t, "us-central1", data["region"])
		assert.Equal(t, "EXTERNAL", postBody["addressType"])
		assert.Equal(t, "PREMIUM", postBody["networkTier"])
	})

	t.Run("normalizes region selfLink", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				assert.True(t, strings.HasSuffix(path, "/regions/us-central1/addresses"))
				return opDone("op-create"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op-create"), nil
				}
				return addressJSON("web-ip", "34.1.2.3", "us-central1", "RESERVED", "EXTERNAL", "PREMIUM"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "web-ip",
				"region": "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
	})

	t.Run("API error -> fails execution", func(t *testing.T) {
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusConflict, Message: "already exists"}
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "web-ip", "region": "us-central1"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to reserve static IP")
	})
}
