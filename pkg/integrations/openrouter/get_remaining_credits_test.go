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

func TestGetRemainingCredits_Name(t *testing.T) {
	c := &GetRemainingCredits{}
	require.Equal(t, "openrouter.getRemainingCredits", c.Name())
}

func TestGetRemainingCredits_Label(t *testing.T) {
	c := &GetRemainingCredits{}
	require.Equal(t, "Get Remaining Credits", c.Label())
}

func TestGetRemainingCredits_ExampleOutput(t *testing.T) {
	c := &GetRemainingCredits{}
	out := c.ExampleOutput()
	require.NotNil(t, out)
	require.Equal(t, RemainingCreditsPayloadType, out["type"])
	require.Equal(t, 100.0, out["totalCredits"])
	require.Equal(t, 25.5, out["totalUsage"])
	require.Equal(t, 74.5, out["remaining"])
}

func TestGetRemainingCredits_Execute(t *testing.T) {
	jsonResp := `{"data":{"total_credits":200.0,"total_usage":75.5}}`
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(jsonResp)),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "mgmt-key"},
	}

	ctx := core.ExecutionContext{
		Configuration:  map[string]any{},
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	}

	err := (&GetRemainingCredits{}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, RemainingCreditsPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	wrapped, ok := execState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(RemainingCreditsPayload)
	require.True(t, ok)
	require.Equal(t, 200.0, data.TotalCredits)
	require.Equal(t, 75.5, data.TotalUsage)
	require.Equal(t, 124.5, data.Remaining)
}

func TestGetRemainingCredits_Execute_NoIntegrationKey(t *testing.T) {
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{},
	}
	ctx := core.ExecutionContext{
		ExecutionState: execState,
		HTTP:           &contexts.HTTPContext{},
		Integration:    integrationCtx,
	}

	err := (&GetRemainingCredits{}).Execute(ctx)
	require.Error(t, err)
}

func TestGetRemainingCredits_Execute_APIError(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error":{"message":"Management key required"}}`)),
		}},
	}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "standard-key"},
	}
	ctx := core.ExecutionContext{
		ExecutionState: execState,
		HTTP:           httpCtx,
		Integration:    integrationCtx,
	}

	err := (&GetRemainingCredits{}).Execute(ctx)
	require.Error(t, err)
}
