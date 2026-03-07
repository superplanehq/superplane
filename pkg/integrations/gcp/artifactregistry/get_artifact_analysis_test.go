package artifactregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetArtifactAnalysisSetup(t *testing.T) {
	component := &GetArtifactAnalysis{}

	t.Run("rejects missing resourceUrl", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Integration:   &testcontexts.IntegrationContext{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "resourceUrl is required")
	})

	t.Run("accepts valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"resourceUrl": "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func TestGetArtifactAnalysisExecute(t *testing.T) {
	component := &GetArtifactAnalysis{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Contains(t, fullURL, "containeranalysis.googleapis.com/v1")
			assert.Contains(t, fullURL, "projects/demo-project/occurrences")
			return []byte(`{"occurrences":[]}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"resourceUrl": "https://us-central1-docker.pkg.dev/demo-project/my-repo/my-image@sha256:abc123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, getArtifactAnalysisOutputChannel, executionState.Channel)
}

func TestGetArtifactAnalysisMetadata(t *testing.T) {
	component := &GetArtifactAnalysis{}
	assert.Equal(t, "gcp.artifactregistry.getArtifactAnalysis", component.Name())
	assert.Equal(t, "Artifact Registry • Get Artifact Analysis", component.Label())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
	assert.Equal(t, "gcp", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
