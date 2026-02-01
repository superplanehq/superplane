package cloudflare

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteDNSRecord__Setup(t *testing.T) {
	component := &DeleteDNSRecord{}

	t.Run("missing record returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"record": "",
			},
		})
		require.ErrorContains(t, err, "record is required")
	})

	t.Run("valid config passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"record": "zone123/record123",
			},
		})
		require.NoError(t, err)
	})
}

func Test__DeleteDNSRecord__Execute(t *testing.T) {
	component := &DeleteDNSRecord{}

	t.Run("zone/record ID -> successful delete emits on default channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "record123",
								"type": "CNAME",
								"name": "app.example.com",
								"content": "example.com",
								"ttl": 360,
								"proxied": false
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
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"record": "zone123/record123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records/record123", httpContext.Requests[0].URL.String())
	})

	t.Run("404 from Cloudflare -> error returned", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message":"Not found"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"record": "zone123/record123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "zone or DNS record not found")
	})

	t.Run("record name not found in any zone -> error returned", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": []}`)),
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

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"record": "nonexistent.example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no DNS record found with name")
	})

	t.Run("record ID only (e.g. from createDnsRecord.data.id) -> resolved by ID and deleted", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": [
								{
									"id": "6810f944d5310fa0d710f5c13f45ce5a",
									"type": "CNAME",
									"name": "app.example.com",
									"content": "example.com",
									"ttl": 360,
									"proxied": false
								}
							]
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "6810f944d5310fa0d710f5c13f45ce5a",
								"type": "CNAME",
								"name": "app.example.com",
								"content": "example.com",
								"ttl": 360,
								"proxied": false
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

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"record": "6810f944d5310fa0d710f5c13f45ce5a",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/zones/zone123/dns_records")
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records/6810f944d5310fa0d710f5c13f45ce5a", httpContext.Requests[1].URL.String())
	})
}
