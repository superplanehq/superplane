package compute

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__validatePorts(t *testing.T) {
	t.Run("valid ports", func(t *testing.T) {
		out, err := validatePorts([]string{"80", " 443 "})
		require.NoError(t, err)
		assert.Equal(t, []string{"80", "443"}, out)
	})
	t.Run("empty rejected", func(t *testing.T) {
		_, err := validatePorts(nil)
		require.ErrorContains(t, err, "at least one port")
	})
	t.Run("out of range rejected", func(t *testing.T) {
		_, err := validatePorts([]string{"70000"})
		require.ErrorContains(t, err, "invalid port")
	})
}

func Test__CreateLoadBalancer__Setup(t *testing.T) {
	c := &CreateLoadBalancer{}
	setup := func(cfg map[string]any) error {
		return c.Setup(core.SetupContext{Configuration: cfg, Metadata: &contexts.MetadataContext{}})
	}
	base := map[string]any{"name": "web-lb", "region": "us-central1", "instanceGroup": "ig", "ports": []any{"80"}}

	t.Run("missing name", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"region": "us-central1", "instanceGroup": "ig", "ports": []any{"80"}}), "name is required")
	})
	t.Run("missing instance group", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "web-lb", "region": "us-central1", "ports": []any{"80"}}), "instance group is required")
	})
	t.Run("no ports", func(t *testing.T) {
		require.ErrorContains(t, setup(map[string]any{"name": "web-lb", "region": "us-central1", "instanceGroup": "ig"}), "at least one port")
	})
	t.Run("valid", func(t *testing.T) {
		require.NoError(t, setup(base))
	})
}

func Test__CreateLoadBalancer__Execute(t *testing.T) {
	c := &CreateLoadBalancer{}
	ig := "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/instanceGroups/web-servers"

	t.Run("creates the chain in order and emits created", func(t *testing.T) {
		var posts []string
		bodies := map[string]map[string]any{}
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				switch {
				case strings.HasSuffix(path, "/healthChecks"):
					posts, bodies["hc"] = append(posts, "hc"), body.(map[string]any)
				case strings.HasSuffix(path, "/backendServices"):
					posts, bodies["bes"] = append(posts, "bes"), body.(map[string]any)
				case strings.HasSuffix(path, "/forwardingRules"):
					posts, bodies["fr"] = append(posts, "fr"), body.(map[string]any)
				}
				return opDone("op"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/healthChecks/"):
					return []byte(`{"selfLink":"hc-link"}`), nil
				case strings.Contains(path, "/backendServices/"):
					return []byte(`{"selfLink":"bes-link"}`), nil
				case strings.Contains(path, "/forwardingRules/"):
					return []byte(`{"selfLink":"fr-link","IPAddress":"34.1.2.3"}`), nil
				}
				return nil, assert.AnError
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "web-lb", "region": "us-central1", "protocol": "TCP",
				"ports": []any{"80", "443"}, "instanceGroup": ig,
			},
			ExecutionState: state,
		}))

		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.loadBalancer.created", state.Type)
		assert.Equal(t, []string{"hc", "bes", "fr"}, posts)
		assert.Equal(t, []string{"hc-link"}, bodies["bes"]["healthChecks"])
		assert.Equal(t, ig, bodies["bes"]["backends"].([]any)[0].(map[string]any)["group"])
		assert.Equal(t, "bes-link", bodies["fr"]["backendService"])
		assert.Equal(t, []string{"80", "443"}, bodies["fr"]["ports"])

		data := state.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "web-lb", data["name"])
		assert.Equal(t, "34.1.2.3", data["ipAddress"])
		assert.Equal(t, "web-lb-fr", data["forwardingRule"])
	})

	t.Run("rolls back created resources when the forwarding rule fails", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				if strings.HasSuffix(path, "/forwardingRules") {
					return nil, assert.AnError
				}
				return opDone("op"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op"), nil
				}
				return []byte(`{"selfLink":"link"}`), nil
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/backendServices/"):
					deleted = append(deleted, "bes")
				case strings.Contains(path, "/healthChecks/"):
					deleted = append(deleted, "hc")
				}
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "web-lb", "region": "us-central1", "ports": []any{"80"}, "instanceGroup": ig},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "forwarding rule")
		// Backend service and health check created before the failure are torn down.
		assert.ElementsMatch(t, []string{"bes", "hc"}, deleted)
	})
}

func Test__DeleteLoadBalancer__Execute(t *testing.T) {
	d := &DeleteLoadBalancer{}
	fr := "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/forwardingRules/web-lb-fr"

	t.Run("deletes forwarding rule, backend service and health check", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/forwardingRules/"):
					return []byte(`{"name":"web-lb-fr","backendService":"https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/backendServices/web-lb-backend"}`), nil
				case strings.Contains(path, "/backendServices/"):
					return []byte(`{"name":"web-lb-backend","healthChecks":["https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/healthChecks/web-lb-hc"]}`), nil
				}
				return nil, assert.AnError
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/forwardingRules/"):
					deleted = append(deleted, "fr")
				case strings.Contains(path, "/backendServices/"):
					deleted = append(deleted, "bes")
				case strings.Contains(path, "/healthChecks/"):
					deleted = append(deleted, "hc")
				}
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"loadBalancer": fr},
			ExecutionState: state,
		}))
		assert.True(t, state.Passed)
		assert.Equal(t, "gcp.compute.loadBalancer.deleted", state.Type)
		assert.Equal(t, []string{"fr", "bes", "hc"}, deleted)
	})

	t.Run("rejects a cross-project load balancer", func(t *testing.T) {
		mc := &mockStaticIPClient{projectID: "my-project"}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })
		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"loadBalancer": "https://www.googleapis.com/compute/v1/projects/other/regions/us-central1/forwardingRules/x"},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "cross-project")
	})
}
