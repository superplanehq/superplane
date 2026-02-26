package openrouter

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetCurrentKeyDetails_Name(t *testing.T) {
	c := &GetCurrentKeyDetails{}
	require.Equal(t, "openrouter.getCurrentKeyDetails", c.Name())
}

func TestGetCurrentKeyDetails_Label(t *testing.T) {
	c := &GetCurrentKeyDetails{}
	require.Equal(t, "Get Current Key Details", c.Label())
}

func TestGetCurrentKeyDetails_ExampleOutput(t *testing.T) {
	c := &GetCurrentKeyDetails{}
	out := c.ExampleOutput()
	require.NotNil(t, out)
	require.Equal(t, CurrentKeyDetailsPayloadType, out["type"])
	require.Equal(t, "My API Key", out["label"])
	require.Equal(t, 100.0, out["limit"])
	require.Equal(t, 25.0, out["usage"])
	require.Equal(t, 75.0, out["limitRemaining"])
	require.Equal(t, false, out["isFreeTier"])
}

func TestGetCurrentKeyDetails_Execute(t *testing.T) {
	jsonResp := `{
		"data":{
			"label":"Test Key",
			"limit":50,
			"usage":10,
			"usage_daily":2,
			"usage_weekly":8,
			"usage_monthly":10,
			"is_free_tier":false,
			"is_management_key":true,
			"limit_remaining":40,
			"limit_reset":"monthly",
			"include_byok_in_limit":true,
			"expires_at":"2026-06-01T00:00:00Z"
		}
	}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "test-key"},
	}

	ctx := core.ExecutionContext{
		Configuration:  map[string]any{},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	}

	err := (&GetCurrentKeyDetails{}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, CurrentKeyDetailsPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped, ok := execState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(CurrentKeyDetailsPayload)
	require.True(t, ok)
	require.Equal(t, "Test Key", data.Label)
	require.NotNil(t, data.Limit)
	require.Equal(t, 50.0, *data.Limit)
	require.Equal(t, 10.0, data.Usage)
	require.Equal(t, 2.0, data.UsageDaily)
	require.Equal(t, 8.0, data.UsageWeekly)
	require.Equal(t, 10.0, data.UsageMonthly)
	require.NotNil(t, data.LimitRemaining)
	require.Equal(t, 40.0, *data.LimitRemaining)
	require.NotNil(t, data.LimitReset)
	require.Equal(t, "monthly", *data.LimitReset)
	require.True(t, data.IsManagementKey)
	require.True(t, data.IncludeByokInLimit)
	require.NotNil(t, data.ExpiresAt)
	require.Equal(t, "2026-06-01T00:00:00Z", *data.ExpiresAt)
}

func TestGetCurrentKeyDetails_Execute_NoIntegrationKey(t *testing.T) {
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{},
	}
	ctx := core.ExecutionContext{
		ExecutionState: execState,
		HTTP:           &contexts.HTTPContext{},
		Integration:    integrationCtx,
	}

	err := (&GetCurrentKeyDetails{}).Execute(ctx)
	require.Error(t, err)
}

func TestGetCurrentKeyDetails_Execute_APIError(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusUnauthorized,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":{"message":"Invalid key"}}`)),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "bad-key"},
	}
	ctx := core.ExecutionContext{
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	}

	err := (&GetCurrentKeyDetails{}).Execute(ctx)
	require.Error(t, err)
}
