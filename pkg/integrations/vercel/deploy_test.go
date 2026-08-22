package vercel

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Vercel_TriggerDeployment__Setup(t *testing.T) {
	t.Run("missing project -> error", func(t *testing.T) {
		err := (&TriggerDeployment{}).Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "project is required")
	})

	t.Run("invalid target -> error", func(t *testing.T) {
		err := (&TriggerDeployment{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1", "target": "staging"},
		})
		require.ErrorContains(t, err, "target must be production or preview")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := (&TriggerDeployment{}).Setup(core.SetupContext{
			Configuration: map[string]any{"project": "prj_1"},
		})
		require.NoError(t, err)
	})
}

func Test__Vercel_TriggerDeployment__Execute(t *testing.T) {
	projectResponse := `{"id":"prj_1","name":"my-app","link":{"type":"github","org":"acme","repo":"web","productionBranch":"main"}}`
	deploymentResponse := `{"id":"dpl_1","url":"my-app.vercel.app","readyState":"QUEUED","target":"production","projectId":"prj_1","createdAt":1755850000000}`

	newHTTPContext := func() *contexts.HTTPContext {
		return &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(projectResponse)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(deploymentResponse)),
				},
			},
		}
	}

	t.Run("project without git link -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"prj_1","name":"my-app"}`)),
				},
			},
		}

		err := (&TriggerDeployment{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"project": "prj_1"},
		})

		require.ErrorContains(t, err, "not connected to a supported Git repository")
	})

	t.Run("valid project -> emits queued deployment", func(t *testing.T) {
		httpCtx := newHTTPContext()
		executionState := &contexts.ExecutionStateContext{}

		err := (&TriggerDeployment{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "my-app", "gitRef": "feature-x"},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, TriggerDeploymentPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "dpl_1", data["deploymentId"])
		assert.Equal(t, "QUEUED", data["readyState"])
		assert.Equal(t, "production", data["target"])

		require.Len(t, httpCtx.Requests, 2)

		projectRequest := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, projectRequest.Method)
		assert.Equal(t, "/v9/projects/my-app", projectRequest.URL.Path)
		assert.Empty(t, projectRequest.URL.Query().Get("teamId"))

		createRequest := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPost, createRequest.Method)
		assert.Equal(t, "/v13/deployments", createRequest.URL.Path)

		requestBody, bodyErr := io.ReadAll(createRequest.Body)
		require.NoError(t, bodyErr)

		decodedBody := map[string]any{}
		require.NoError(t, json.Unmarshal(requestBody, &decodedBody))

		gitSource := readMap(decodedBody["gitSource"])
		assert.Equal(t, "github", gitSource["type"])
		assert.Equal(t, "acme", gitSource["org"])
		assert.Equal(t, "web", gitSource["repo"])
		assert.Equal(t, "feature-x", gitSource["ref"])
		assert.Equal(t, "production", decodedBody["target"])
	})

	t.Run("preview target omits target field", func(t *testing.T) {
		// Preview deployments have no target in the Vercel response.
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, projectResponse),
			response(http.StatusOK, `{"id":"dpl_1","url":"my-app.vercel.app","readyState":"QUEUED","projectId":"prj_1"}`),
		}}
		executionState := &contexts.ExecutionStateContext{}

		err := (&TriggerDeployment{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"accessToken": "vercel_token_123"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"project": "my-app", "target": "preview"},
		})

		require.NoError(t, err)

		requestBody, bodyErr := io.ReadAll(httpCtx.Requests[1].Body)
		require.NoError(t, bodyErr)
		decodedBody := map[string]any{}
		require.NoError(t, json.Unmarshal(requestBody, &decodedBody))
		assert.NotContains(t, decodedBody, "target", "Vercel rejects target=preview; omission means preview")

		emitted := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "preview", emitted["target"], "emitted data still reports the chosen target")
	})

	t.Run("team id is sent as query parameter", func(t *testing.T) {
		httpCtx := newHTTPContext()

		err := (&TriggerDeployment{}).Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"accessToken": "vercel_token_123",
				"teamId":      "team_123",
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"project": "my-app"},
		})

		require.NoError(t, err)

		createRequest := httpCtx.Requests[1]
		assert.Equal(t, "team_123", createRequest.URL.Query().Get("teamId"))

		requestBody, bodyErr := io.ReadAll(createRequest.Body)
		require.NoError(t, bodyErr)

		decodedBody := map[string]any{}
		require.NoError(t, json.Unmarshal(requestBody, &decodedBody))
		assert.NotContains(t, decodedBody, "teamId")

		gitSource := readMap(decodedBody["gitSource"])
		assert.Equal(t, "main", gitSource["ref"], "defaults to the production branch")
	})
}

func Test__Vercel_GitSourceFromProject(t *testing.T) {
	t.Run("unsupported git type -> error", func(t *testing.T) {
		_, err := gitSourceFromProject(&Project{
			Name: "my-app",
			Link: &ProjectLink{Type: "azure-devops", Org: "acme", Repo: "web"},
		}, "")

		require.ErrorContains(t, err, "not connected to a supported Git repository")
	})

	t.Run("missing org or repo -> error", func(t *testing.T) {
		_, err := gitSourceFromProject(&Project{
			Name: "my-app",
			Link: &ProjectLink{Type: "github", Org: "", Repo: "web"},
		}, "")

		require.ErrorContains(t, err, "not connected to a supported Git repository")
	})

	t.Run("falls back to main when no branch is known", func(t *testing.T) {
		gitSource, err := gitSourceFromProject(&Project{
			Name: "my-app",
			Link: &ProjectLink{Type: "GitHub", Org: "acme", Repo: "web"},
		}, "")

		require.NoError(t, err)
		assert.Equal(t, "main", gitSource.Ref)
	})
}
