package cursor

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

func Test__GetDailyUsageData__Execute(t *testing.T) {
	c := &GetDailyUsageData{}

	t.Run("success with default dates", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{
								"date": 1710720000000,
								"isActive": true,
								"totalLinesAdded": 1543,
								"email": "dev@company.com"
							}
						],
						"period": {
							"startDate": 1710720000000,
							"endDate": 1710892800000
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
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
		assert.Equal(t, "https://api.cursor.com/teams/daily-usage-data", httpContext.Requests[0].URL.String())

		assert.Equal(t, core.DefaultOutputChannel.Name, executionStateCtx.Channel)
		assert.Equal(t, GetDailyUsageDataPayloadType, executionStateCtx.Type)
	})

	t.Run("success with custom dates", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [],
						"period": {
							"startDate": 1710720000000,
							"endDate": 1710892800000
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2024-03-18",
				"endDate":   "2024-03-20",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)
	})

	t.Run("invalid start date format -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
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

	t.Run("invalid end date format -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
			},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"endDate": "invalid-date",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid end date format")
	})

	t.Run("start date after end date -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
			},
		}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2024-03-25",
				"endDate":   "2024-03-20",
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
			Configuration: map[string]any{},
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

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"server error"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
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
		assert.Contains(t, err.Error(), "failed to fetch usage data")
	})
}

func Test__GetDailyUsageData__OutputChannels(t *testing.T) {
	c := &GetDailyUsageData{}
	channels := c.OutputChannels(nil)

	assert.Len(t, channels, 1)
	assert.Equal(t, core.DefaultOutputChannel, channels[0])
}

func Test__GetDailyUsageData__Configuration(t *testing.T) {
	c := &GetDailyUsageData{}
	fields := c.Configuration()

	assert.Len(t, fields, 2)

	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}

	assert.Contains(t, names, "startDate")
	assert.Contains(t, names, "endDate")
	for _, f := range fields {
		assert.False(t, f.Required)
	}
}
