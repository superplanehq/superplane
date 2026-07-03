package openai

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const usagePageBody = `{
	"object": "page",
	"data": [
		{
			"object": "bucket",
			"start_time": 1730419200,
			"end_time": 1730505600,
			"results": [
				{
					"object": "organization.usage.completions.result",
					"input_tokens": 1000,
					"output_tokens": 500,
					"input_cached_tokens": 800,
					"num_model_requests": 5
				}
			]
		}
	],
	"has_more": false,
	"next_page": ""
}`

func Test__GetUsage__Execute(t *testing.T) {
	c := &GetUsage{}

	t.Run("success with default configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(usagePageBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Contains(t, req.URL.String(), "https://api.openai.com/v1/organization/usage/completions")
		assert.Equal(t, "1d", req.URL.Query().Get("bucket_width"))
		assert.NotEmpty(t, req.URL.Query().Get("start_time"))
		assert.NotEmpty(t, req.URL.Query().Get("end_time"))
		assert.Equal(t, "Bearer test-admin-key", req.Header.Get("Authorization"))

		assert.Equal(t, core.DefaultOutputChannel.Name, executionStateCtx.Channel)
		assert.Equal(t, GetUsagePayloadType, executionStateCtx.Type)
	})

	t.Run("success with costs and custom dates", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"object": "page",
						"data": [],
						"has_more": false,
						"next_page": ""
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"usageType": "costs",
				"startDate": "2026-06-18",
				"endDate":   "2026-06-20",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/organization/costs")
		assert.Equal(t, "3", httpContext.Requests[0].URL.Query().Get("limit"))
	})

	t.Run("group by model is sent as group_by", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(usagePageBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"groupBy": "model",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "model", httpContext.Requests[0].URL.Query().Get("group_by"))
	})

	t.Run("paginates while has_more is set", func(t *testing.T) {
		firstPage := strings.Replace(usagePageBody, `"has_more": false`, `"has_more": true`, 1)
		firstPage = strings.Replace(firstPage, `"next_page": ""`, `"next_page": "page_2"`, 1)

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(firstPage)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(usagePageBody)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID:             uuid.New(),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "page_2", httpContext.Requests[1].URL.Query().Get("page"))
	})

	t.Run("invalid start date format -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "invalid-date",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start date format")
	})

	t.Run("start date after end date -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "test-key",
				"adminKey": "test-admin-key",
			},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2026-06-25",
				"endDate":   "2026-06-20",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start date must be before end date")
	})

	t.Run("missing admin key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		execCtx := core.ExecutionContext{
			ID:            uuid.New(),
			Configuration: map[string]any{},
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "admin API key is not configured")
	})
}

func Test__GetUsage__Setup(t *testing.T) {
	c := &GetUsage{}

	tests := []struct {
		name        string
		config      map[string]any
		expectError string
	}{
		{name: "empty configuration", config: map[string]any{}},
		{name: "valid usage type", config: map[string]any{"usageType": "embeddings"}},
		{name: "costs with line item grouping", config: map[string]any{"usageType": "costs", "groupBy": "line_item"}},
		{
			name:        "invalid usage type",
			config:      map[string]any{"usageType": "bogus"},
			expectError: "invalid usage type",
		},
		{
			name:        "line item grouping outside costs",
			config:      map[string]any{"usageType": "completions", "groupBy": "line_item"},
			expectError: "line item grouping is only available for costs",
		},
		{
			name:        "model grouping for costs",
			config:      map[string]any{"usageType": "costs", "groupBy": "model"},
			expectError: "model grouping is not available for costs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Setup(core.SetupContext{Configuration: tt.config})
			if tt.expectError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
		})
	}
}
