package dataforseo

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunSiteAudit__Execute(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"domain":        "freehire.me",
			"maxCrawlPages": 100,
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "07131248-1535-0216-1000-17384017ad04", "status_code": 20100}]
				}`),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)

	metadata, ok := metadataCtx.Metadata.(RunSiteAuditExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, "07131248-1535-0216-1000-17384017ad04", metadata.TaskID)
	assert.Equal(t, "07131248-1535-0216-1000-17384017ad04", executionState.KVs[RunSiteAuditKVTaskID])
	assert.Equal(t, RunSiteAuditPollAction, requestsCtx.Action)
	assert.Equal(t, RunSiteAuditPollInterval, requestsCtx.Duration)
}
