package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// mockFirewallClient is a configurable Client mock for the firewall rule
// components (create/update/delete). Each test wires only the funcs it needs.
type mockFirewallClient struct {
	projectID  string
	getFunc    func(ctx context.Context, path string) ([]byte, error)
	postFunc   func(ctx context.Context, path string, body any) ([]byte, error)
	patchFunc  func(ctx context.Context, path string, body any) ([]byte, error)
	deleteFunc func(ctx context.Context, path string) ([]byte, error)
	getURLFunc func(ctx context.Context, fullURL string) ([]byte, error)
}

func (m *mockFirewallClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("unexpected Get(%s)", path)
}

func (m *mockFirewallClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, path, body)
	}
	return nil, fmt.Errorf("unexpected Post(%s)", path)
}

func (m *mockFirewallClient) Patch(ctx context.Context, path string, body any) ([]byte, error) {
	if m.patchFunc != nil {
		return m.patchFunc(ctx, path, body)
	}
	return nil, fmt.Errorf("unexpected Patch(%s)", path)
}

func (m *mockFirewallClient) Delete(ctx context.Context, path string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, path)
	}
	return nil, fmt.Errorf("unexpected Delete(%s)", path)
}

func (m *mockFirewallClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.getURLFunc != nil {
		return m.getURLFunc(ctx, fullURL)
	}
	return nil, fmt.Errorf("unexpected GetURL(%s)", fullURL)
}

func (m *mockFirewallClient) ProjectID() string {
	return m.projectID
}

// firewallGetJSON builds a firewalls.get response body matching firewallGetResp.
// action is "allow" or "deny"; for any other value neither array is set.
func firewallGetJSON(name, direction, action string) []byte {
	body := map[string]any{
		"name":              name,
		"selfLink":          "https://www.googleapis.com/compute/v1/projects/my-project/global/firewalls/" + name,
		"network":           "https://www.googleapis.com/compute/v1/projects/my-project/global/networks/default",
		"direction":         direction,
		"priority":          1000,
		"disabled":          false,
		"creationTimestamp": "2026-06-23T12:00:00.000-07:00",
		"targetTags":        []string{"web"},
	}
	rule := []map[string]any{{"IPProtocol": "tcp", "ports": []string{"80"}}}
	switch action {
	case "allow":
		body["allowed"] = rule
		body["sourceRanges"] = []string{"0.0.0.0/0"}
	case "deny":
		body["denied"] = rule
		body["sourceRanges"] = []string{"0.0.0.0/0"}
	}
	b, _ := json.Marshal(body)
	return b
}

// firewallExecGet returns a getFunc that answers operation polls with a DONE
// operation and firewall reads with the given firewall body.
func firewallExecGet(opName string, fwBody []byte) func(ctx context.Context, path string) ([]byte, error) {
	return func(ctx context.Context, path string) ([]byte, error) {
		if isOperationPath(path) {
			return opDone(opName), nil
		}
		return fwBody, nil
	}
}

func Test__CreateFirewall__Setup(t *testing.T) {
	component := &CreateFirewall{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"network": "default"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing network returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "allow-http"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "network is required")
	})

	t.Run("invalid direction returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"name": "allow-http", "network": "default", "direction": "SIDEWAYS"},
			Metadata:      &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "invalid direction")
	})

	t.Run("empty protocols returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "allow-http",
				"network": "default",
				"rules":   []map[string]any{{"protocol": "", "ports": ""}},
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one protocol")
	})

	t.Run("valid config stores firewall name", func(t *testing.T) {
		meta := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "allow-http",
				"network": "default",
				"rules":   []map[string]any{{"protocol": "tcp", "ports": "80,443"}},
			},
			Metadata: meta,
		})
		require.NoError(t, err)
		var stored FirewallNodeMetadata
		require.NoError(t, mapstructure.Decode(meta.Get(), &stored))
		assert.Equal(t, "allow-http", stored.FirewallName)
	})
}

