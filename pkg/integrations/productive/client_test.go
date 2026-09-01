package productive

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func testIntegration(overrides map[string]any) *contexts.IntegrationContext {
	config := map[string]any{
		"apiToken":       "token-1",
		"organizationId": "org-1",
	}
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

func requestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	body := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func Test__NewClient(t *testing.T) {
	t.Run("missing apiToken -> error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, testIntegration(map[string]any{"apiToken": ""}))
		require.ErrorContains(t, err, "missing Productive.io API token")
	})

	t.Run("missing organizationId -> error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, testIntegration(map[string]any{"organizationId": ""}))
		require.ErrorContains(t, err, "missing Productive.io organization id")
	})

	t.Run("uses the default base URL when region is not set", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, testIntegration(nil))
		require.NoError(t, err)
		assert.Equal(t, BaseURL, client.BaseURL)
	})

	t.Run("region overrides the base URL", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, testIntegration(map[string]any{
			"region": "https://eu.productive.io/api/v2/",
		}))

		require.NoError(t, err)
		assert.Equal(t, "https://eu.productive.io/api/v2", client.BaseURL)
	})
}

func Test__Client__ListProjects(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[
			{"id":"1","type":"projects","attributes":{"name":"Payments"}},
			{"id":"2","type":"projects","attributes":{"name":"Growth"}}
		]}`),
	}}

	client := testClient(t, httpContext)

	projects, err := client.ListProjects()
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Equal(t, Project{ID: "1", Name: "Payments"}, projects[0])
	assert.Equal(t, Project{ID: "2", Name: "Growth"}, projects[1])

	require.Len(t, httpContext.Requests, 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/projects?")
	assert.Equal(t, "token-1", httpContext.Requests[0].Header.Get(AuthTokenHeader))
	assert.Equal(t, "org-1", httpContext.Requests[0].Header.Get(OrganizationIDHeader))
}

func Test__Client__ListProjects__StopsOnShortPage(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":[{"id":"1","type":"projects","attributes":{"name":"Payments"}}]}`),
	}}

	projects, err := testClient(t, httpContext).ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 1)
	assert.Len(t, httpContext.Requests, 1, "a page shorter than the page size must not be followed by another request")
}

func Test__Client__GetProject(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{"data":{"id":"1","type":"projects","attributes":{"name":"Payments"}}}`),
		}}

		project, err := testClient(t, httpContext).GetProject("1")
		require.NoError(t, err)
		assert.Equal(t, &Project{ID: "1", Name: "Payments"}, project)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/projects/1")
	})

	t.Run("not found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(`{"data":{}}`),
		}}

		_, err := testClient(t, httpContext).GetProject("missing")
		require.ErrorContains(t, err, "not found")
	})
}

func Test__Client__CreateWebhook(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":{"id":"w1","type":"webhooks"}}`),
	}}

	webhook, err := testClient(t, httpContext).CreateWebhook("https://sp.test/hook", "s3cr3t", []string{TaskCreatedEvent}, "1")
	require.NoError(t, err)
	assert.Equal(t, &Webhook{ID: "w1"}, webhook)

	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.Contains(t, req.URL.String(), "/webhooks")

	body := requestBody(t, req)
	data := body["data"].(map[string]any)
	attributes := data["attributes"].(map[string]any)
	assert.Equal(t, "https://sp.test/hook", attributes["target_url"])
	assert.Equal(t, "s3cr3t", attributes["secret"])

	relationships := data["relationships"].(map[string]any)
	project := relationships["project"].(map[string]any)["data"].(map[string]any)
	assert.Equal(t, "1", project["id"])
}

func Test__Client__CreateWebhook__WithoutProject(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":{"id":"w1","type":"webhooks"}}`),
	}}

	_, err := testClient(t, httpContext).CreateWebhook("https://sp.test/hook", "s3cr3t", []string{TaskCreatedEvent}, "")
	require.NoError(t, err)

	body := requestBody(t, httpContext.Requests[0])
	data := body["data"].(map[string]any)
	assert.NotContains(t, data, "relationships")
}

func Test__Client__DeleteWebhook(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(""))},
	}}

	err := testClient(t, httpContext).DeleteWebhook("w1")
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/webhooks/w1")
}
