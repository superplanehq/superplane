package factories

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const testOrderCreateFactoryID = "11111111-1111-1111-1111-111111111111"

// newOrderCreateServer wires up /me, /users, and the create-work-order POST
// endpoint. capture, if non-nil, receives the decoded request body so tests
// can assert on it.
func newOrderCreateServer(t *testing.T, capture *map[string]any) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"me-id","email":"me@example.com","organizationId":"org-1"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"users":[{"metadata":{"id":"user-1","email":"alice@example.com"}},{"metadata":{"id":"user-2","email":"bob@example.com"}}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/factories/"+testOrderCreateFactoryID+"/orders":
			if capture != nil {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				*capture = body
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"order":{"id":"order-1","title":"Ship it","state":"STATE_DRAFT","assignees":[{"id":"me-id","name":"Me"}]}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOrderCreateCommand_TitleRequired(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	err := (&orderCreateCommand{}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--title is required")
}

func TestOrderCreateCommand_DescriptionAndFileMutuallyExclusive(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	title := "Ship it"
	description := "inline"
	file := "some-file.md"
	err := (&orderCreateCommand{title: &title, description: &description, file: &file}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one of --description or --file")
}

func TestOrderCreateCommand_DefaultsAssigneeToCurrentUser(t *testing.T) {
	var body map[string]any
	server := newOrderCreateServer(t, &body)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderCreateFactoryID
	title := "Ship it"
	description := "the description"
	err := (&orderCreateCommand{factory: &factory, title: &title, description: &description}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.Equal(t, "Ship it", body["title"])
	assert.Equal(t, "the description", body["description"])
	assert.Equal(t, []any{"me-id"}, body["assigneeIds"])

	out := stdout.String()
	assert.Contains(t, out, "Work order created: order-1")
	assert.Contains(t, out, "Draft")
}

func TestOrderCreateCommand_ExplicitAssigneesResolveUUIDAndEmail(t *testing.T) {
	var body map[string]any
	server := newOrderCreateServer(t, &body)
	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderCreateFactoryID
	title := "Ship it"
	assignees := []string{"22222222-2222-2222-2222-222222222222", "bob@example.com"}
	err := (&orderCreateCommand{factory: &factory, title: &title, assignees: &assignees}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.ElementsMatch(t, []any{"22222222-2222-2222-2222-222222222222", "user-2"}, body["assigneeIds"])
}

func TestOrderCreateCommand_DescriptionFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "description.md")
	require.NoError(t, os.WriteFile(path, []byte("# Plan\nDetails"), 0o600))

	var body map[string]any
	server := newOrderCreateServer(t, &body)
	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderCreateFactoryID
	title := "Ship it"
	file := path
	err := (&orderCreateCommand{factory: &factory, title: &title, file: &file}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.Equal(t, "# Plan\nDetails", body["description"])
}

func TestOrderCreateCommand_DescriptionFromStdin(t *testing.T) {
	var body map[string]any
	server := newOrderCreateServer(t, &body)
	ctx, _ := cli.NewCommandContext(t, server, "text")
	ctx.Cmd.SetIn(strings.NewReader("from stdin"))

	factory := testOrderCreateFactoryID
	title := "Ship it"
	file := "-"
	err := (&orderCreateCommand{factory: &factory, title: &title, file: &file}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.Equal(t, "from stdin", body["description"])
}

func TestOrderCreateCommand_JSONOutput(t *testing.T) {
	server := newOrderCreateServer(t, nil)
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	factory := testOrderCreateFactoryID
	title := "Ship it"
	err := (&orderCreateCommand{factory: &factory, title: &title}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	assert.Contains(t, out, `"id": "order-1"`)
}
