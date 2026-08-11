package factories

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const testOrderDescribeFactoryID = "11111111-1111-1111-1111-111111111111"
const testOrderDescribeOrderID = "22222222-2222-2222-2222-222222222222"

const describeWorkOrderPayload = `{
  "order": {
    "id": "` + testOrderDescribeOrderID + `",
    "title": "Ship the feature",
    "description": "Line one\nLine two",
    "state": "STATE_OPEN",
    "result": "RESULT_UNSPECIFIED",
    "createdAt": "2025-01-15T10:00:00Z",
    "updatedAt": "2025-01-15T10:05:00Z",
    "assignees": [{"id": "user-1", "name": "Alice"}],
    "createdBy": {"id": "user-2", "name": "Bob"}
  }
}`

// Events are returned newest-first by the API, matching production behavior.
const listWorkOrderEventsPayload = `{
  "events": [
    {
      "timestamp": "2025-01-15T10:10:00Z",
      "type": "order.comment.added",
      "event": {"body": "Looks good", "author": {"kind": "user", "userId": "user-1"}}
    },
    {
      "timestamp": "2025-01-15T10:05:00Z",
      "type": "order.assignees.updated",
      "event": {"assigned": [{"id": "user-1"}], "user": {"id": "user-2"}}
    },
    {
      "timestamp": "2025-01-15T10:00:00Z",
      "type": "order.status.updated",
      "event": {"fromState": "draft", "toState": "open", "user": {"id": "user-2"}}
    }
  ],
  "totalCount": 3,
  "hasNextPage": false
}`

const testOrderDescribeOrgID = "org-1"

// membersPayload stubs /api/v1/users with the two members referenced by
// listWorkOrderEventsPayload's user IDs, so the "describe" command can
// resolve them to emails.
const membersPayload = `{"users":[
  {"metadata":{"id":"user-1","email":"alice@example.com"}},
  {"metadata":{"id":"user-2","email":"bob@example.com"}}
]}`

// newOrderDescribeServer stubs the describe/events endpoints, plus /me and
// /users (needed to resolve event actor IDs to emails), following the same
// pattern as newOrderListServer.
func newOrderDescribeServer(t *testing.T, eventsPayload string) *httptest.Server {
	t.Helper()
	return newOrderDescribeServerWithMembers(t, eventsPayload, http.StatusOK, membersPayload)
}

