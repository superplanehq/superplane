package artifactregistry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestAnalyzeArtifactSetup(t *testing.T) {
	component := &AnalyzeArtifact{}

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

func TestAnalyzeArtifactExecuteEmitsImmediatelyWhenOccurrencesExist(t *testing.T) {
	component := &AnalyzeArtifact{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Contains(t, fullURL, "containeranalysis.googleapis.com/v1")
			return []byte(`{"occurrences":[{"name":"occ-1","kind":"VULNERABILITY"}]}`), nil
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
		Metadata:       &testcontexts.MetadataContext{},
		Requests:       &testcontexts.RequestContext{},
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, analyzeArtifactOutputChannel, executionState.Channel)
}

func TestAnalyzeArtifactExecuteSchedulesPollWhenNoOccurrences(t *testing.T) {
	component := &AnalyzeArtifact{}
	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, _ string) ([]byte, error) {
			return []byte(`{"occurrences":[]}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}
	requestCtx := &testcontexts.RequestContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"resourceUrl": "https://us-central1-docker.pkg.dev/demo-project/my-repo/my-image@sha256:abc123",
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
		Metadata:       &testcontexts.MetadataContext{},
		Requests:       requestCtx,
	})

	require.NoError(t, err)
	assert.False(t, executionState.Passed, "should not emit immediately when no occurrences")
	assert.Equal(t, analyzeArtifactPollAction, requestCtx.Action)
}

func TestAnalyzeArtifactMetadata(t *testing.T) {
	component := &AnalyzeArtifact{}
	assert.Equal(t, "gcp.artifactregistry.analyzeArtifact", component.Name())
	assert.Equal(t, "Artifact Registry • Analyze Artifact", component.Label())
	assert.NotEmpty(t, component.Description())
	assert.NotEmpty(t, component.Documentation())
	assert.Equal(t, "gcp", component.Icon())
	assert.Equal(t, "gray", component.Color())
}
