package productive

import (
	"net/http"
	"testing"

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

func Test__Client__LatestTaskUpdateChangeset(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[
			{
				"id":"1",
				"type":"activities",
				"attributes":{
					"created_at":"2026-09-25T15:00:00Z",
					"changeset":{"title":["Old","New"]}
				}
			},
			{
				"id":"2",
				"type":"activities",
				"attributes":{
					"created_at":"2026-09-25T16:00:00Z",
					"changeset":{"task_list_id":[10,20]}
				}
			}
		]}`),
	}}

	changeset, err := testClient(t, httpContext).latestTaskUpdateChangeset("91")
	require.NoError(t, err)

	move, ok := taskListMoveFromChangeset(changeset)
	require.True(t, ok)
	assert.Equal(t, TaskListMove{From: "10", To: "20"}, move)

	require.Len(t, httpContext.Requests, 1)
	query := httpContext.Requests[0].URL.Query()
	assert.Equal(t, "91", query.Get("filter[task_id]"))
	assert.Equal(t, "update", query.Get("filter[event]"))
	assert.Equal(t, "task", query.Get("filter[item_type]"))
	assert.Equal(t, "2", query.Get("filter[type]"))
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/activities")
}