// newOrderDescribeServerWithMembers is like newOrderDescribeServer but lets
// tests control the /api/v1/users response, to exercise the "members list
// unavailable" degradation path.
func newOrderDescribeServerWithMembers(t *testing.T, eventsPayload string, membersStatus int, membersBody string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"me","email":"me@example.com","organizationId":"` + testOrderDescribeOrgID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(membersStatus)
			_, _ = w.Write([]byte(membersBody))
		case r.Method == http.MethodGet &&
			r.URL.Path == "/api/v1/factories/"+testOrderDescribeFactoryID+"/orders/"+testOrderDescribeOrderID:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(describeWorkOrderPayload))
		case r.Method == http.MethodGet &&
			r.URL.Path == "/api/v1/factories/"+testOrderDescribeFactoryID+"/orders/"+testOrderDescribeOrderID+"/events":
			assert.Equal(t, "200", r.URL.Query().Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(eventsPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOrderDescribeCommand_TextOutput(t *testing.T) {
	server := newOrderDescribeServer(t, listWorkOrderEventsPayload)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, testOrderDescribeOrderID)
	assert.Contains(t, out, "Ship the feature")
	assert.Contains(t, out, "Open")
	assert.Contains(t, out, "Bob (user-2)")

	assert.Contains(t, out, "Assignees:")
	assert.Contains(t, out, "Alice (user-1)")

	assert.Contains(t, out, "Description:")
	assert.Contains(t, out, "Line one")
	assert.Contains(t, out, "Line two")

	assert.Contains(t, out, "Comments:")
	assert.Contains(t, out, "alice@example.com: Looks good")

	assert.Contains(t, out, "Events:")
	assert.Contains(t, out, "order.status.updated")
	assert.Contains(t, out, "order.assignees.updated")
	assert.Contains(t, out, "order.comment.added")
	assert.Contains(t, out, "Work order opened by bob@example.com")
	assert.Contains(t, out, "Work order assigned to alice@example.com by bob@example.com")
	assert.Contains(t, out, "alice@example.com commented: Looks good")

	// The events table must not leak raw user IDs (the "Created By"/
	// "Assignees" fields above are a separate code path fed by the API's
	// already-resolved SuperplaneFactoriesUserRef and are unaffected).
	eventsSection := out[indexOf(out, "Events:"):]
	assert.NotContains(t, eventsSection, "user-1")
	assert.NotContains(t, eventsSection, "user-2")

	// Events must read oldest-first (status -> assignees -> comment).
	statusIdx := indexOf(out, "Work order opened")
	assigneesIdx := indexOf(out, "Work order assigned to")
	commentIdx := indexOf(out, "alice@example.com commented")
	require.True(t, statusIdx >= 0 && assigneesIdx >= 0 && commentIdx >= 0)
	assert.Less(t, statusIdx, assigneesIdx)
	assert.Less(t, assigneesIdx, commentIdx)

	assert.NotContains(t, out, "showing latest")
}

func TestOrderDescribeCommand_CommentsOnlyContainCommentEvents(t *testing.T) {
	server := newOrderDescribeServer(t, listWorkOrderEventsPayload)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	commentsSection := out[indexOf(out, "Comments:"):indexOf(out, "Events:")]
	assert.Contains(t, commentsSection, "Looks good")
	assert.NotContains(t, commentsSection, "Work order opened")
	assert.NotContains(t, commentsSection, "Work order assigned")
}

func TestOrderDescribeCommand_MembersUnavailableDegradesGracefully(t *testing.T) {
	server := newOrderDescribeServerWithMembers(t, listWorkOrderEventsPayload, http.StatusInternalServerError, `{"error":"boom"}`)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "Work order opened by unknown user")
	assert.Contains(t, out, "unknown user commented: Looks good")
}

func TestOrderDescribeCommand_UnknownEventTypeFallsBack(t *testing.T) {
	payload := `{
      "events": [
        {"timestamp": "2025-01-15T10:00:00Z", "type": "order.mystery.happened", "event": {"foo": "bar"}}
      ],
      "totalCount": 1,
      "hasNextPage": false
    }`
	server := newOrderDescribeServer(t, payload)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "order.mystery.happened")
	assert.Contains(t, out, "No comments.")
}

func TestOrderDescribeCommand_TruncationNote(t *testing.T) {
	payload := `{
      "events": [
        {"timestamp": "2025-01-15T10:00:00Z", "type": "order.status.updated", "event": {"fromState": "draft", "toState": "open"}}
      ],
      "totalCount": 500,
      "hasNextPage": true
    }`
	server := newOrderDescribeServer(t, payload)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), "(showing latest 200 of 500 events)")
}

func TestOrderDescribeCommand_OrderIDAlias(t *testing.T) {
	server := newOrderDescribeServer(t, listWorkOrderEventsPayload)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDescribeFactoryID
	// Simulates binding via --order-id instead of --order: same dest pointer.
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Ship the feature")
}

func TestOrderDescribeCommand_MissingOrderErrors(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	factory := testOrderDescribeFactoryID
	orderID := ""
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--order is required")
}

func TestOrderDescribeCommand_JSONOutput(t *testing.T) {
	server := newOrderDescribeServer(t, listWorkOrderEventsPayload)
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	factory := testOrderDescribeFactoryID
	orderID := testOrderDescribeOrderID
	err := (&orderDescribeCommand{factory: &factory, orderID: &orderID}).Execute(ctx)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Contains(t, result, "order")
	assert.Contains(t, result, "comments")
	assert.Contains(t, result, "events")
	assert.Contains(t, result, "eventsTruncated")

	comments, ok := result["comments"].([]interface{})
	require.True(t, ok)
	assert.Len(t, comments, 1)

	events, ok := result["events"].([]interface{})
	require.True(t, ok)
	assert.Len(t, events, 3)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