func Test__CreateFirewall__Execute(t *testing.T) {
	component := &CreateFirewall{}

	t.Run("creates allow ingress rule -> emits created event", func(t *testing.T) {
		var postPath string
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postPath = path
				postBody = body.(map[string]any)
				return opDone("op-create"), nil
			},
			getFunc: firewallExecGet("op-create", firewallGetJSON("allow-http", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":         "allow-http",
				"network":      "default",
				"direction":    "INGRESS",
				"action":       "allow",
				"priority":     1000,
				"rules":        []map[string]any{{"protocol": "tcp", "ports": "80, 443"}},
				"sourceRanges": []string{"0.0.0.0/0"},
				"targetTags":   []string{"web"},
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.firewallRule.created", state.Type)
		assert.True(t, strings.HasSuffix(postPath, "/global/firewalls"))

		// Body should carry an allowed array (not denied), resolved network, and source ranges.
		assert.Contains(t, postBody, "allowed")
		assert.NotContains(t, postBody, "denied")
		assert.Contains(t, postBody["network"], "global/networks/default")
		assert.Equal(t, []string{"0.0.0.0/0"}, postBody["sourceRanges"])
		allowed := postBody["allowed"].([]map[string]any)
		assert.Equal(t, "tcp", allowed[0]["IPProtocol"])
		assert.Equal(t, []string{"80", "443"}, allowed[0]["ports"])

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "allow-http", data["name"])
		assert.Equal(t, "ALLOW", data["action"])
		assert.Contains(t, data["link"], "console.cloud.google.com/networking/firewalls/details/allow-http")
	})

	t.Run("deny action populates denied array", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("deny-ssh", "INGRESS", "deny")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":    "deny-ssh",
				"network": "default",
				"action":  "deny",
				"rules":   []map[string]any{{"protocol": "tcp", "ports": "22"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Contains(t, postBody, "denied")
		assert.NotContains(t, postBody, "allowed")
	})

	t.Run("egress rule uses destination ranges", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("egress-rule", "EGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":              "egress-rule",
				"network":           "default",
				"direction":         "EGRESS",
				"rules":             []map[string]any{{"protocol": "tcp", "ports": "443"}},
				"destinationRanges": []string{"10.0.0.0/8"},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"10.0.0.0/8"}, postBody["destinationRanges"])
		assert.NotContains(t, postBody, "sourceRanges")
	})

	t.Run("explicit priority 0 is sent, not dropped", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postBody = body.(map[string]any)
				return opDone("op"), nil
			},
			getFunc: firewallExecGet("op", firewallGetJSON("top-prio", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "top-prio",
				"network":  "default",
				"priority": 0,
				"rules":    []map[string]any{{"protocol": "tcp", "ports": "443"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, 0, postBody["priority"])
	})

	t.Run("out-of-range priority -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "bad-prio",
				"network":  "default",
				"priority": 70000,
				"rules":    []map[string]any{{"protocol": "tcp", "ports": "443"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "must be between 0 and 65535")
	})

	t.Run("empty protocols -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":    "bad",
				"network": "default",
				"rules":   []map[string]any{{"protocol": "", "ports": ""}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "at least one protocol")
	})

	t.Run("API error on create -> fails execution", func(t *testing.T) {
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				return []byte("not-json"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":    "allow-http",
				"network": "default",
				"rules":   []map[string]any{{"protocol": "tcp", "ports": "80"}},
			},
			ExecutionState: state,
		})
		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "create firewall rule operation response")
	})

	t.Run("target service accounts and logging are sent in the body", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postBody = body.(map[string]any)
				return opDone("op-create"), nil
			},
			getFunc: firewallExecGet("op-create", firewallGetJSON("allow-sa", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                  "allow-sa",
				"network":               "default",
				"direction":             "INGRESS",
				"action":                "allow",
				"rules":                 []map[string]any{{"protocol": "tcp", "ports": "443"}},
				"targetServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
				"enableLogging":         true,
				"logMetadata":           "EXCLUDE_ALL_METADATA",
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{"sa@my-project.iam.gserviceaccount.com"}, postBody["targetServiceAccounts"])
		assert.NotContains(t, postBody, "targetTags")
		logCfg := postBody["logConfig"].(map[string]any)
		assert.Equal(t, true, logCfg["enable"])
		assert.Equal(t, "EXCLUDE_ALL_METADATA", logCfg["metadata"])
	})

	t.Run("mixing network tags and service accounts -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                  "mix",
				"network":               "default",
				"rules":                 []map[string]any{{"protocol": "tcp", "ports": "80"}},
				"targetTags":            []string{"web"},
				"targetServiceAccounts": []string{"sa@my-project.iam.gserviceaccount.com"},
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "cannot combine network tags and service accounts")
	})

	t.Run("dropdown and custom service accounts are merged (deduped) in the body", func(t *testing.T) {
		var postBody map[string]any
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				postBody = body.(map[string]any)
				return opDone("op-create"), nil
			},
			getFunc: firewallExecGet("op-create", firewallGetJSON("allow-sa", "INGRESS", "allow")),
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                        "allow-sa",
				"network":                     "default",
				"direction":                   "INGRESS",
				"action":                      "allow",
				"rules":                       []map[string]any{{"protocol": "tcp", "ports": "443"}},
				"targetServiceAccounts":       []string{"a@my-project.iam.gserviceaccount.com"},
				"targetServiceAccountsCustom": []string{"b@other-project.iam.gserviceaccount.com", "a@my-project.iam.gserviceaccount.com"},
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.True(t, state.Passed)
		assert.Equal(t, []string{
			"a@my-project.iam.gserviceaccount.com",
			"b@other-project.iam.gserviceaccount.com",
		}, postBody["targetServiceAccounts"])
	})

	t.Run("non-service-account email in custom field -> fails before API call", func(t *testing.T) {
		var called bool
		mc := &mockFirewallClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				called = true
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":                        "bad-sa",
				"network":                     "default",
				"rules":                       []map[string]any{{"protocol": "tcp", "ports": "80"}},
				"targetServiceAccountsCustom": []string{"someone@example.com"},
			},
			ExecutionState: state,
		})

		require.NoError(t, err)
		assert.False(t, state.Passed)
		assert.False(t, called)
		assert.Contains(t, state.FailureMessage, "not a service account email")
	})
}
