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

func Test__DeleteFirewall__Setup(t *testing.T) {
	component := &DeleteFirewall{}

	t.Run("missing firewall returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "firewall rule is required")
	})

	t.Run("stores parsed firewall name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"firewall": "https://www.googleapis.com/compute/v1/projects/my-project/global/firewalls/allow-http",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored FirewallNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "allow-http", stored.FirewallName)
	})

	t.Run("expression firewall stored verbatim", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"firewall": "{{ $.nodes.create.outputs.default[0].data.selfLink }}",
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored FirewallNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Contains(t, stored.FirewallName, "{{")
	})
}

func Test__DeleteFirewall__Execute(t *testing.T) {
	component := &DeleteFirewall{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		var deletePath string
		mc := &mockFirewallClient{
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
			Configuration:  map[string]any{"firewall": "allow-http"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.firewallRule.deleted", state.Type)
		assert.True(t, strings.HasSuffix(deletePath, "/global/firewalls/allow-http"))
		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "allow-http", data["name"])
	})

	t.Run("not found (404) -> fails execution", func(t *testing.T) {
		mc := &mockFirewallClient{
			projectID: "my-project",
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "firewall not found"}
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"firewall": "allow-http"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "failed to delete firewall rule")
	})

	t.Run("cross-project selfLink -> fails before delete", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
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
				"firewall": "https://www.googleapis.com/compute/v1/projects/other-project/global/firewalls/allow-http",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
