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
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/activities")
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
