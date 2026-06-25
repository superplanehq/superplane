package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateFirewall__Setup(t *testing.T) {
	component := &UpdateFirewall{}

	t.Run("missing firewall returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "firewall rule is required")
	})

	t.Run("invalid enabled state returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"firewall": "allow-http", "enabledState": "MAYBE"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid enabled state")
	})

	t.Run("valid config stores firewall name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"firewall": "allow-http", "enabledState": "DISABLED"},
			Metadata:      meta,
		})
		require.NoError(t, err)
		var stored FirewallNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "allow-http", stored.FirewallName)
	})
}

func Test__UpdateFirewall__Execute(t *testing.T) {
	component := &UpdateFirewall{}

	t.Run("updates priority and disables -> emits updated event", func(t *testing.T) {
		var patchPath string
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchPath = path
				patchBody = body.(map[string]any)
				return opDone("op-update"), nil
			},
			getFunc: firewallExecGet("op-update", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":     "allow-http",
				"enabledState": "DISABLED",
				"priority":     900,
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.firewallRule.updated", state.Type)
		assert.True(t, strings.HasSuffix(patchPath, "/global/firewalls/allow-http"))
		assert.Equal(t, true, patchBody["disabled"])
		assert.Equal(t, 900, patchBody["priority"])

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "allow-http", data["name"])
		assert.Contains(t, data["link"], "console.cloud.google.com/networking/firewalls/details/allow-http")
	})

	t.Run("rules patch the array matching the rule's action (deny)", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("deny-ssh", "INGRESS", "deny")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall": "deny-ssh",
				"rules":    []map[string]any{{"protocol": "tcp", "ports": "22, 2222"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Contains(t, patchBody, "denied")
		assert.NotContains(t, patchBody, "allowed")
	})

	t.Run("ranges route to destinationRanges for egress rules", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("egress-rule", "EGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall": "egress-rule",
				"ranges":   []string{"10.0.0.0/8"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"10.0.0.0/8"}, patchBody["destinationRanges"])
		assert.NotContains(t, patchBody, "sourceRanges")
	})

	t.Run("ranges route to sourceRanges for ingress rules", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall": "allow-http",
				"ranges":   []string{"192.168.0.0/16"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"192.168.0.0/16"}, patchBody["sourceRanges"])
		assert.NotContains(t, patchBody, "destinationRanges")
	})

	t.Run("empty target tags clears them (broadens to all VMs)", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":   "allow-http",
				"targetTags": []string{},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Contains(t, patchBody, "targetTags")
		assert.Empty(t, patchBody["targetTags"])
	})

	t.Run("source service accounts and logging are patched on an ingress rule", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			// A rule without tags, so adding service accounts doesn't mix the two.
			getFunc: firewallExecGet("op", firewallGetJSONNoTags("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"sourceServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
				"logging":               "ENABLED",
				"logMetadata":           "INCLUDE_ALL_METADATA",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"sa@my-project.iam.gserviceaccount.com"}, patchBody["sourceServiceAccounts"])
		logCfg := patchBody["logConfig"].(map[string]any)
		assert.Equal(t, true, logCfg["enable"])
		assert.Equal(t, "INCLUDE_ALL_METADATA", logCfg["metadata"])
	})

	t.Run("adding service accounts to a tag-based rule fails (would mix tags + SAs)", func(t *testing.T) {
		var patched bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patched = true
				return opDone("op"), nil
			},
			// Current rule already has targetTags ["web"].
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"targetServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "cannot combine network tags and service accounts")
	})

	t.Run("switching tags to service accounts succeeds when the tags are cleared", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"targetServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
				"targetTags":            []string{}, // explicitly clear the existing tags
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"sa@my-project.iam.gserviceaccount.com"}, patchBody["targetServiceAccounts"])
		require.Contains(t, patchBody, "targetTags")
		assert.Empty(t, patchBody["targetTags"])
	})

	t.Run("source tags rejected on an egress rule", func(t *testing.T) {
		var patched bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patched = true
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("deny-egress", "EGRESS", "deny")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":   "deny-egress",
				"sourceTags": []string{"web"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "source tags apply only to INGRESS")
	})

	t.Run("nothing to update -> fails without patching", func(t *testing.T) {
		var patched bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patched = true
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"firewall": "allow-http", "enabledState": "NO_CHANGE"},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "nothing to update")
	})

	t.Run("cross-project selfLink -> fails before patch", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				called = true
				return firewallGetJSON("allow-http", "INGRESS", "allow"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":     "https://www.googleapis.com/compute/v1/projects/other-project/global/firewalls/allow-http",
				"enabledState": "DISABLED",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
