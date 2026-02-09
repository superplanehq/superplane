package cursor

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetDailyUsageData__Execute(t *testing.T) {
	component := &GetDailyUsageData{}

	t.Run("missing admin key -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			ID:            uuid.New(),
			WorkflowID:    uuid.New().String(),
			NodeID:        "n1",
			Configuration: GetDailyUsageSpec{},
			Integration:   &contexts.IntegrationContext{Configuration: map[string]any{}},
			ExecutionState: &contexts.ExecutionStateContext{
				KVs: map[string]string{},
			},
		})

		assert.ErrorContains(t, err, "admin API key required")
	})

	t.Run("success -> emits usage payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"days":[]}`)),
			},
		}}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{
			"adminApiKey": "admin",
		}}

		err := component.Execute(core.ExecutionContext{
			ID:             uuid.New(),
			WorkflowID:     uuid.New().String(),
			NodeID:         "n1",
			HTTP:           httpCtx,
			Configuration:  GetDailyUsageSpec{StartDate: "1d", EndDate: "today"},
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		assert.NoError(t, err)
		assert.True(t, execState.IsFinished())
		assert.Equal(t, DailyUsagePayloadType, execState.Type)
	})
}
