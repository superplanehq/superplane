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

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "not a map"})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("requires repository", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"path": "README.md"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("requires file path", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"repository": "hello"},
		})

		require.ErrorContains(t, err, "file path is required")
	})
}

func Test__GetFileContent__Execute(t *testing.T) {
	component := GetFileContent{}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("requires repository", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"path": "README.md"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("requires file path", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{"repository": "hello"},
		})

		require.ErrorContains(t, err, "file path is required")
	})

	t.Run("reads and emits decoded file content", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"type": "file",
					"encoding": "base64",
					"size": 14,
					"name": "README.md",
					"path": "docs/README.md",
					"content": "SGVsbG8sIHdvcmxkIQo=",
					"sha": "4d4ba9968a4c7c80e25b715bd7522f2c4dc51f3f",
					"url": "https://api.github.com/repos/testhq/hello/contents/docs/README.md",
					"html_url": "https://github.com/testhq/hello/blob/main/docs/README.md",
					"download_url": "https://raw.githubusercontent.com/testhq/hello/main/docs/README.md"
				}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    mocks.IntegrationContextForNewSetupFlow(),
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"repository": "hello",
				"path":       "docs/README.md",
				"ref":        "main",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		require.Equal(t, "github.fileContent", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "/repos/testhq/hello/contents/docs/README.md", httpCtx.Requests[0].URL.Path)
		assert.Equal(t, "main", httpCtx.Requests[0].URL.Query().Get("ref"))

		payload := executionState.Payloads[0].(map[string]any)
		output := payload["data"].(GetFileContentOutput)
		assert.Equal(t, "Hello, world!\n", output.Content)
		assert.Equal(t, "README.md", output.Name)
		assert.Equal(t, "docs/README.md", output.Path)
		assert.Equal(t, "4d4ba9968a4c7c80e25b715bd7522f2c4dc51f3f", output.SHA)
		assert.Equal(t, 14, output.Size)
		assert.Equal(t, "main", output.Ref)
	})

	t.Run("rejects directory paths", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `[
					{"type":"file","name":"one.txt","path":"docs/one.txt","sha":"abc"}
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

		require.ErrorContains(t, err, `path "docs" points to a directory; expected a file`)
	})
}
