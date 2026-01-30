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

	t.Run("missing zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "",
				"recordId": "record123",
			},
		})
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing recordId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "",
			},
		})
		require.ErrorContains(t, err, "recordId is required")
	})

	t.Run("valid config passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":     "zone123",
				"recordId": "record123",
			},
		})
		require.NoError(t, err)
	})
}

func Test__DeleteDNSRecord__Execute(t *testing.T) {
	component := &DeleteDNSRecord{}

	t.Run("zone name resolved from metadata -> successful delete emits success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "record123"
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
				"zone":     "example.com",
				"recordId": "record123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, DNSRecordDeleteSuccessChannel, execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/dns_records/record123", httpContext.Requests[0].URL.String())
	})

	t.Run("zone name not in metadata -> failed emitted without HTTP call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
			Metadata: Metadata{
				Zones: []Zone{
					{ID: "zone123", Name: "other.com", Status: "active"},
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":     "example.com",
				"recordId": "record123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, DNSRecordDeleteFailedChannel, execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)
		assert.Len(t, httpContext.Requests, 0)
	})

	t.Run("404 from Cloudflare -> failed channel with statusCode", func(t *testing.T) {
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
				"zone":     "zone123",
				"recordId": "record123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, DNSRecordDeleteFailedChannel, execState.Channel)
		assert.Equal(t, DNSRecordPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)
	})
}

