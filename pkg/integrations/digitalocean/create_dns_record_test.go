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

func Test__CreateDNSRecord__Setup(t *testing.T) {
	component := &CreateDNSRecord{}

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

	t.Run("missing recordType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain": "example.com",
				"name":   "www",
				"data":   "1.2.3.4",
			},
		})

		require.ErrorContains(t, err, "recordType is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain":     "example.com",
				"recordType": "A",
				"data":       "1.2.3.4",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing data returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"domain":     "example.com",
				"recordType": "A",
				"name":       "www",
			},
		})

		require.ErrorContains(t, err, "data is required")
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

func Test__CreateDNSRecord__Execute(t *testing.T) {
	component := &CreateDNSRecord{}

	t.Run("successful creation -> emits record", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
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
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.dns_record.created", executionState.Type)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"domain not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"domain":     "nonexistent.com",
				"recordType": "A",
				"name":       "www",
				"data":       "1.2.3.4",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create DNS record")
	})
}
