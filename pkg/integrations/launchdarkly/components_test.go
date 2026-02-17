package launchdarkly

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

func Test__GetFlag__Execute__404HandledAsNotFound(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`{"message":"not found"}`)),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{}

	err := (&GetFlag{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"projectKey": "default",
			"flagKey":    "missing-flag",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiAccessToken": "token-123",
		}},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.Equal(t, GetFlagPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	payload := execState.Payloads[0].(map[string]any)["data"].(GetFlagOutput)
	assert.False(t, payload.Found)
	assert.Contains(t, payload.Message, "not found")
}

func Test__DeleteFlag__Execute__204Success(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			},
		},
	}

	execState := &contexts.ExecutionStateContext{}

	err := (&DeleteFlag{}).Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"projectKey": "default",
			"flagKey":    "my-flag",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{
			"apiAccessToken": "token-123",
		}},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.Equal(t, DeleteFlagPayloadType, execState.Type)
	require.Len(t, execState.Payloads, 1)

	payload := execState.Payloads[0].(map[string]any)["data"].(DeleteFlagOutput)
	assert.True(t, payload.Deleted)
	assert.Equal(t, http.StatusNoContent, payload.StatusCode)
}
