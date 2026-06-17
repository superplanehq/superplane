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
					return []byte(`{"selfLink":"https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/forwardingRules/web-lb-fr","IPAddress":"34.1.2.3"}`), nil
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
		// The emitted forwarding rule must be a reference Delete can consume.
		_, _, frParsed, perr := parseRegionalResource(data["forwardingRule"].(string), "forwardingRules")
		require.NoError(t, perr)
		assert.Equal(t, "web-lb-fr", frParsed)
	})

	t.Run("resolves a reserved IP selfLink to its literal address", func(t *testing.T) {
		ipLink := "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/addresses/web-ip"
		var frIP any
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				if strings.HasSuffix(path, "/forwardingRules") {
					frIP = body.(map[string]any)["IPAddress"]
				}
				return opDone("op"), nil
			},
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/addresses/"):
					return []byte(`{"address":"35.1.1.1"}`), nil
				case strings.Contains(path, "/healthChecks/"):
					return []byte(`{"selfLink":"hc-link"}`), nil
				case strings.Contains(path, "/backendServices/"):
					return []byte(`{"selfLink":"bes-link"}`), nil
				case strings.Contains(path, "/forwardingRules/"):
					return []byte(`{"selfLink":"fr-link","IPAddress":"35.1.1.1"}`), nil
				}
				return nil, assert.AnError
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "web-lb", "region": "us-central1", "ports": []any{"80"},
				"instanceGroup": ig, "ipAddress": ipLink,
			},
			ExecutionState: state,
		}))
		assert.True(t, state.Passed)
		// The forwarding rule receives the literal IP, not the selfLink.
		assert.Equal(t, "35.1.1.1", frIP)
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

	t.Run("does not roll back a resource it did not create", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc: func(ctx context.Context, path string, body any) ([]byte, error) {
				if strings.HasSuffix(path, "/backendServices") {
					// Insert rejected (e.g. a backend service of this name already
					// exists for another load balancer) — we did not create it.
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
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "web-lb", "region": "us-central1", "ports": []any{"80"}, "instanceGroup": ig},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		// Only the health check (which we did create) is rolled back; the
		// pre-existing backend service is left untouched.
		assert.Equal(t, []string{"hc"}, deleted)
	})

	t.Run("rolls back a resource whose insert completed but read-back failed", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc:  func(ctx context.Context, path string, body any) ([]byte, error) { return opDone("op"), nil },
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/healthChecks/"):
					return []byte(`{"selfLink":"hc-link"}`), nil
				case strings.Contains(path, "/backendServices/"):
					return nil, assert.AnError // insert finished, but the read-back fails
				}
				return nil, assert.AnError
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
		assert.Contains(t, state.FailureMessage, "backend service")
		// The backend service insert completed but its read-back failed; rollback
		// must still tear it down (along with the health check).
		assert.ElementsMatch(t, []string{"bes", "hc"}, deleted)
	})

	t.Run("rolls back and fails when the forwarding rule has no assigned IP", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			postFunc:  func(ctx context.Context, path string, body any) ([]byte, error) { return opDone("op"), nil },
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/healthChecks/"):
					return []byte(`{"selfLink":"hc-link"}`), nil
				case strings.Contains(path, "/backendServices/"):
					return []byte(`{"selfLink":"bes-link"}`), nil
				case strings.Contains(path, "/forwardingRules/"):
					return []byte(`{"selfLink":"fr-link"}`), nil // no IPAddress assigned
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
		require.NoError(t, c.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"name": "web-lb", "region": "us-central1", "ports": []any{"80"}, "instanceGroup": ig},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "no IP address")
		assert.ElementsMatch(t, []string{"fr", "bes", "hc"}, deleted)
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

	t.Run("fails without deleting when the forwarding rule cannot be read", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				if strings.Contains(path, "/operations/") {
					return opDone("op"), nil
				}
				return nil, assert.AnError // forwarding rule GET fails
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				deleted = append(deleted, "x")
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"loadBalancer": fr},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "forwarding rule")
		// Nothing is deleted, so the backend service is not orphaned.
		assert.Empty(t, deleted)
	})

	t.Run("fails without deleting when the backend service cannot be read", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/forwardingRules/"):
					return []byte(`{"name":"web-lb-fr","backendService":"https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/backendServices/web-lb-backend"}`), nil
				}
				return nil, assert.AnError // backend service GET fails
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				deleted = append(deleted, "x")
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"loadBalancer": fr},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "backend service")
		// Nothing is deleted, so neither the backend service nor the health check is orphaned.
		assert.Empty(t, deleted)
	})

	t.Run("retries cleanup when the forwarding rule is already gone", func(t *testing.T) {
		var deleted []string
		notFound := &gcpcommon.GCPAPIError{StatusCode: http.StatusNotFound, Message: "not found"}
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/forwardingRules/"):
					return nil, notFound // already deleted by a previous, partially failed run
				}
				return nil, assert.AnError
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/forwardingRules/"):
					deleted = append(deleted, "fr")
					return nil, notFound // already gone
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
		// The forwarding rule was already gone; the leftover backend service and
		// health check (recovered from the naming convention) are still cleaned up.
		assert.ElementsMatch(t, []string{"fr", "bes", "hc"}, deleted)
	})

	t.Run("rejects a forwarding rule with no backend service", func(t *testing.T) {
		var deleted []string
		mc := &mockStaticIPClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				switch {
				case strings.Contains(path, "/operations/"):
					return opDone("op"), nil
				case strings.Contains(path, "/forwardingRules/"):
					// A legacy target-pool NLB: a target, no backend service.
					return []byte(`{"name":"web-lb-fr","target":"https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/targetPools/legacy-pool"}`), nil
				}
				return nil, assert.AnError
			},
			deleteFunc: func(ctx context.Context, path string) ([]byte, error) {
				deleted = append(deleted, "x")
				return opDone("op"), nil
			},
		}
		SetClientFactory(func(ctx core.ExecutionContext) (Client, error) { return mc, nil })

		state := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		require.NoError(t, d.Execute(core.ExecutionContext{
			Configuration:  map[string]any{"loadBalancer": fr},
			ExecutionState: state,
		}))
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "backend service")
		// Nothing is deleted, so the target pool is not left orphaned.
		assert.Empty(t, deleted)
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
