package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateDNSRecord__Setup(t *testing.T) {
	component := &CreateDNSRecord{}

	t.Run("missing zone returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "",
				"name":    "api",
				"content": "192.0.2.1",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "type is required")
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "BAD",
				"name":    "api",
				"content": "192.0.2.1",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "type must be one of")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "",
				"content": "192.0.2.1",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing content returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "content is required")
	})

	t.Run("proxied set for unsupported type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "TXT",
				"name":    "verification",
				"content": "token",
				"proxied": true,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "proxied is only supported")
	})

	t.Run("priority set for unsupported type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"type":     "A",
				"name":     "api",
				"content":  "192.0.2.1",
				"priority": 10,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "priority is only supported")
	})

	t.Run("ttl 0 returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     0,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "TTL must be 1 (automatic) or between 60 and 86400 seconds")
	})

	t.Run("ttl between 2 and 59 returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     50,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "TTL must be 1 (automatic) or between 60 and 86400 seconds")
	})

	t.Run("ttl above 86400 returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     90000,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "TTL must be 1 (automatic) or between 60 and 86400 seconds")
	})

	t.Run("ttl 1 is valid", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     1,
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("ttl 60 and 86400 are valid", func(t *testing.T) {
		for _, ttl := range []int{60, 86400} {
			ctx := core.SetupContext{
				Configuration: map[string]any{
					"zone":    "zone123",
					"type":    "A",
					"name":    "api",
					"content": "192.0.2.1",
					"ttl":     ttl,
				},
			}

			err := component.Setup(ctx)
			require.NoError(t, err, "TTL %d should be valid", ttl)
		}
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     120,
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__CreateDNSRecord__Execute(t *testing.T) {
	component := &CreateDNSRecord{}

	t.Run("successful create emits record", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "record123",
								"type": "A",
								"name": "api.example.com",
								"content": "192.0.2.1",
								"proxied": true,
								"ttl": 120
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "zone123", Name: "example.com", Status: "active"},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":    "example.com",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
				"ttl":     120,
				"proxied": true,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records", httpContext.Requests[0].URL.String())

		bodyBytes, err := io.ReadAll(httpContext.Requests[0].Body)
		require.NoError(t, err)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(bodyBytes, &payload))
		assert.Equal(t, "A", payload["type"])
		assert.Equal(t, "api", payload["name"])
		assert.Equal(t, "192.0.2.1", payload["content"])
		assert.Equal(t, float64(120), payload["ttl"])
		assert.Equal(t, true, payload["proxied"])

		output := execState.Payloads[0].(map[string]any)
		data := output["data"].(map[string]any)
		assert.Equal(t, "record123", data["id"])
		assert.Equal(t, "A", data["type"])
		assert.Equal(t, "api.example.com", data["name"])
	})

	t.Run("validation error fails run", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"code": 1004, "message": "DNS record is invalid"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create DNS record")
	})

	t.Run("unauthorized error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"code": 9109, "message": "Invalid token"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":    "zone123",
				"type":    "A",
				"name":    "api",
				"content": "192.0.2.1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
	})
}
