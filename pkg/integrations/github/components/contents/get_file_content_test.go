package contents

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__GetFileContent__Setup(t *testing.T) {
	component := GetFileContent{}

	t.Run("path is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "",
			},
		})

		require.ErrorContains(t, err, "path is required")
	})

	t.Run("stores repository metadata", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Integration: mocks.IntegrationContextForNewSetupFlow(),
			HTTP:        httpCtx,
			Metadata:    metadata,
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "docs/guide.md",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
	})
}

func Test__GetFileContent__Configuration(t *testing.T) {
	fields := (&GetFileContent{}).Configuration()
	require.Len(t, fields, 3)

	assert.Equal(t, "repository", fields[0].Name)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "path", fields[1].Name)
	assert.True(t, fields[1].Required)
	assert.Equal(t, "ref", fields[2].Name)
	assert.False(t, fields[2].Required)
}

func Test__GetFileContent__Execute(t *testing.T) {
	component := GetFileContent{}

	t.Run("gets and decodes a file at a ref", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"type": "file",
					"encoding": "base64",
					"name": "guide.md",
					"path": "docs/guide.md",
					"sha": "abc123",
					"content": "IyBHdWlkZQo="
				}`),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "docs/guide.md",
				"ref":        "feature/docs",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/repos/testhq/hello/contents/docs/guide.md", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "feature/docs", httpCtx.Requests[0].URL.Query().Get("ref"))
		assert.Equal(t, "github.fileContent", executionState.Type)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		output := payload["data"].(GetFileContentOutput)
		assert.Equal(t, "# Guide\n", output.Content)
		assert.Equal(t, "abc123", output.SHA)
		assert.Equal(t, "docs/guide.md", output.Path)
		assert.Equal(t, "feature/docs", output.Ref)
	})

	t.Run("rejects a directory", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `[
					{"type": "file", "name": "guide.md", "path": "docs/guide.md", "sha": "abc123"}
				]`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "docs",
			},
		})

		require.ErrorContains(t, err, "path points to a directory")
	})

	t.Run("uses the default branch when ref is omitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"type": "file",
					"encoding": "base64",
					"path": "README.md",
					"sha": "def456",
					"content": "IyBTdXBlclBsYW5lCg=="
				}`),
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "README.md",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Empty(t, httpCtx.Requests[0].URL.Query().Get("ref"))

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		output := payload["data"].(GetFileContentOutput)
		assert.Equal(t, "# SuperPlane\n", output.Content)
		assert.Empty(t, output.Ref)
	})
}
