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

const testOrderAssignFactoryID = "11111111-1111-1111-1111-111111111111"
const testOrderAssignOrderID = "order-1"

func newOrderAssignServer(t *testing.T, capture *map[string]any) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"me-id","email":"me@example.com","organizationId":"org-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"users":[{"metadata":{"id":"user-1","email":"alice@example.com"}}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/factories/"+testOrderAssignFactoryID+"/orders/"+testOrderAssignOrderID+"/assignees":
			if capture != nil {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				*capture = body
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"order":{"id":"order-1","title":"Ship it","state":"STATE_OPEN","assignees":[{"id":"user-1","name":"Alice"}]}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOrderAssignCommand_OrderRequired(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	assignees := []string{"alice@example.com"}
	err := (&orderAssignCommand{assignees: &assignees}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--order is required")
}

func TestOrderAssignCommand_AssigneeRequired(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	orderID := testOrderAssignOrderID
	err := (&orderAssignCommand{orderID: &orderID}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--assignee is required")
	assert.Contains(t, err.Error(), "replaces the full assignee list")
}

func TestOrderAssignCommand_BlankAssigneesTreatedAsMissing(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	orderID := testOrderAssignOrderID
	assignees := []string{"  ", ""}
	err := (&orderAssignCommand{orderID: &orderID, assignees: &assignees}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--assignee is required")
}

func TestOrderAssignCommand_Success(t *testing.T) {
	var body map[string]any
	server := newOrderAssignServer(t, &body)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderAssignFactoryID
	orderID := testOrderAssignOrderID
	assignees := []string{"22222222-2222-2222-2222-222222222222", "alice@example.com"}
	err := (&orderAssignCommand{factory: &factory, orderID: &orderID, assignees: &assignees}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.ElementsMatch(t, []any{"22222222-2222-2222-2222-222222222222", "user-1"}, body["assigneeIds"])

	out := stdout.String()
	assert.Contains(t, out, "Work order assignees updated: order-1")
	assert.Contains(t, out, "Alice")
}

func TestOrderAssignCommand_JSONOutput(t *testing.T) {
	server := newOrderAssignServer(t, nil)
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	factory := testOrderAssignFactoryID
	orderID := testOrderAssignOrderID
	assignees := []string{"alice@example.com"}
	err := (&orderAssignCommand{factory: &factory, orderID: &orderID, assignees: &assignees}).Execute(ctx)
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), `"id": "order-1"`)
}

func TestOrderAssignCommand_UnknownAssigneeEmail(t *testing.T) {
	server := newOrderAssignServer(t, nil)
	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderAssignFactoryID
	orderID := testOrderAssignOrderID
	assignees := []string{"nobody@example.com"}
	err := (&orderAssignCommand{factory: &factory, orderID: &orderID, assignees: &assignees}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody@example.com")
}
