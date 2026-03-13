package artifactregistry

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestGetArtifactExecuteRespectsInputMode(t *testing.T) {
	component := &GetArtifact{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://artifactregistry.googleapis.com/v1/projects/demo-project/locations/us-central1/repositories/my-repo/packages/my-image/versions/sha256:abc123",
				fullURL,
			)
			return []byte(`{"name":"projects/demo-project/locations/us-central1/repositories/my-repo/packages/my-image/versions/sha256:abc123"}`), nil
		},
	}
	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	execState := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"inputMode":   "select",
			"resourceUrl": "https://us-central1-docker.pkg.dev/other-project/other-repo/other-image@sha256:stale",
			"location":    "us-central1",
			"repository":  "my-repo",
			"package":     "my-image",
			"version":     "sha256:abc123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, getArtifactPayloadType, execState.Type)
}

func TestGetArtifactExecuteURLModeRequiresResourceURL(t *testing.T) {
	component := &GetArtifact{}
	execState := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"inputMode": "url"},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.False(t, execState.Passed)
	assert.Contains(t, execState.FailureMessage, "resourceUrl is required in url mode")
}

func TestGetArtifactAnalysisExecuteRespectsInputMode(t *testing.T) {
	component := &GetArtifactAnalysis{}
	expectedEncodedResourceURL := "resourceUrl%3D%22https%3A%2F%2Fus-central1-docker.pkg.dev%2Fdemo-project%2Fmy-repo%2Fmy-image%40sha256%3Aabc123%22"
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Contains(t, fullURL, expectedEncodedResourceURL)
			assert.NotContains(t, fullURL, "other-project")
			return []byte(`{"occurrences":[]}`), nil
		},
	}
	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	execState := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"inputMode":   "select",
			"resourceUrl": "https://us-central1-docker.pkg.dev/other-project/other-repo/other-image@sha256:stale",
			"location":    "us-central1",
			"repository":  "my-repo",
			"package":     "my-image",
			"version":     "sha256:abc123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, getArtifactAnalysisPayloadType, execState.Type)
}

func TestGetArtifactAnalysisExecuteURLModeRequiresResourceURL(t *testing.T) {
	component := &GetArtifactAnalysis{}
	execState := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}

	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"inputMode": "url"},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.False(t, execState.Passed)
	assert.Contains(t, execState.FailureMessage, "resourceUrl is required in url mode")
}

func TestListPackageResourcesPreservesNestedPackagePath(t *testing.T) {
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, _ string) ([]byte, error) {
			return []byte(`{
				"packages": [
					{
						"name": "projects/demo-project/locations/us-central1/repositories/my-repo/packages/team/service/image"
					}
				]
			}`), nil
		},
	}

	resources, err := ListPackageResources(context.Background(), client, "", "us-central1", "my-repo")
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "team/service/image", resources[0].ID)
}

func TestListVersionResourcesUsesFullPackagePath(t *testing.T) {
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.True(
				t,
				strings.Contains(fullURL, "/packages/team/service/image/versions"),
				"expected version list URL to include full package path, got: %s",
				fullURL,
			)
			return []byte(`{"versions":[]}`), nil
		},
	}

	resources, err := ListVersionResources(context.Background(), client, "", "us-central1", "my-repo", "team/service/image")
	require.NoError(t, err)
	assert.Empty(t, resources)
}
