package render

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

func Test__Render_ListDeploys__Execute(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`[{"cursor":"a","deploy":{"id":"dep-1","status":"live","createdAt":"2026-05-30T12:00:00Z"}}]`,
			)),
		}},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := (&ListDeploys{}).Execute(core.ExecutionContext{
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
		ExecutionState: executionState,
		Configuration:  map[string]any{"service": "srv-123", "statuses": []string{"live"}, "limit": 5},
	})

	require.NoError(t, err)
	assert.Equal(t, ListDeploysPayloadType, executionState.Type)
	data := readMap(readMap(executionState.Payloads[0])["data"])
	assert.Equal(t, "srv-123", data["serviceId"])
	assert.Equal(t, 1, data["count"])
	assert.NotNil(t, data["latestSuccessful"])

	require.Len(t, httpCtx.Requests, 1)
	request := httpCtx.Requests[0]
	assert.Equal(t, http.MethodGet, request.Method)
	assert.Equal(t, "/v1/services/srv-123/deploys", request.URL.Path)
	assert.Equal(t, "5", request.URL.Query().Get("limit"))
	assert.Equal(t, "live", request.URL.Query().Get("status"))
}
