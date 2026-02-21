package jfrogartifactory

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

func Test__DeleteArtifact__ComponentInfo(t *testing.T) {
	component := DeleteArtifact{}

	assert.Equal(t, "jfrogArtifactory.deleteArtifact", component.Name())
	assert.Equal(t, "Delete Artifact", component.Label())
	assert.Equal(t, "jfrogArtifactory", component.Icon())
	assert.Equal(t, "gray", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func Test__DeleteArtifact__Configuration(t *testing.T) {
	component := DeleteArtifact{}
	config := component.Configuration()

	assert.Len(t, config, 2)

	fieldNames := make([]string, len(config))
	for i, f := range config {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "repository")
	assert.Contains(t, fieldNames, "path")

	for _, f := range config {
		assert.True(t, f.Required)
	}
}

func Test__DeleteArtifact__Setup(t *testing.T) {
	component := DeleteArtifact{}

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"path": "some/path",
			},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing path -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "libs-release",
			},
		})

		require.ErrorContains(t, err, "path is required")
	})

	t.Run("valid configuration -> stores metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP:        &contexts.HTTPContext{},
			Integration: &contexts.IntegrationContext{},
			Metadata:    metadataCtx,
			Configuration: map[string]any{
				"repository": "libs-release",
				"path":       "com/example/artifact.jar",
			},
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(DeleteArtifactNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "libs-release", metadata.Repository)
	})
}

func Test__DeleteArtifact__Execute(t *testing.T) {
	component := DeleteArtifact{}

	t.Run("successful delete -> emits deleted payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "test-token",
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "libs-release-local",
				"path":       "com/example/artifact-1.0.jar",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: execState,
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, DeleteArtifactPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)

		wrapped, ok := execState.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "libs-release-local", payload["repo"])
		assert.Equal(t, "com/example/artifact-1.0.jar", payload["path"])
	})

	t.Run("delete failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"status":403,"message":"Not enough permissions"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":         "https://mycompany.jfrog.io",
				"accessToken": "test-token",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"repository": "libs-release-local",
				"path":       "com/example/artifact-1.0.jar",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error deleting artifact")
	})
}
