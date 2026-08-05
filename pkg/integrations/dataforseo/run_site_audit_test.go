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

func Test__RunSiteAudit__Poll__InProgress(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 3},
	}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: RunSiteAuditPollAction,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "result": [{"crawl_progress": "in_progress"}]}]
				}`),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, RunSiteAuditPollAction, requestsCtx.Action)
	assert.Equal(t, RunSiteAuditPollInterval, requestsCtx.Duration)

	metadata, ok := metadataCtx.Metadata.(RunSiteAuditExecutionMetadata)
	require.True(t, ok)
	assert.Equal(t, 4, metadata.PollAttempt)
}

func Test__RunSiteAudit__Poll__AttemptCapExceeded(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: RunSiteAuditMaxPollAttempts},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name:           RunSiteAuditPollAction,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	assert.True(t, executionState.Finished)
	assert.False(t, executionState.Passed)
	assert.Contains(t, executionState.FailureMessage, "72")
}

func Test__RunSiteAudit__Poll__Finished_IssuesFound(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 1},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: RunSiteAuditPollAction,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "result": [{"crawl_progress": "finished"}]}]
				}`),
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "result": [{"items": [
						{"url": "https://freehire.me/jobs/1", "checks": {"duplicate_title": true, "broken_links": false, "duplicate_description": false, "is_broken": false}},
						{"url": "https://freehire.me/jobs/2", "checks": {"duplicate_title": false, "broken_links": false, "duplicate_description": false, "is_broken": false}}
					]}]}]
				}`),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, RunSiteAuditIssuesChannel, executionState.Channel)

	// The mock ExecutionStateContext wraps each raw payload as
	// {"type": ..., "timestamp": ..., "data": <the map Emit was given>} —
	// see test/support/contexts/*.go:280-289. Unwrap through "data" to reach it.
	require.Len(t, executionState.Payloads, 1)
	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	pages, ok := data["pages"].([]PageResult)
	require.True(t, ok)
	require.Len(t, pages, 1) // only the page with an actual issue is included
	assert.Equal(t, "https://freehire.me/jobs/1", pages[0].URL)
	assert.True(t, pages[0].Checks.DuplicateTitle)
}

func Test__RunSiteAudit__Poll__Finished_Clean(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 1},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.HandleHook(core.ActionHookContext{
		Name: RunSiteAuditPollAction,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		},
		HTTP: &contexts.HTTPContext{
			Responses: []*http.Response{
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "result": [{"crawl_progress": "finished"}]}]
				}`),
				mockResponse(http.StatusOK, `{
					"status_code": 20000,
					"tasks": [{"id": "task-1", "result": [{"items": [
						{"url": "https://freehire.me/jobs/1", "checks": {"duplicate_title": false, "broken_links": false, "duplicate_description": false, "is_broken": false}}
					]}]}]
				}`),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, RunSiteAuditCleanChannel, executionState.Channel)
}
