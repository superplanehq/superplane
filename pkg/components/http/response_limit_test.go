package http_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	httpcomponent "github.com/superplanehq/superplane/pkg/components/http"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestHTTPRetryEmitsFailureForOversizedResponse(t *testing.T) {
	t.Setenv("SUPERPLANE_MAX_PAYLOAD_SIZE", "1024")
	for _, tc := range []struct {
		name   string
		hook   bool
		status int
	}{
		{"first attempt", false, http.StatusOK},
		{"successful retry", true, http.StatusOK},
		{"exhausted retries", true, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			db := database.DB(t.Context())
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
			root := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
			execution := support.CreateCanvasNodeExecution(t, canvas.ID, "request", root.ID, root.ID)
			ctx := core.ExecutionContext{
				Logger: log.NewEntry(log.StandardLogger()),
				Configuration: map[string]any{
					"method": "GET", "url": "https://api.example.com",
					"retry": httpcomponent.RetrySpec{Enabled: true, Strategy: "fixed", MaxAttempts: 1, IntervalSeconds: 5},
				},
				HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
					StatusCode: tc.status, Header: http.Header{"Content-Type": {"application/json"}},
					Body: io.NopCloser(strings.NewReader(`{"extra":"` + strings.Repeat("x", 2048) + `"}`)),
				}}},
				Metadata: &contexts.MetadataContext{Metadata: httpcomponent.Metadata{
					Retry: &httpcomponent.RetryMetadata{Attempts: 1, MaxAttempts: 1},
				}},
				ExecutionState: workercontexts.NewExecutionStateContext(db, execution, nil),
			}
			action := &httpcomponent.HTTP{}
			if tc.hook {
				require.NoError(t, action.HandleHook(core.ActionHookContext{
					Name: "retryRequest", Logger: ctx.Logger, Configuration: ctx.Configuration,
					HTTP: ctx.HTTP, Metadata: ctx.Metadata, ExecutionState: ctx.ExecutionState,
				}))
			} else {
				require.NoError(t, action.Execute(ctx))
			}
			var outputs []models.CanvasEvent
			require.NoError(t, db.Where("execution_id = ?", execution.ID).Find(&outputs).Error)
			require.Len(t, outputs, 1)
			assert.Equal(t, httpcomponent.FailureOutputChannel, outputs[0].Channel)
			payload := outputs[0].Data.Data().(map[string]any)["data"].(map[string]any)
			assert.Equal(t, json.Number(strconv.Itoa(tc.status)), payload["status"])
			assert.Contains(t, payload["error"], core.ErrEventPayloadTooLarge.Error())
			assert.Nil(t, payload["body"])
			assert.Nil(t, payload["headers"])
		})
	}
}
