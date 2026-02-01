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

func Test__UpdateDNSRecord__Setup(t *testing.T) {
	component := &UpdateDNSRecord{}

	t.Run("missing zone returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "",
				"recordId": "record123",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing recordId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "recordId is required")
	})

	t.Run("missing content returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "record123",
				"content":  "",
				"ttl":      360,
				"proxied":  false,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "content is required")
	})

	t.Run("ttl < 1 returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "record123",
				"content":  "1.2.3.4",
				"ttl":      0,
				"proxied":  false,
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "ttl must be")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "record123",
				"content":  "1.2.3.4",
				"ttl":      360,
				"proxied":  true,
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__UpdateDNSRecord__Execute(t *testing.T) {
	component := &UpdateDNSRecord{}

	t.Run("successful update emits success channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Response for GetDNSRecord
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "record123",
								"type": "A",
								"name": "app.example.com",
								"content": "1.1.1.1",
								"ttl": 120,
								"proxied": false
							}
						}
					`)),
				},
				// Response for UpdateDNSRecord
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "record123",
								"type": "A",
								"name": "app.example.com",
								"content": "2.2.2.2",
								"ttl": 1,
								"proxied": true
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
				"zone":     "example.com", // ensure domain name resolves to zone id
				"recordId": "record123",
				"content":  "2.2.2.2",
				"ttl":      1,
				"proxied":  true,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)

		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records/record123", httpContext.Requests[0].URL.String())

		assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records/record123", httpContext.Requests[1].URL.String())

		body, readErr := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, readErr)

		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		assert.Equal(t, "A", req["type"])
		assert.Equal(t, "app.example.com", req["name"])
		assert.Equal(t, "2.2.2.2", req["content"])
		assert.Equal(t, float64(1), req["ttl"])
		assert.Equal(t, true, req["proxied"])
	})

	t.Run("record lookup error returns error and does not emit", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Record not found"}]}`)),
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
				"zone":     "zone123",
				"recordId": "record123",
				"content":  "1.2.3.4",
				"ttl":      360,
				"proxied":  false,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.ErrorContains(t, err, "get DNS record")

		assert.False(t, execState.Passed)
		assert.Empty(t, execState.Channel)
	})
}
