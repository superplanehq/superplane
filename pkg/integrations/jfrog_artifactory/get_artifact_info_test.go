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

func Test__GetArtifactInfo__ComponentInfo(t *testing.T) {
	component := GetArtifactInfo{}

	assert.Equal(t, "jfrogArtifactory.getArtifactInfo", component.Name())
	assert.Equal(t, "Get Artifact Info", component.Label())
	assert.Equal(t, "jfrogArtifactory", component.Icon())
	assert.Equal(t, "gray", component.Color())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
}

func Test__GetArtifactInfo__Configuration(t *testing.T) {
	component := GetArtifactInfo{}
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

func Test__GetArtifactInfo__Setup(t *testing.T) {
	component := GetArtifactInfo{}

	t.Run("missing repository -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"path": "some/path"},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("missing path -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "libs-release"},
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

		metadata, ok := metadataCtx.Metadata.(GetArtifactInfoNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "libs-release", metadata.Repository)
	})
}

func Test__GetArtifactInfo__Execute(t *testing.T) {
	component := GetArtifactInfo{}

	t.Run("successful execution -> emits artifact info", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"repo": "libs-release-local",
						"path": "/com/example/artifact-1.0.jar",
						"created": "2026-01-15T10:30:00.000Z",
						"createdBy": "admin",
						"lastModified": "2026-01-15T10:30:00.000Z",
						"modifiedBy": "admin",
						"downloadUri": "https://mycompany.jfrog.io/libs-release-local/com/example/artifact-1.0.jar",
						"mimeType": "application/java-archive",
						"size": "12345",
						"checksums": {
							"sha1": "abc123",
							"md5": "def456",
							"sha256": "ghi789"
						},
						"uri": "https://mycompany.jfrog.io/api/storage/libs-release-local/com/example/artifact-1.0.jar"
					}`)),
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
		assert.Equal(t, GetArtifactInfoPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
	})

	t.Run("artifact not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"status":404,"message":"File not found."}]}`)),
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
				"path":       "nonexistent/artifact.jar",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Logger:         logrus.NewEntry(logrus.New()),
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting artifact info")
	})
}
