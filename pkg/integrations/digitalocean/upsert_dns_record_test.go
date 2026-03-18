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
				"type": "A",
				"name": "www",
				"data": "1.2.3.4",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "domain is required")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "type is required")
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "INVALID",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "invalid record type")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"data":   "1.2.3.4",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing data returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "data is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__UpsertDNSRecord__Execute(t *testing.T) {
	component := &UpsertDNSRecord{}

	t.Run("no existing record -> creates new record and emits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: ListDNSRecords (returns empty list)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": []
					}`)),
				},
				// Second call: CreateDNSRecord
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_record": {
							"id": 12345678,
							"type": "A",
							"name": "www",
							"data": "1.2.3.4",
							"priority": null,
							"port": null,
							"ttl": 1800,
							"weight": null
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.dns.record.upserted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("existing record matches name and type -> updates it and emits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: ListDNSRecords (returns existing record)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": [
							{
								"id": 12345678,
								"type": "A",
								"name": "www",
								"data": "1.1.1.1",
								"priority": null,
								"port": null,
								"ttl": 1800,
								"weight": null
							}
						]
					}`)),
				},
				// Second call: UpdateDNSRecord
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_record": {
							"id": 12345678,
							"type": "A",
							"name": "www",
							"data": "2.2.2.2",
							"priority": null,
							"port": null,
							"ttl": 1800,
							"weight": null
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
				"data":   "2.2.2.2",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.dns.record.upserted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("existing record has different type -> creates new record", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: ListDNSRecords (returns record with different type)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": [
							{
								"id": 11111111,
								"type": "TXT",
								"name": "www",
								"data": "some-text",
								"priority": null,
								"port": null,
								"ttl": 1800,
								"weight": null
							}
						]
					}`)),
				},
				// Second call: CreateDNSRecord (no match found for type A)
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_record": {
							"id": 22222222,
							"type": "A",
							"name": "www",
							"data": "1.2.3.4",
							"priority": null,
							"port": null,
							"ttl": 1800,
							"weight": null
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.dns.record.upserted", executionState.Type)
	})

	t.Run("list DNS records API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"domain does not exist"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain": "nonexistent.com",
				"type":   "A",
				"name":   "www",
				"data":   "1.2.3.4",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list DNS records")
		assert.False(t, executionState.Passed)
	})

	t.Run("update DNS record API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: ListDNSRecords (returns existing record)
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"domain_records": [
							{
								"id": 12345678,
								"type": "A",
								"name": "www",
								"data": "1.1.1.1",
								"priority": null,
								"port": null,
								"ttl": 1800,
								"weight": null
							}
						]
					}`)),
				},
				// Second call: UpdateDNSRecord - fails
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"type":   "A",
				"name":   "www",
				"data":   "2.2.2.2",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update DNS record")
		assert.False(t, executionState.Passed)
	})
}
