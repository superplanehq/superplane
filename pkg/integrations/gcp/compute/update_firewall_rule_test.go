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

	t.Run("invalid target type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"firewall": "allow-http", "targetType": "bogus"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid target type")
	})

	t.Run("setup rejects cross-side tag/SA mix", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"targetType":            "tags",
				"targetTags":            []string{"web"},
				"sourceFilterType":      "serviceAccounts",
				"sourceServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "cannot combine network tags and service accounts")
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
				"firewall":          "deny-ssh",
				"protocolsAndPorts": "specified",
				"rules":             []map[string]any{{"protocol": "tcp", "ports": "22, 2222"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Contains(t, patchBody, "denied")
		assert.NotContains(t, patchBody, "allowed")
	})

	t.Run("protocolsAndPorts=all patches the match-everything rule", func(t *testing.T) {
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
				"firewall":          "allow-http",
				"protocolsAndPorts": "all",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		allowed := patchBody["allowed"].([]map[string]any)
		require.Len(t, allowed, 1)
		assert.Equal(t, "all", allowed[0]["IPProtocol"])
		assert.NotContains(t, allowed[0], "ports")
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

	t.Run("targetType=all clears both target fields (broadens to all VMs)", func(t *testing.T) {
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
				"targetType": "all",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Contains(t, patchBody, "targetTags")
		assert.Empty(t, patchBody["targetTags"])
		require.Contains(t, patchBody, "targetServiceAccounts")
		assert.Empty(t, patchBody["targetServiceAccounts"])
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
				"sourceFilterType":      "serviceAccounts",
				"sourceServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
				"logging":               "ENABLED",
				"logMetadata":           "INCLUDE_ALL_METADATA",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"sa@my-project.iam.gserviceaccount.com"}, patchBody["sourceServiceAccounts"])
		require.Contains(t, patchBody, "sourceTags")
		assert.Empty(t, patchBody["sourceTags"])
		logCfg := patchBody["logConfig"].(map[string]any)
		assert.Equal(t, true, logCfg["enable"])
		assert.Equal(t, "INCLUDE_ALL_METADATA", logCfg["metadata"])
	})

	t.Run("cross-side tag/SA mix fails (target tags + source SAs)", func(t *testing.T) {
		var patched bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patched = true
				return opDone("op"), nil
			},
			// Current rule already has targetTags ["web"]; targetType is NO_CHANGE so
			// those tags survive and conflict with the new source service accounts.
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"targetType":            "NO_CHANGE",
				"sourceFilterType":      "serviceAccounts",
				"sourceServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "cannot combine network tags and service accounts")
	})

	t.Run("switching targets to service accounts auto-clears the existing tags", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			// Current rule has targetTags ["web"]; the dropdown clears them automatically.
			getFunc: firewallExecGet("op", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":              "allow-http",
				"targetType":            "serviceAccounts",
				"targetServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
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
				"firewall":         "deny-egress",
				"sourceFilterType": "tags",
				"sourceTags":       []string{"web"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "source filters apply only to INGRESS")
	})

	t.Run("target tags selected but empty fails", func(t *testing.T) {
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
			Configuration: map[string]any{
				"firewall":   "allow-http",
				"targetType": "tags",
				"targetTags": []string{},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "select at least one target tag")
	})

	t.Run("IP ranges only clears source tags and SAs on ingress", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSONNoTags("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":         "allow-http",
				"sourceFilterType": "ranges",
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		require.Contains(t, patchBody, "sourceTags")
		assert.Empty(t, patchBody["sourceTags"])
		require.Contains(t, patchBody, "sourceServiceAccounts")
		assert.Empty(t, patchBody["sourceServiceAccounts"])
	})

	t.Run("source filter serviceAccounts on egress rejected", func(t *testing.T) {
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
				"firewall":              "deny-egress",
				"sourceFilterType":      "serviceAccounts",
				"sourceServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, patched)
		assert.Contains(t, state.FailureMessage, "source filters apply only to INGRESS")
	})

	t.Run("source tags preserve ranges (ranges+tags combo)", func(t *testing.T) {
		var patchBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			patchFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				patchBody = body.(map[string]any)
				return opDone("op"), nil
			},
			// No current tags so the new source tags don't mix with anything.
			getFunc: firewallExecGet("op", firewallGetJSONNoTags("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"firewall":         "allow-http",
				"ranges":           []string{"10.0.0.0/8"},
				"sourceFilterType": "tags",
				"sourceTags":       []string{"web"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"10.0.0.0/8"}, patchBody["sourceRanges"])
		assert.Equal(t, []string{"web"}, patchBody["sourceTags"])
		require.Contains(t, patchBody, "sourceServiceAccounts")
		assert.Empty(t, patchBody["sourceServiceAccounts"])
	})

	t.Run("priority-only update touches no targeting", func(t *testing.T) {
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
				"priority": 500,
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, 500, patchBody["priority"])
		assert.NotContains(t, patchBody, "targetTags")
		assert.NotContains(t, patchBody, "targetServiceAccounts")
		assert.NotContains(t, patchBody, "sourceTags")
		assert.NotContains(t, patchBody, "sourceServiceAccounts")
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
