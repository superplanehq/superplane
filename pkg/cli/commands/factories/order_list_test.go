package factories

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const testOrderListFactoryID = "11111111-1111-1111-1111-111111111111"
const testOrderListOrgID = "org-1"

const workOrdersPayload = `{
  "orders": [
    {
      "id": "order-1",
      "title": "Ship the feature",
      "state": "STATE_OPEN",
      "result": "RESULT_UNSPECIFIED",
      "createdAt": "2025-01-15T10:00:00Z",
      "assignees": [{"id": "user-1", "name": "Alice"}],
      "executions": [{"id": "exec-1"}, {"id": "exec-2"}]
    }
  ]
}`

func newOrderListServer(t *testing.T, assertQuery func(t *testing.T, query map[string][]string)) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"me","email":"me@example.com","organizationId":"` + testOrderListOrgID + `"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"users":[{"metadata":{"id":"user-1","email":"alice@example.com"}}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/factories/"+testOrderListFactoryID+"/orders":
			if assertQuery != nil {
				assertQuery(t, map[string][]string(r.URL.Query()))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(workOrdersPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOrderListCommand_TextOutput(t *testing.T) {
	server := newOrderListServer(t, nil)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	err := (&orderListCommand{factory: &factory}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "TITLE")
	assert.Contains(t, out, "STATE")
	assert.Contains(t, out, "RESULT")
	assert.Contains(t, out, "ASSIGNEES")
	assert.Contains(t, out, "EXECUTIONS")
	assert.Contains(t, out, "CREATED")
	assert.Contains(t, out, "order-1")
	assert.Contains(t, out, "Ship the feature")
	assert.Contains(t, out, "Open")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "2")
}

func TestOrderListCommand_JSONOutput(t *testing.T) {
	server := newOrderListServer(t, nil)
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	factory := testOrderListFactoryID
	err := (&orderListCommand{factory: &factory}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, `"id": "order-1"`)
	assert.Contains(t, out, `"title": "Ship the feature"`)
}

func TestOrderListCommand_EmptyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"orders":[]}`))
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	factory := testOrderListFactoryID
	err := (&orderListCommand{factory: &factory}).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, "No work orders found.\n", stdout.String())
}

func TestOrderListCommand_Filters(t *testing.T) {
	var seenQuery map[string][]string
	server := newOrderListServer(t, func(t *testing.T, query map[string][]string) {
		seenQuery = query
	})

	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	assignees := []string{"alice@example.com", "22222222-2222-2222-2222-222222222222"}
	states := []string{"open", "STATE_CLOSED"}
	results := []string{"completed"}
	unassigned := true

	err := (&orderListCommand{
		factory:    &factory,
		assignees:  &assignees,
		states:     &states,
		results:    &results,
		unassigned: &unassigned,
	}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, seenQuery)
	assert.ElementsMatch(t, []string{"user-1", "22222222-2222-2222-2222-222222222222"}, seenQuery["assigneeIds"])
	assert.ElementsMatch(t, []string{"STATE_OPEN", "STATE_CLOSED"}, seenQuery["states"])
	assert.ElementsMatch(t, []string{"RESULT_COMPLETED"}, seenQuery["results"])
	assert.Equal(t, []string{"true"}, seenQuery["unassigned"])
}

func TestOrderListCommand_DefaultsToOpenState(t *testing.T) {
	var seenQuery map[string][]string
	server := newOrderListServer(t, func(t *testing.T, query map[string][]string) {
		seenQuery = query
	})

	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	err := (&orderListCommand{factory: &factory}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, seenQuery)
	assert.Equal(t, []string{"STATE_OPEN"}, seenQuery["states"])
}

func TestOrderListCommand_StateAllDisablesDefaultFilter(t *testing.T) {
	var seenQuery map[string][]string
	server := newOrderListServer(t, func(t *testing.T, query map[string][]string) {
		seenQuery = query
	})

	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	states := []string{"all"}
	err := (&orderListCommand{factory: &factory, states: &states}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, seenQuery)
	assert.Empty(t, seenQuery["states"])
}

func TestOrderListCommand_StateAllCaseInsensitive(t *testing.T) {
	var seenQuery map[string][]string
	server := newOrderListServer(t, func(t *testing.T, query map[string][]string) {
		seenQuery = query
	})

	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	states := []string{" All "}
	err := (&orderListCommand{factory: &factory, states: &states}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, seenQuery)
	assert.Empty(t, seenQuery["states"])
}

func TestOrderListCommand_UnknownAssigneeEmail(t *testing.T) {
	server := newOrderListServer(t, nil)
	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderListFactoryID
	assignees := []string{"nobody@example.com"}
	err := (&orderListCommand{factory: &factory, assignees: &assignees}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody@example.com")
	assert.Contains(t, err.Error(), "not found")
}

func TestNormalizeFilterValues(t *testing.T) {
	got := normalizeFilterValues([]string{"open", "STATE_CLOSED", "Draft"}, shortOrderStateTokens)
	assert.Equal(t, []string{"STATE_OPEN", "STATE_CLOSED", "STATE_DRAFT"}, got)

	got = normalizeFilterValues([]string{"completed", "RESULT_FAILED", "unknown-token"}, shortOrderResultTokens)
	assert.Equal(t, []string{"RESULT_COMPLETED", "RESULT_FAILED", "unknown-token"}, got)
}

func TestResolveStateFilter(t *testing.T) {
	assert.Equal(t, []string{"open"}, resolveStateFilter(nil))
	assert.Equal(t, []string{"open"}, resolveStateFilter([]string{}))
	assert.Equal(t, []string{"closed"}, resolveStateFilter([]string{"closed"}))
	assert.Nil(t, resolveStateFilter([]string{"all"}))
	assert.Nil(t, resolveStateFilter([]string{"ALL"}))
	assert.Nil(t, resolveStateFilter([]string{" all "}))
	assert.Nil(t, resolveStateFilter([]string{"open", "all"}))
}
