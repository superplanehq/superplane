package factories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestFormatOrderState(t *testing.T) {
	cases := []struct {
		state openapi_client.FactoriesWorkOrderState
		want  string
	}{
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_DRAFT, "Draft"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_OPEN, "Open"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_CLOSED, "Closed"},
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_UNSPECIFIED, "-"},
		{openapi_client.FactoriesWorkOrderState("bogus"), "-"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatOrderState(tc.state))
	}
}

func TestFormatOrderResult(t *testing.T) {
	cases := []struct {
		result openapi_client.FactoriesWorkOrderResult
		want   string
	}{
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_COMPLETED, "Completed"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_REJECTED, "Rejected"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_FAILED, "Failed"},
		{openapi_client.FACTORIESWORKORDERRESULT_RESULT_UNSPECIFIED, "-"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, formatOrderResult(tc.result))
	}
}

func TestFormatRelativeTimeAt(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		value time.Time
		want  string
	}{
		{"zero value", time.Time{}, "-"},
		{"seconds ago", now.Add(-5 * time.Second), "5s ago"},
		{"a minute ago", now.Add(-70 * time.Second), "1m ago"},
		{"minutes ago", now.Add(-10 * time.Minute), "10m ago"},
		{"an hour ago", now.Add(-90 * time.Minute), "1h ago"},
		{"hours ago", now.Add(-5 * time.Hour), "5h ago"},
		{"a day ago", now.Add(-30 * time.Hour), "1d ago"},
		{"days ago", now.Add(-72 * time.Hour), "3d ago"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatRelativeTimeAt(tc.value, now))
		})
	}
}

func TestFormatAssigneeList(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "-", formatAssigneeList(nil))
	})

	t.Run("uses name, falls back to id", func(t *testing.T) {
		named := openapi_client.NewSuperplaneFactoriesUserRef()
		named.SetId("user-1")
		named.SetName("Alice")

		unnamed := openapi_client.NewSuperplaneFactoriesUserRef()
		unnamed.SetId("user-2")

		got := formatAssigneeList([]openapi_client.SuperplaneFactoriesUserRef{*named, *unnamed})
		assert.Equal(t, "Alice, user-2", got)
	})
}

func TestFormatUserRef(t *testing.T) {
	t.Run("name and id", func(t *testing.T) {
		ref := openapi_client.NewSuperplaneFactoriesUserRef()
		ref.SetId("user-1")
		ref.SetName("Alice")
		assert.Equal(t, "Alice (user-1)", formatUserRef(*ref))
	})

	t.Run("id only", func(t *testing.T) {
		ref := openapi_client.NewSuperplaneFactoriesUserRef()
		ref.SetId("user-1")
		assert.Equal(t, "user-1", formatUserRef(*ref))
	})

	t.Run("neither set", func(t *testing.T) {
		ref := openapi_client.NewSuperplaneFactoriesUserRef()
		assert.Equal(t, "-", formatUserRef(*ref))
	})
}

func newTestEvent(eventType string, payload map[string]interface{}) openapi_client.FactoriesWorkOrderEvent {
	event := openapi_client.NewFactoriesWorkOrderEvent()
	event.SetType(eventType)
	event.SetEvent(payload)
	return *event
}

func TestDescribeEvent(t *testing.T) {
	t.Run("status updated with user actor and result", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"user":      map[string]interface{}{"id": "user-1"},
			"fromState": "open",
			"toState":   "closed",
			"toResult":  "completed",
		})
		got := describeEvent(event)
		assert.Equal(t, "status changed: Open -> Closed (result: Completed) by user user-1", got)
	})

	t.Run("status updated with automation actor", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"automation": map[string]interface{}{"nodeName": "deploy-step"},
			"fromState":  "draft",
			"toState":    "open",
		})
		got := describeEvent(event)
		assert.Equal(t, "status changed: Draft -> Open by automation (deploy-step)", got)
	})

	t.Run("assignees updated", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderAssigneesUpdated, map[string]interface{}{
			"user":       map[string]interface{}{"id": "user-1"},
			"assigned":   []interface{}{map[string]interface{}{"id": "user-2"}},
			"unassigned": []interface{}{map[string]interface{}{"id": "user-3"}},
		})
		got := describeEvent(event)
		assert.Equal(t, "assigned user-2; unassigned user-3 by user user-1", got)
	})

	t.Run("comment added by user", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body": "Looks good to me",
			"author": map[string]interface{}{
				"kind":   "user",
				"userId": "user-1",
			},
		})
		got := describeEvent(event)
		assert.Equal(t, "user-1 commented: Looks good to me", got)
	})

	t.Run("comment added by automation", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body": "Build finished",
			"author": map[string]interface{}{
				"kind":       "automation",
				"automation": map[string]interface{}{"appName": "ci-app"},
			},
		})
		got := describeEvent(event)
		assert.Equal(t, "automation (ci-app) commented: Build finished", got)
	})

	t.Run("artifact added", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{"id": "artifact-1", "type": "pr"},
			"user":     map[string]interface{}{"id": "user-1"},
		})
		got := describeEvent(event)
		assert.Equal(t, "added pr artifact artifact-1 by user user-1", got)
	})

	t.Run("step execution created", func(t *testing.T) {
		event := newTestEvent(eventTypeStepExecutionCreated, map[string]interface{}{
			"stepName": "build",
			"line":     map[string]interface{}{"name": "release"},
		})
		got := describeEvent(event)
		assert.Equal(t, `step "build" started (line: release)`, got)
	})

	t.Run("step execution finished", func(t *testing.T) {
		event := newTestEvent(eventTypeStepExecutionFinished, map[string]interface{}{
			"stepName": "build",
		})
		got := describeEvent(event)
		assert.Equal(t, `step "build" finished`, got)
	})

	t.Run("unknown event type falls back to type and raw JSON", func(t *testing.T) {
		event := newTestEvent("order.something.new", map[string]interface{}{"foo": "bar"})
		got := describeEvent(event)
		assert.Contains(t, got, "order.something.new")
		assert.Contains(t, got, `"foo":"bar"`)
	})
}

func TestDecodeCommentEvent(t *testing.T) {
	t.Run("decodes body and author", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body":   "hello",
			"author": map[string]interface{}{"kind": "user", "userId": "user-1"},
		})
		author, body, ok := decodeCommentEvent(event)
		assert.True(t, ok)
		assert.Equal(t, "user-1", author)
		assert.Equal(t, "hello", body)
	})
}
