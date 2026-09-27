package productive

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func testClock() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

type activityRecord struct {
	id        string
	at        time.Time
	changeset map[string]any
}

func activitiesResponse(records []activityRecord) *http.Response {
	data := make([]map[string]any, 0, len(records))
	for _, record := range records {
		data = append(data, map[string]any{
			"id":   record.id,
			"type": "activities",
			"attributes": map[string]any{
				"event":      "update",
				"created_at": record.at.Format(time.RFC3339Nano),
				"changeset":  record.changeset,
			},
		})
	}
	body, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		panic(err)
	}
	return jsonResponse(string(body))
}

func TestTaskListMoveFromChangeset(t *testing.T) {
	t.Parallel()

	move, ok := taskListMoveFromChangeset(map[string]any{
		"task_list_id": []any{float64(10), float64(20)},
	})
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)

	move, ok = taskListMoveFromChangeset(map[string]any{
		"milestone_id": map[string]any{"old": "features", "new": "bugs"},
	})
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "features", To: "bugs"}, move)

	_, ok = taskListMoveFromChangeset(map[string]any{
		"title": []any{"Old", "New"},
	})
	assert.False(t, ok)

	_, ok = taskListMoveFromChangeset(map[string]any{
		"task_list_id": []any{float64(20), float64(20)},
	})
	assert.False(t, ok)

	move, ok = taskListMoveFromChangeset([]any{
		map[string]any{"attribute": "task_list_id", "from": "10", "to": "20"},
	})
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)
}

func TestActivityForDelivery_IgnoresALaterUpdate(t *testing.T) {
	t.Parallel()

	delivered := testClock()
	moveDocument := map[string]any{
		"attributes": map[string]any{"title": "Fix payment retries"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}
	activities := []taskActivity{
		{
			at:        delivered.Add(30 * time.Second),
			changeset: map[string]any{"title": []any{"Fix payment retries", "Renamed"}},
		},
		{
			at:        delivered,
			changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}},
		},
	}

	changeset, _, ok := activityForDelivery(activities, delivered, moveDocument)
	require.True(t, ok)
	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)

	titleDocument := map[string]any{
		"attributes": map[string]any{"title": "Renamed"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}
	changeset, _, ok = activityForDelivery(activities, delivered.Add(30*time.Second), titleDocument)
	require.True(t, ok)
	_, ok = taskListMoveFromChangeset(changeset)
	assert.False(t, ok)

	_, _, ok = activityForDelivery(activities, delivered.Add(10*time.Minute), moveDocument)
	assert.False(t, ok)
}

func Test__Client__TaskUpdateChangesetAt(t *testing.T) {
	delivered := testClock()
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		activitiesResponse([]activityRecord{
			{id: "2", at: delivered.Add(30 * time.Second), changeset: map[string]any{"title": []any{"Fix payment retries", "Renamed"}}},
			{id: "1", at: delivered, changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}}},
		}),
	}}

	document := map[string]any{
		"attributes": map[string]any{"title": "Fix payment retries"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}
	changeset, found, err := testClient(t, httpContext).taskUpdateChangesetAt("91", delivered, document)
	require.NoError(t, err)
	require.True(t, found)

	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)

	require.Len(t, httpContext.Requests, 1)
	query := httpContext.Requests[0].URL.Query()
	assert.Equal(t, "91", query.Get("filter[task_id]"))
	assert.Equal(t, "update", query.Get("filter[event]"))
	assert.Equal(t, "task", query.Get("filter[item_type]"))
	assert.Equal(t, "2", query.Get("filter[type]"))
	assert.Equal(t, delivered.Add(-taskActivityMatchWindow).Format(time.RFC3339Nano), query.Get("filter[after]"))
	assert.Equal(t, delivered.Add(taskActivityMatchWindow).Format(time.RFC3339Nano), query.Get("filter[before]"))
	assert.Equal(t, "1", query.Get("page[number]"))
	assert.Equal(t, strconv.Itoa(taskActivityPageSize), query.Get("page[size]"))
	assert.False(t, query.Has("sort"), "Productive.io rejects a sort on activities")
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/activities")
}

