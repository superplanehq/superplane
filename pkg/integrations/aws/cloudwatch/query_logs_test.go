package cloudwatch

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

func validQueryLogsConfig() map[string]any {
	return map[string]any{
		"region":         "us-east-1",
		"logGroups":      []string{"/aws/lambda/my-function"},
		"queryString":    "fields @timestamp, @message | sort @timestamp desc | limit 20",
		"lookbackPeriod": "1h",
		"limit":          100,
	}
}

func credentialSecrets() map[string]core.IntegrationSecret {
	return map[string]core.IntegrationSecret{
		"accessKeyId":     {Name: "accessKeyId", Value: []byte("key")},
		"secretAccessKey": {Name: "secretAccessKey", Value: []byte("secret")},
		"sessionToken":    {Name: "sessionToken", Value: []byte("token")},
	}
}

func TestQueryLogs_Setup(t *testing.T) {
	component := &QueryLogs{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		config := validQueryLogsConfig()
		config["region"] = ""
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing log groups -> error", func(t *testing.T) {
		config := validQueryLogsConfig()
		config["logGroups"] = []string{}
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "at least one log group is required")
	})

	t.Run("missing query string -> error", func(t *testing.T) {
		config := validQueryLogsConfig()
		config["queryString"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "query string is required")
	})

	t.Run("invalid lookback period -> error", func(t *testing.T) {
		config := validQueryLogsConfig()
		config["lookbackPeriod"] = "3h"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "invalid lookbackPeriod")
	})

	t.Run("limit out of range -> error", func(t *testing.T) {
		config := validQueryLogsConfig()
		config["limit"] = 20000
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "limit must be between 1 and 10000")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: validQueryLogsConfig()})
		require.NoError(t, err)
	})
}

func TestQueryLogs_Execute(t *testing.T) {
	component := &QueryLogs{}

	t.Run("invalid configuration -> fails execution", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "failed to decode configuration")
	})

	t.Run("missing credentials -> fails execution", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validQueryLogsConfig(),
			ExecutionState: execState,
			Integration:    &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "credentials")
	})

	t.Run("valid request -> starts query and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"queryId":"query-123"}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validQueryLogsConfig(),
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished, "should keep waiting for the query to complete")
		assert.Equal(t, queryLogsPollHook, requests.Action)
		assert.Equal(t, queryLogsPollInterval, requests.Duration)

		stored, ok := metadata.Metadata.(QueryLogsExecutionMetadata)
		require.True(t, ok)
		assert.Equal(t, "query-123", stored.QueryID)
		assert.Equal(t, []string{"/aws/lambda/my-function"}, stored.LogGroups)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "Logs_20140328.StartQuery", httpContext.Requests[0].Header.Get("X-Amz-Target"))
	})
}

func TestQueryLogs_HandleHook(t *testing.T) {
	component := &QueryLogs{}

	t.Run("unknown hook -> fails execution", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleHook(core.ActionHookContext{
			Name:           "somethingElse",
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "unknown action")
	})

	t.Run("execution already finished -> no-op", func(t *testing.T) {
		execState := &contexts.ExecutionStateContext{Finished: true, KVs: map[string]string{}}
		err := component.HandleHook(core.ActionHookContext{
			Name:           queryLogsPollHook,
			ExecutionState: execState,
		})

		require.NoError(t, err)
	})

	t.Run("status Complete -> emits rows", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"status": "Complete",
						"results": [
							[{"field": "@timestamp", "value": "2026-05-29 08:59:42.123"}, {"field": "@message", "value": "hello world"}]
						],
						"statistics": {"bytesScanned": 100, "recordsMatched": 1, "recordsScanned": 5}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleHook(core.ActionHookContext{
			Name:           queryLogsPollHook,
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata: &contexts.MetadataContext{Metadata: QueryLogsExecutionMetadata{
				QueryID: "query-123",
				Region:  "us-east-1",
			}},
			Integration: &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "Complete", data["status"])

		rows := data["rows"].([]map[string]string)
		require.Len(t, rows, 1)
		assert.Equal(t, "hello world", rows[0]["@message"])

		statistics := data["statistics"].(map[string]any)
		assert.Equal(t, float64(1), statistics["recordsMatched"])
	})

	t.Run("status Running -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status": "Running", "results": []}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{Metadata: QueryLogsExecutionMetadata{QueryID: "query-123", Region: "us-east-1"}}
		requests := &contexts.RequestContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           queryLogsPollHook,
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata:       metadata,
			Requests:       requests,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Equal(t, queryLogsPollHook, requests.Action)

		stored := metadata.Metadata.(QueryLogsExecutionMetadata)
		assert.Equal(t, 1, stored.PollAttempts)
	})

	t.Run("status Failed -> fails execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status": "Failed", "results": []}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleHook(core.ActionHookContext{
			Name:           queryLogsPollHook,
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata:       &contexts.MetadataContext{Metadata: QueryLogsExecutionMetadata{QueryID: "query-123", Region: "us-east-1"}},
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "Failed")
	})

	t.Run("exceeds max poll attempts -> fails execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"status": "Running", "results": []}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.HandleHook(core.ActionHookContext{
			Name:           queryLogsPollHook,
			ExecutionState: execState,
			HTTP:           httpContext,
			Metadata: &contexts.MetadataContext{Metadata: QueryLogsExecutionMetadata{
				QueryID:      "query-123",
				Region:       "us-east-1",
				PollAttempts: maxQueryLogsPollTries - 1,
			}},
			Integration: &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		assert.False(t, execState.Passed)
		assert.Contains(t, execState.FailureMessage, "timed out")
	})
}
