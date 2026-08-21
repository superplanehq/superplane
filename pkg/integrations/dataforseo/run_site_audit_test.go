package dataforseo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunSiteAudit__Setup(t *testing.T) {
	r := &RunSiteAudit{}

	t.Run("empty domain", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"domain":        "   ",
				"maxCrawlPages": 100,
			},
		}
		err := r.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "domain is required")
	})

	t.Run("non-positive maxCrawlPages", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"domain":        "freehire.me",
				"maxCrawlPages": 0,
			},
		}
		err := r.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxCrawlPages")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"domain":        "freehire.me",
				"maxCrawlPages": 100,
			},
		}
		err := r.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__RunSiteAudit__Execute(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			mockResponse(http.StatusOK, `{
				"status_code": 20000,
				"tasks": [{"id": "07131248-1535-0216-1000-17384017ad04", "status_code": 20100}]
			}`),
		},
	}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"domain":        "freehire.me",
			"maxCrawlPages": 100,
		},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "dXNlcjpwYXNz"},
		},
		HTTP:           httpCtx,
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

	// Guard against typos in the request body field names (e.g.
	// "max_crawl_page" missing the trailing "s") that would otherwise only
	// be caught by a real API call.
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://api.dataforseo.com/v3/on_page/task_post", httpCtx.Requests[0].URL.String())

	bodyBytes, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)

	var body []map[string]any
	require.NoError(t, json.Unmarshal(bodyBytes, &body))
	require.Len(t, body, 1)
	assert.Equal(t, "freehire.me", body[0]["target"])
	assert.Equal(t, float64(100), body[0]["max_crawl_pages"])
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

func Test__RunSiteAudit__Poll__GetSummaryError(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 5},
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
				mockResponse(http.StatusInternalServerError, `{"status_code": 50000, "status_message": "Internal error"}`),
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
	assert.Equal(t, 6, metadata.PollAttempt)
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

func Test__RunSiteAudit__Poll__Finished_Truncated(t *testing.T) {
	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 1},
	}
	executionState := &contexts.ExecutionStateContext{}

	// The pages response returns exactly runSiteAuditPagesLimit (1000) items,
	// none with issues, so DataForSEO may still have more pages we never
	// fetched. The result must still be reported "clean" (per current logic,
	// since nothing checked had issues) but flagged as truncated so a canvas
	// consumer knows coverage was partial.
	items := make([]map[string]any, runSiteAuditPagesLimit)
	for i := range items {
		items[i] = map[string]any{
			"url": fmt.Sprintf("https://freehire.me/jobs/%d", i),
			"checks": map[string]any{
				"duplicate_title": false, "broken_links": false, "duplicate_description": false, "is_broken": false,
			},
		}
	}
	pagesBody, err := json.Marshal(map[string]any{
		"status_code": 20000,
		"tasks":       []map[string]any{{"id": "task-1", "result": []map[string]any{{"items": items}}}},
	})
	require.NoError(t, err)

	err = component.HandleHook(core.ActionHookContext{
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
				mockResponse(http.StatusOK, string(pagesBody)),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, RunSiteAuditCleanChannel, executionState.Channel)

	require.Len(t, executionState.Payloads, 1)
	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, data["truncated"])
	assert.Equal(t, 0, data["issuePageCount"])
}

func Test__RunSiteAudit__Poll__Finished_CapsEmittedIssuePages(t *testing.T) {
	original := runSiteAuditIssuePagesEmitCap
	runSiteAuditIssuePagesEmitCap = 2
	defer func() { runSiteAuditIssuePagesEmitCap = original }()

	component := &RunSiteAudit{}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunSiteAuditExecutionMetadata{TaskID: "task-1", PollAttempt: 1},
	}
	executionState := &contexts.ExecutionStateContext{}

	// 4 pages all with issues; with the cap lowered to 2, only 2 should be
	// emitted in "pages", but issuePageCount must reflect the true total (4).
	items := []map[string]any{}
	for i := 0; i < 4; i++ {
		items = append(items, map[string]any{
			"url": fmt.Sprintf("https://freehire.me/jobs/%d", i),
			"checks": map[string]any{
				"duplicate_title": true, "broken_links": false, "duplicate_description": false, "is_broken": false,
			},
		})
	}
	pagesBody, err := json.Marshal(map[string]any{
		"status_code": 20000,
		"tasks":       []map[string]any{{"id": "task-1", "result": []map[string]any{{"items": items}}}},
	})
	require.NoError(t, err)

	err = component.HandleHook(core.ActionHookContext{
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
				mockResponse(http.StatusOK, string(pagesBody)),
			},
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         log.NewEntry(log.New()),
	})

	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, RunSiteAuditIssuesChannel, executionState.Channel)

	require.Len(t, executionState.Payloads, 1)
	wrapped, ok := executionState.Payloads[0].(map[string]any)
	require.True(t, ok)
	data, ok := wrapped["data"].(map[string]any)
	require.True(t, ok)
	pages, ok := data["pages"].([]PageResult)
	require.True(t, ok)
	assert.Len(t, pages, 2) // capped, even though 4 pages have issues
	assert.Equal(t, 4, data["issuePageCount"])
	assert.Equal(t, false, data["truncated"])
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