// The activity and task list bodies are Productive.io responses for a task
// moved from Factory to Bugs.
func Test__Client__TaskUpdateChangesetAt_ReadsProductiveTaskListLabels(t *testing.T) {
	delivered, err := time.Parse(time.RFC3339Nano, "2026-09-27T19:29:02.000+02:00")
	require.NoError(t, err)
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[
			{"id":"312030744","type":"activities","attributes":{"event":"update","created_at":"2026-09-27T19:29:01.515+02:00",
				"changeset":[{"task_list":[{"value":"Folder: Factory"},{"value":"Folder: Bugs"}]}]}},
			{"id":"312030743","type":"activities","attributes":{"event":"update","created_at":"2026-09-27T19:28:58.610+02:00",
				"changeset":[{"title":[{"value":"test"},{"value":"test 1 1 11"}]}]}}
		]}`),
		jsonResponse(`{"data":[
			{"id":"2897278","type":"task_lists","attributes":{"name":"Bugs"},"relationships":{"folder":{"data":{"type":"folders","id":"1238177"}}}},
			{"id":"2897297","type":"task_lists","attributes":{"name":"Factory"},"relationships":{"folder":{"data":{"type":"folders","id":"1238177"}}}}
		],"included":[{"id":"1238177","type":"folders","attributes":{"name":"Folder"}}]}`),
	}}
	document := map[string]any{
		"attributes": map[string]any{"title": "test 1 1 11"},
		"relationships": map[string]any{
			"project":   map[string]any{"data": map[string]any{"id": "1052266"}},
			"task_list": map[string]any{"data": map[string]any{"id": "2897278"}},
		},
	}

	changeset, found, err := testClient(t, httpContext).taskUpdateChangesetAt("20373726", delivered, document)
	require.NoError(t, err)
	require.True(t, found)
	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "2897297", To: "2897278"}, move)

	require.Len(t, httpContext.Requests, 2)
	listsQuery := httpContext.Requests[1].URL.Query()
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/task_lists")
	assert.Equal(t, "1052266", listsQuery.Get("filter[project_id]"))
	assert.Equal(t, "folder", listsQuery.Get("include"))
}

func TestChangesetValue_ReadsWrappedValues(t *testing.T) {
	t.Parallel()

	from, to, ok := changePair([]any{map[string]any{"value": "test"}, map[string]any{"value": "test 1 1 11"}})
	require.True(t, ok)
	assert.Equal(t, "test", from)
	assert.Equal(t, "test 1 1 11", to)

	from, to, ok = changePair([]any{nil, map[string]any{"value": "Folder: Factory"}})
	require.True(t, ok)
	assert.Equal(t, "", from)
	assert.Equal(t, "Folder: Factory", to)
}

func Test__Client__TaskUpdateChangesetAt_ReadsThePageThatHoldsThisDelivery(t *testing.T) {
	delivered := testClock()
	firstPage := make([]activityRecord, taskActivityPageSize)
	for i := range firstPage {
		firstPage[i] = activityRecord{
			id:        strconv.Itoa(i + 1),
			at:        delivered.Add(time.Duration(i+1) * time.Second),
			changeset: map[string]any{"title": []any{"Old", "Other"}},
		}
	}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		activitiesResponse(firstPage),
		activitiesResponse([]activityRecord{{
			id:        "move",
			at:        delivered,
			changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}},
		}}),
	}}
	document := map[string]any{
		"attributes": map[string]any{"title": "Fix payment retries"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}

	changeset, found, err := testClient(t, httpContext).taskUpdateChangesetAt("91", delivered, document)
	require.NoError(t, err)
	require.True(t, found)
	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)

	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, "1", httpContext.Requests[0].URL.Query().Get("page[number]"))
	assert.Equal(t, "2", httpContext.Requests[1].URL.Query().Get("page[number]"))
}

func Test__Client__TaskUpdateChangesetAt_StopsWhenThisDeliveryIsOnTheFirstPage(t *testing.T) {
	delivered := testClock()
	firstPage := make([]activityRecord, taskActivityPageSize)
	for i := range firstPage {
		firstPage[i] = activityRecord{
			id:        strconv.Itoa(i + 1),
			at:        delivered.Add(time.Duration(i+1) * time.Second),
			changeset: map[string]any{"title": []any{"Old", "Other"}},
		}
	}
	firstPage[0] = activityRecord{
		id:        "move",
		at:        delivered,
		changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}},
	}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{activitiesResponse(firstPage)}}
	document := map[string]any{
		"attributes": map[string]any{"title": "Fix payment retries"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}

	changeset, found, err := testClient(t, httpContext).taskUpdateChangesetAt("91", delivered, document)
	require.NoError(t, err)
	require.True(t, found)
	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)
	require.Len(t, httpContext.Requests, 1)
}

func Test__Client__TaskUpdateChangesetAt_KeepsReadingForACloserActivity(t *testing.T) {
	delivered := testClock()
	firstPage := make([]activityRecord, taskActivityPageSize)
	for i := range firstPage {
		firstPage[i] = activityRecord{
			id:        strconv.Itoa(i + 1),
			at:        delivered.Add(time.Duration(i+1) * time.Second),
			changeset: map[string]any{"description": []any{"Old", "New"}},
		}
	}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		activitiesResponse(firstPage),
		activitiesResponse([]activityRecord{{
			id:        "move",
			at:        delivered,
			changeset: map[string]any{"task_list_id": []any{float64(10), float64(20)}},
		}}),
	}}
	document := map[string]any{
		"attributes": map[string]any{"title": "Fix payment retries"},
		"relationships": map[string]any{
			"task_list": map[string]any{"data": map[string]any{"id": "20"}},
		},
	}

	changeset, found, err := testClient(t, httpContext).taskUpdateChangesetAt("91", delivered, document)
	require.NoError(t, err)
	require.True(t, found)
	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)
	require.Len(t, httpContext.Requests, 2)
}
