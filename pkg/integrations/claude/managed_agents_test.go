package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func jsonResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
}

func TestClient_ListManagedAgents(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":[{"id":"agent_1","name":"First","version":3}],"next_page":"page_2"}`),
			jsonResponse(`{"data":[{"id":"agent_2","name":"Second","version":1}],"next_page":""}`),
		},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	agents, err := client.ListManagedAgents()
	require.NoError(t, err)
	require.Len(t, agents, 2)
	assert.Equal(t, "agent_1", agents[0].ID)
	assert.Equal(t, "First", agents[0].Name)
	assert.Equal(t, 3, agents[0].Version)
	assert.Equal(t, "agent_2", agents[1].ID)

	require.Len(t, httpCtx.Requests, 2)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/agents"))
	assert.Equal(t, anthropicManagedAgentsBeta, httpCtx.Requests[0].Header.Get("anthropic-beta"))
	assert.Equal(t, "page_2", httpCtx.Requests[1].URL.Query().Get("page"))
}

func TestClient_ListManagedEnvironments(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":[{"id":"env_1","name":"prod"},{"id":"env_2","name":"staging"}],"next_page":""}`),
		},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	envs, err := client.ListManagedEnvironments()
	require.NoError(t, err)
	require.Len(t, envs, 2)
	assert.Equal(t, "env_1", envs[0].ID)
	assert.Equal(t, "prod", envs[0].Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/environments"))
}

func TestClient_ListManagedAgentVersions_sortsNewestFirst(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":[{"version":1},{"version":3},{"version":2}],"next_page":""}`),
		},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	versions, err := client.ListManagedAgentVersions("agent_1")
	require.NoError(t, err)
	require.Len(t, versions, 3)
	assert.Equal(t, 3, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, 1, versions[2].Version)
	assert.True(t, strings.Contains(httpCtx.Requests[0].URL.Path, "/agents/agent_1/versions"))
}

func TestClient_ListManagedAgentVersions_requiresAgent(t *testing.T) {
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}
	_, err := client.ListManagedAgentVersions("")
	require.Error(t, err)
}

func listResourcesCtx(response *http.Response, params map[string]string) core.ListResourcesContext {
	var responses []*http.Response
	if response != nil {
		responses = []*http.Response{response}
	}
	return core.ListResourcesContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        &contexts.HTTPContext{Responses: responses},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Parameters:  params,
	}
}

func TestClaude_ListResources_agents(t *testing.T) {
	i := &Claude{}
	res, err := i.ListResources("agent", listResourcesCtx(
		jsonResponse(`{"data":[{"id":"agent_1","name":"First"},{"id":"agent_2","name":""}],"next_page":""}`), nil))
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "agent_1", res[0].ID)
	assert.Equal(t, "First", res[0].Name)
	// Falls back to the ID when the agent has no name.
	assert.Equal(t, "agent_2", res[1].Name)
}

func TestClaude_ListResources_environments(t *testing.T) {
	i := &Claude{}
	res, err := i.ListResources("environment", listResourcesCtx(
		jsonResponse(`{"data":[{"id":"env_1","name":"prod"}],"next_page":""}`), nil))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, "env_1", res[0].ID)
	assert.Equal(t, "prod", res[0].Name)
}

func TestClaude_ListResources_agentVersions_prependsLatest(t *testing.T) {
	i := &Claude{}
	res, err := i.ListResources("agentVersion", listResourcesCtx(
		jsonResponse(`{"data":[{"version":2},{"version":1}],"next_page":""}`),
		map[string]string{"agent": "agent_1"}))
	require.NoError(t, err)
	require.Len(t, res, 3)
	// An explicit Latest option comes first so the field can be reset to latest.
	assert.Equal(t, "latest", res[0].ID)
	assert.Equal(t, "Latest", res[0].Name)
	assert.Equal(t, "2", res[1].ID)
	assert.Equal(t, "2", res[1].Name)
	assert.Equal(t, "1", res[2].ID)
	assert.Equal(t, "1", res[2].Name)
}

func TestClaude_ListResources_agentVersions_noAgentSelected(t *testing.T) {
	i := &Claude{}
	// No agent parameter yet: return empty without calling the API.
	res, err := i.ListResources("agentVersion", listResourcesCtx(nil, nil))
	require.NoError(t, err)
	assert.Empty(t, res)
}
