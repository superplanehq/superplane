package artifactregistry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

type mockClient struct {
	projectID string
	getURL    func(ctx context.Context, fullURL string) ([]byte, error)
	postURL   func(ctx context.Context, fullURL string, body any) ([]byte, error)
}

func (m *mockClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.getURL != nil {
		return m.getURL(ctx, fullURL)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) PostURL(ctx context.Context, fullURL string, body any) ([]byte, error) {
	if m.postURL != nil {
		return m.postURL(ctx, fullURL, body)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) ProjectID() string {
	return m.projectID
}

func setTestClientFactory(
	t *testing.T,
	fn func(httpCtx core.HTTPContext, integration core.IntegrationContext) (Client, error),
) {
	t.Helper()

	clientFactoryMu.RLock()
	previous := clientFactory
	clientFactoryMu.RUnlock()

	SetClientFactory(fn)
	t.Cleanup(func() {
		SetClientFactory(previous)
	})
}

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

	t.Run("accepts valid resourceUrl", func(t *testing.T) {
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

func TestAnalyzeArtifactExecuteEmitsWhenOccurrencesFound(t *testing.T) {
	component := &AnalyzeArtifact{}
	resourceURL := "https://us-central1-docker.pkg.dev/demo-project/my-repo/my-image@sha256:abc123"

	client := &mockClient{
		projectID: "demo-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Contains(t, fullURL, "containeranalysis.googleapis.com/v1")
			assert.Contains(t, fullURL, "demo-project")
			assert.Contains(t, fullURL, "VULNERABILITY")
			return []byte(`{"occurrences":[{"name":"projects/demo-project/occurrences/vuln-1","kind":"VULNERABILITY"}]}`), nil
		},
	}

	setTestClientFactory(t, func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
		return client, nil
	})

	executionState := &testcontexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"resourceUrl": resourceURL,
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
		Metadata:       &testcontexts.MetadataContext{},
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, analyzeArtifactOutputChannel, executionState.Channel)
}

func TestAnalyzeArtifactExecutePollsWhenNoOccurrences(t *testing.T) {
	component := &AnalyzeArtifact{}
	resourceURL := "https://us-central1-docker.pkg.dev/demo-project/my-repo/my-image@sha256:abc123"

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
	requests := &testcontexts.RequestContext{}
	metadata := &testcontexts.MetadataContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"resourceUrl": resourceURL,
		},
		Integration:    &testcontexts.IntegrationContext{},
		ExecutionState: executionState,
		Requests:       requests,
		Metadata:       metadata,
	})

	require.NoError(t, err)
	assert.False(t, executionState.Passed)
	assert.Equal(t, analyzeArtifactPollAction, requests.Action)
}
