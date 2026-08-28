package factories

import (
	"strings"
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
		{openapi_client.FACTORIESWORKORDERSTATE_STATE_INTAKE, "Intake"},
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

// testMemberLookup is a fixture memberEmailLookup shared by the describeEvent
// test cases below.
var testMemberLookup = memberEmailLookup{
	emailByID: map[string]string{
		"user-1": "alice@example.com",
		"user-2": "bob@example.com",
		"user-3": "carol@example.com",
	},
}

func TestMemberEmailLookup(t *testing.T) {
	t.Run("known id resolves to email", func(t *testing.T) {
		assert.Equal(t, "alice@example.com", testMemberLookup.actorLabel("user-1"))
	})

	t.Run("id with empty email falls back to display name", func(t *testing.T) {
		users := []openapi_client.SuperplaneUsersUser{
			{
				Metadata: &openapi_client.UsersUserMetadata{Id: openapi_client.PtrString("user-4")},
				Spec:     &openapi_client.UsersUserSpec{DisplayName: openapi_client.PtrString("Dana")},
			},
		}
		lookup := newMemberEmailLookup(users)
		assert.Equal(t, "Dana", lookup.actorLabel("user-4"))
	})

	t.Run("unknown id falls back to unknown user", func(t *testing.T) {
		assert.Equal(t, "unknown user", testMemberLookup.actorLabel("user-999"))
	})

	t.Run("blank id falls back to unknown user", func(t *testing.T) {
		assert.Equal(t, "unknown user", testMemberLookup.actorLabel(""))
	})
}

func TestDescribeEvent(t *testing.T) {
	t.Run("status updated: created (no fromState)", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"user":      map[string]interface{}{"id": "user-1"},
			"fromState": "",
			"toState":   "draft",
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "Work order created by alice@example.com", got)
	})

	t.Run("status updated: closed with result", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"user":      map[string]interface{}{"id": "user-2"},
			"fromState": "open",
			"toState":   "closed",
			"toResult":  "completed",
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "Work order closed as Completed by bob@example.com", got)
	})

	t.Run("status updated: draft to open", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"automation": map[string]interface{}{"nodeName": "deploy-step"},
			"fromState":  "draft",
			"toState":    "open",
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "Work order opened by automation (deploy-step)", got)
	})

	t.Run("status updated: unresolved user id falls back to unknown user", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderStatusUpdated, map[string]interface{}{
			"user":      map[string]interface{}{"id": "some-uuid-not-in-org"},
			"fromState": "",
			"toState":   "draft",
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "Work order created by unknown user", got)
		assert.NotContains(t, got, "some-uuid-not-in-org")
	})

	t.Run("assignees updated", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderAssigneesUpdated, map[string]interface{}{
			"user":       map[string]interface{}{"id": "user-1"},
			"assigned":   []interface{}{map[string]interface{}{"id": "user-2"}},
			"unassigned": []interface{}{map[string]interface{}{"id": "user-3"}},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "Work order assigned to bob@example.com; unassigned carol@example.com by alice@example.com", got)
	})

	t.Run("comment added by user", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body": "Looks good to me",
			"author": map[string]interface{}{
				"kind":   "user",
				"userId": "user-1",
			},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "alice@example.com commented: Looks good to me", got)
	})

	t.Run("comment added by automation", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body": "Build finished",
			"author": map[string]interface{}{
				"kind":       "automation",
				"automation": map[string]interface{}{"appName": "ci-app"},
			},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "automation (ci-app) commented: Build finished", got)
	})

	t.Run("artifact added with title", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{
				"id":   "artifact-1",
				"type": "markdown",
				"data": map[string]interface{}{"title": "PLAN.md"},
			},
			"user": map[string]interface{}{"id": "user-1"},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "markdown added: PLAN.md", got)
		assert.NotContains(t, got, "artifact-1")
		assert.NotContains(t, got, "alice@example.com")
	})

	t.Run("artifact added with name", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{
				"id":   "artifact-2",
				"type": "branch",
				"data": map[string]interface{}{"name": "fix/render-issue"},
			},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "branch added: fix/render-issue", got)
	})

	t.Run("artifact added with url only", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{
				"id":   "artifact-3",
				"type": "pr",
				"data": map[string]interface{}{"url": "https://github.com/example/repo/pull/1"},
			},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "PR added: https://github.com/example/repo/pull/1", got)
	})

	t.Run("artifact added with no data falls back to type only", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{"id": "artifact-4", "type": "pr"},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "PR added", got)
	})

	t.Run("artifact added with long title keeps 'added' next to the type, not trailing", func(t *testing.T) {
		// Regression test: a long artifact label (e.g. a PR title) can push a
		// terminal line past its width. The "<type> added" phrase must come
		// first so it can never be wrapped onto its own line, orphaned from
		// the rest of the message.
		longTitle := "feat(factory): add findWorkOrder component and explicit orderId targeting"
		event := newTestEvent(eventTypeOrderArtifactAdded, map[string]interface{}{
			"artifact": map[string]interface{}{
				"id":   "artifact-5",
				"type": "pr",
				"data": map[string]interface{}{"title": longTitle},
			},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, "PR added: "+longTitle, got)
		assert.True(t, strings.HasPrefix(got, "PR added:"), "expected message to start with the fixed phrase, got %q", got)
	})

	t.Run("step execution created", func(t *testing.T) {
		event := newTestEvent(eventTypeStepExecutionCreated, map[string]interface{}{
			"stepName": "build",
			"line":     map[string]interface{}{"name": "release"},
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, `step "build" started (line: release)`, got)
	})

	t.Run("step execution finished", func(t *testing.T) {
		event := newTestEvent(eventTypeStepExecutionFinished, map[string]interface{}{
			"stepName": "build",
		})
		got := describeEvent(event, testMemberLookup)
		assert.Equal(t, `step "build" finished`, got)
	})

	t.Run("unknown event type falls back to type and raw JSON", func(t *testing.T) {
		event := newTestEvent("order.something.new", map[string]interface{}{"foo": "bar"})
		got := describeEvent(event, testMemberLookup)
		assert.Contains(t, got, "order.something.new")
		assert.Contains(t, got, `"foo":"bar"`)
	})
}

func TestDecodeCommentEvent(t *testing.T) {
	t.Run("decodes body and resolved author", func(t *testing.T) {
		event := newTestEvent(eventTypeOrderCommentAdded, map[string]interface{}{
			"body":   "hello",
			"author": map[string]interface{}{"kind": "user", "userId": "user-1"},
		})
		author, body, ok := decodeCommentEvent(event, testMemberLookup)
		assert.True(t, ok)
		assert.Equal(t, "alice@example.com", author)
		assert.Equal(t, "hello", body)
	})
}
