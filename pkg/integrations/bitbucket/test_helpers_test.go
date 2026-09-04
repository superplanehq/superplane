package bitbucket

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

const (
	testWorkspace      = "superplane"
	testRepositorySlug = "web"
	testRepositoryName = "superplane/web"
)

func testIntegrationMetadata() Metadata {
	return Metadata{
		AuthType: AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{
			UUID: "{workspace-uuid}",
			Name: "SuperPlane",
			Slug: testWorkspace,
		},
	}
}

// testNodeMetadata pre-resolves the repository the way Setup() does, so Execute()
// tests only need to mock the API call under test.
func testNodeMetadata() NodeMetadata {
	return NodeMetadata{
		Repository: &RepositoryMetadata{
			UUID:     "{repository-uuid}",
			Name:     "web",
			FullName: testRepositoryName,
			Slug:     testRepositorySlug,
		},
	}
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type executionFixture struct {
	Context        core.ExecutionContext
	HTTP           *contexts.HTTPContext
	ExecutionState *contexts.ExecutionStateContext
}

func newExecutionFixture(config map[string]any, responses ...*http.Response) executionFixture {
	httpCtx := &contexts.HTTPContext{Responses: responses}
	executionState := &contexts.ExecutionStateContext{}

	return executionFixture{
		HTTP:           httpCtx,
		ExecutionState: executionState,
		Context: core.ExecutionContext{
			Logger:         log.NewEntry(log.New()),
			HTTP:           httpCtx,
			Configuration:  config,
			Metadata:       &contexts.MetadataContext{Metadata: testNodeMetadata()},
			ExecutionState: executionState,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"token": "token"},
				Metadata:      testIntegrationMetadata(),
			},
		},
	}
}

func newSetupContext(config map[string]any, responses ...*http.Response) core.SetupContext {
	return core.SetupContext{
		Logger:        log.NewEntry(log.New()),
		HTTP:          &contexts.HTTPContext{Responses: responses},
		Configuration: config,
		Metadata:      &contexts.MetadataContext{Metadata: testNodeMetadata()},
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      testIntegrationMetadata(),
		},
	}
}

// emittedPayload unwraps the {type, timestamp, data} envelope the execution state puts
// around every emitted payload.
func emittedPayload[T any](t *testing.T, state *contexts.ExecutionStateContext, index int) T {
	t.Helper()

	require.Greater(t, len(state.Payloads), index)

	wrapper, ok := state.Payloads[index].(map[string]any)
	require.True(t, ok, "payload is not an envelope")

	payload, ok := wrapper["data"].(T)
	require.True(t, ok, "payload data has an unexpected type: %T", wrapper["data"])

	return payload
}

// requestBody decodes the JSON payload of a recorded request. The fake HTTP context
// never reads the body, so it is still available to assert on.
func requestBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()

	require.NotNil(t, request.Body)

	raw, err := io.ReadAll(request.Body)
	require.NoError(t, err)

	body := map[string]any{}
	require.NoError(t, json.Unmarshal(raw, &body))

	return body
}
