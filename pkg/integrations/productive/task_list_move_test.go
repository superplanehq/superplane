package productive

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

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

	delivered := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
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

	changeset, ok := activityForDelivery(activities, delivered, moveDocument)
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
	changeset, ok = activityForDelivery(activities, delivered.Add(30*time.Second), titleDocument)
	require.True(t, ok)
	_, ok = taskListMoveFromChangeset(changeset)
	assert.False(t, ok)

	_, ok = activityForDelivery(activities, delivered.Add(10*time.Minute), moveDocument)
	assert.False(t, ok)
}

func Test__Client__TaskUpdateChangesetAt(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[
			{
				"id":"2",
				"type":"activities",
				"attributes":{
					"created_at":"2026-09-25T16:00:30Z",
					"changeset":{"title":["Fix payment retries","Renamed"]}
				}
			},
			{
				"id":"1",
				"type":"activities",
				"attributes":{
					"created_at":"2026-09-25T16:00:00Z",
					"changeset":{"task_list_id":[10,20]}
				}
			}
		]}`),
	}}

	delivered := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
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
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/activities")
}
