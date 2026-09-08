package notion

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func testIntegration(overrides map[string]any) *contexts.IntegrationContext {
	config := map[string]any{"apiToken": "secret_token"}
	for k, v := range overrides {
		config[k] = v
	}
	return &contexts.IntegrationContext{Configuration: config}
}

func testClient(t *testing.T, http *contexts.HTTPContext) *Client {
	t.Helper()
	client, err := NewClient(http, testIntegration(nil))
	require.NoError(t, err)
	return client
}

func Test__NewClient(t *testing.T) {
	t.Run("missing apiToken -> error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, testIntegration(map[string]any{"apiToken": ""}))
		require.ErrorContains(t, err, "missing Notion internal integration token")
	})

	t.Run("uses the Notion API base URL", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, testIntegration(nil))
		require.NoError(t, err)
		assert.Equal(t, BaseURL, client.BaseURL)
	})
}

func Test__Client__ListDatabases(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[
			{"id":"db-1","object":"database","title":[{"plain_text":"Tasks"}]},
			{"id":"db-2","object":"database","title":[{"plain_text":"Bugs"}]}
		],"has_more":false}`),
	}}

	databases, err := testClient(t, httpContext).ListDatabases()
	require.NoError(t, err)
	require.Len(t, databases, 2)
	assert.Equal(t, Database{ID: "db-1", Name: "Tasks"}, databases[0])
	assert.Equal(t, Database{ID: "db-2", Name: "Bugs"}, databases[1])

	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/search")
	assert.Equal(t, "Bearer secret_token", httpContext.Requests[0].Header.Get("Authorization"))
	assert.Equal(t, APIVersion, httpContext.Requests[0].Header.Get("Notion-Version"))
}

func Test__Client__ListDatabases__WalksPagination(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[{"id":"db-1","object":"database","title":[{"plain_text":"Tasks"}]}],"has_more":true,"next_cursor":"cursor-1"}`),
		jsonResponse(`{"results":[{"id":"db-2","object":"database","title":[{"plain_text":"Bugs"}]}],"has_more":false}`),
	}}

	databases, err := testClient(t, httpContext).ListDatabases()
	require.NoError(t, err)
	require.Len(t, databases, 2)
	assert.Len(t, httpContext.Requests, 2)
}

func Test__Client__GetDatabase(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{"id":"db-1","object":"database","title":[{"plain_text":"Tasks"}]}`),
		}}

		database, err := testClient(t, httpContext).GetDatabase("db-1")
		require.NoError(t, err)
		assert.Equal(t, &Database{ID: "db-1", Name: "Tasks"}, database)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/databases/db-1")
	})

	t.Run("not found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{}`),
		}}

		_, err := testClient(t, httpContext).GetDatabase("missing")
		require.ErrorContains(t, err, "not found")
	})
}

func Test__Client__ListNewestPages(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[
			{"id":"page-1","created_time":"2026-01-03T09:00:00.000Z","properties":{"Name":{"type":"title","title":[{"plain_text":"Fix payment retries"}]}}}
		],"has_more":false}`),
	}}

	pages, err := testClient(t, httpContext).ListNewestPages("db-1", 30)
	require.NoError(t, err)
	require.Len(t, pages, 1)

	// The whole page travels, because a seeded page is read the way the
	// trigger would emit it.
	assert.Equal(t, "page-1", pages[0]["id"])
	assert.Equal(t, "Fix payment retries", PageTitle(pages[0]))

	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/databases/db-1/query")
}

func Test__Client__ListChangedPageDocuments(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[{"id":"page-1","created_time":"2026-01-03T09:00:00.000Z"}],"has_more":true,"next_cursor":"cursor-1"}`),
	}}

	documents, hasMore, nextCursor, err := testClient(t, httpContext).ListChangedPageDocuments("db-1", "cursor-0", 50)
	require.NoError(t, err)
	require.Len(t, documents, 1)
	assert.True(t, hasMore)
	assert.Equal(t, "cursor-1", nextCursor)
}

func Test__Client__PageContent(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[
			{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Retries fail silently."}]}},
			{"type":"bulleted_list_item","bulleted_list_item":{"rich_text":[{"plain_text":"Reproduce with a slow gateway."}]}},
			{"type":"to_do","to_do":{"rich_text":[{"plain_text":"Add a retry metric."}],"checked":true}},
			{"type":"divider","divider":{}}
		]}`),
	}}

	content, err := testClient(t, httpContext).PageContent("page-1")
	require.NoError(t, err)
	assert.Equal(t, "Retries fail silently.\n\n- Reproduce with a slow gateway.\n\n[x] Add a retry metric.", content)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/blocks/page-1/children")
}

func Test__Client__GetPage(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"id":"page-1","url":"https://www.notion.so/page-1","properties":{"Name":{"type":"title","title":[{"plain_text":"Fix payment retries"}]}}}`),
		jsonResponse(`{"results":[{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Retries fail silently."}]}}]}`),
	}}

	page, err := testClient(t, httpContext).GetPage("page-1")
	require.NoError(t, err)
	assert.Equal(t, &Page{
		ID:      "page-1",
		Title:   "Fix payment retries",
		Content: "Retries fail silently.",
		URL:     "https://www.notion.so/page-1",
	}, page)
}

func Test__Client__ListPages(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"results":[
			{"id":"page-1","url":"https://www.notion.so/page-1","properties":{"Name":{"type":"title","title":[{"plain_text":"Fix payment retries"}]}}},
			{"id":"page-2","url":"https://www.notion.so/page-2","properties":{"Name":{"type":"title","title":[{"plain_text":"Add retry metrics"}]}}}
		],"has_more":false}`),
	}}

	pages, err := testClient(t, httpContext).ListPages("db-1", "retry metrics", 10)
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, "page-2", pages[0].ID)
	assert.Equal(t, "Add retry metrics", pages[0].Title)
}

func Test__PageTitle(t *testing.T) {
	page := map[string]any{
		"properties": map[string]any{
			"Status": map[string]any{"type": "status"},
			"Name":   map[string]any{"type": "title", "title": []any{map[string]any{"plain_text": "Fix payment retries"}}},
		},
	}

	assert.Equal(t, "Fix payment retries", PageTitle(page))
}
