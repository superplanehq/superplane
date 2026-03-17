package digitalocean

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateLoadBalancer__Setup(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing region returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "my-lb",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing forwarding rules returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "at least one forwarding rule is required")
	})

	t.Run("forwarding rule missing entryProtocol returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "entryProtocol is required")
	})

	t.Run("forwarding rule missing entryPort returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "entryPort is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__CreateLoadBalancer__Execute(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("successful creation -> stores LB ID in metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "new",
							"region": {"slug": "nyc3", "name": "New York 3"},
							"forwarding_rules": [{"entry_protocol":"http","entry_port":80,"target_protocol":"http","target_port":80}]
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "4de7ac8b-495b-4884-9a69-1050c6793cd6", metadata["lbID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 15*time.Second, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Region is not available"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create load balancer")
	})

	t.Run("with droplet IDs -> sends droplet_ids in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "new",
							"region": {"slug": "nyc3", "name": "New York 3"},
							"droplet_ids": [12345, 67890],
							"forwarding_rules": [{"entry_protocol":"http","entry_port":80,"target_protocol":"http","target_port":80}]
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
				"droplets": []any{"12345", "67890"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "4de7ac8b-495b-4884-9a69-1050c6793cd6", metadata["lbID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 15*time.Second, requestCtx.Duration)
		assert.False(t, executionState.Passed)

		// Verify the request body contains droplet_ids
		require.Len(t, httpContext.Requests, 1)
		reqBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(reqBody), `"droplet_ids":[12345,67890]`)
	})

	t.Run("with tag -> sends tag in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "new",
							"region": {"slug": "nyc3", "name": "New York 3"},
							"tag": "web-servers",
							"forwarding_rules": [{"entry_protocol":"http","entry_port":80,"target_protocol":"http","target_port":80}]
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
				"tag": "web-servers",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "4de7ac8b-495b-4884-9a69-1050c6793cd6", metadata["lbID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.False(t, executionState.Passed)

		// Verify the request body contains the tag
		require.Len(t, httpContext.Requests, 1)
		reqBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(reqBody), `"tag":"web-servers"`)
	})

	t.Run("invalid droplet ID -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
				},
				"droplets": []any{"not-a-number"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid droplet ID")
	})

	t.Run("with multiple forwarding rules and droplets -> sends all in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "new",
							"region": {"slug": "nyc3", "name": "New York 3"},
							"droplet_ids": [111, 222, 333],
							"forwarding_rules": [
								{"entry_protocol":"http","entry_port":80,"target_protocol":"http","target_port":80},
								{"entry_protocol":"https","entry_port":443,"target_protocol":"http","target_port":8080,"tls_passthrough":true}
							]
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":   "my-lb",
				"region": "nyc3",
				"forwardingRules": []any{
					map[string]any{
						"entryProtocol":  "http",
						"entryPort":      80,
						"targetProtocol": "http",
						"targetPort":     80,
					},
					map[string]any{
						"entryProtocol":  "https",
						"entryPort":      443,
						"targetProtocol": "http",
						"targetPort":     8080,
						"tlsPassthrough": true,
					},
				},
				"droplets": []any{"111", "222", "333"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "4de7ac8b-495b-4884-9a69-1050c6793cd6", metadata["lbID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.False(t, executionState.Passed)

		// Verify the request body contains all droplet IDs and forwarding rules
		require.Len(t, httpContext.Requests, 1)
		reqBody, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)
		assert.Contains(t, string(reqBody), `"droplet_ids":[111,222,333]`)
		assert.Contains(t, string(reqBody), `"tls_passthrough":true`)
	})
}

func Test__CreateLoadBalancer__HandleAction(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("poll: status new -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "new"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"lbID": "4de7ac8b-495b-4884-9a69-1050c6793cd6"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.False(t, executionState.Passed)
	})

	t.Run("poll: status active -> emits success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"ip": "104.131.186.241",
							"status": "active",
							"algorithm": "round_robin",
							"region": {"slug": "nyc3", "name": "New York 3"}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"lbID": "4de7ac8b-495b-4884-9a69-1050c6793cd6"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.loadbalancer.created", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("poll: status errored -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"load_balancer": {
							"id": "4de7ac8b-495b-4884-9a69-1050c6793cd6",
							"name": "my-lb",
							"status": "errored"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"lbID": "4de7ac8b-495b-4884-9a69-1050c6793cd6"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error status")
	})

	t.Run("unknown action -> returns error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           "unknown",
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}
