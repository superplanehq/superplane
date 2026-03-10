package digitalocean

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

func Test__UpsertDNSRecord__Setup(t *testing.T) {
	component := &UpsertDNSRecord{}

	t.Run("missing domain returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"recordType": "A",
				"name":       "www",
				"data":       "1.2.3.4",
			},
		})

		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain":     "example.com",
				"recordType": "A",
				"name":       "www",
				"data":       "1.2.3.4",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpsertDNSRecord__Execute(t *testing.T) {
	component := &UpsertDNSRecord{}

	t.Run("record does not exist -> creates new record", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": [
							{"id": 1, "type": "NS", "name": "@", "data": "ns1.digitalocean.com", "ttl": 1800}
						]
					}`)),
				},
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_record": {
							"id": 12345,
							"type": "A",
							"name": "www",
							"data": "104.131.186.241",
							"ttl": 1800
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain":     "example.com",
				"recordType": "A",
				"name":       "www",
				"data":       "104.131.186.241",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.dns_record.upserted", executionState.Type)
	})

	t.Run("record exists -> updates existing record", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": [
							{"id": 12345, "type": "A", "name": "www", "data": "1.2.3.4", "ttl": 1800}
						]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_record": {
							"id": 12345,
							"type": "A",
							"name": "www",
							"data": "104.131.186.241",
							"ttl": 1800
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain":     "example.com",
				"recordType": "A",
				"name":       "www",
				"data":       "104.131.186.241",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.dns_record.upserted", executionState.Type)
	})
}
