package artifactregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetArtifactSetupRejectsMissingFields(t *testing.T) {
	component := &GetArtifact{}

	t.Run("rejects missing location", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"repository": "my-repo",
				"image":      "img@sha256:abc",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "location is required")
	})

	t.Run("rejects missing repository", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"location": "us-central1",
				"image":    "img@sha256:abc",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("rejects missing image", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"location":   "us-central1",
				"repository": "my-repo",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "image is required")
	})

	t.Run("accepts valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"location":   "us-central1",
				"repository": "my-repo",
				"image":      "my-image@sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func TestGetArtifactExecuteCallsCorrectURL(t *testing.T) {
	component := &GetArtifact{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Contains(t, fullURL, "artifactregistry.googleapis.com/v1")
			assert.Contains(t, fullURL, "projects/demo-project")
			assert.Contains(t, fullURL, "locations/us-central1")
			assert.Contains(t, fullURL, "repositories/my-repo")
			return []byte(`{"name":"projects/demo-project/locations/us-central1/repositories/my-repo/dockerImages/img","uri":"test"}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"location":   "us-central1",
			"repository": "my-repo",
			"image":      "my-image@sha256:abc123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, getArtifactOutputChannel, executionState.Channel)
}

func TestGetArtifactMetadata(t *testing.T) {
	component := &GetArtifact{}
	assert.Equal(t, "gcp.artifactregistry.getArtifact", component.Name())
	assert.Equal(t, "Artifact Registry • Get Artifact", component.Label())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
	assert.Equal(t, "gcp", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
