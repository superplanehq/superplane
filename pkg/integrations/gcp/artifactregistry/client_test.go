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

func TestParseArtifactResourceURL(t *testing.T) {
	t.Run("parses digest URL", func(t *testing.T) {
		loc, repo, pkg, ver, err := parseArtifactResourceURL("https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123")
		require.NoError(t, err)
		assert.Equal(t, "us-central1", loc)
		assert.Equal(t, "my-repo", repo)
		assert.Equal(t, "my-image", pkg)
		assert.Equal(t, "sha256:abc123", ver)
	})

	t.Run("parses tag URL", func(t *testing.T) {
		loc, repo, pkg, ver, err := parseArtifactResourceURL("https://europe-west1-docker.pkg.dev/proj/repo/image:latest")
		require.NoError(t, err)
		assert.Equal(t, "europe-west1", loc)
		assert.Equal(t, "repo", repo)
		assert.Equal(t, "image", pkg)
		assert.Equal(t, "latest", ver)
	})

	t.Run("rejects missing digest or tag", func(t *testing.T) {
		_, _, _, _, err := parseArtifactResourceURL("https://us-central1-docker.pkg.dev/proj/repo/image")
		require.ErrorContains(t, err, "must include @digest or :tag")
	})

	t.Run("rejects non-artifact-registry host", func(t *testing.T) {
		_, _, _, _, err := parseArtifactResourceURL("https://gcr.io/proj/image@sha256:abc")
		require.ErrorContains(t, err, "-docker.pkg.dev")
	})
}

func TestGetArtifactSetup(t *testing.T) {
	component := &GetArtifact{}

	t.Run("accepts empty configuration in url mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Integration:   &testcontexts.IntegrationContext{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("rejects select mode with missing location", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"inputMode": "select"},
			Integration:   &testcontexts.IntegrationContext{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "location is required")
	})

	t.Run("accepts valid resourceUrl in url mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":   "url",
				"resourceUrl": "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("accepts expression in resourceUrl", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":   "url",
				"resourceUrl": "{{ $[\"trigger\"].data.digest }}",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("accepts all four fields in select mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":  "select",
				"location":   "us-central1",
				"repository": "my-repo",
				"package":    "my-image",
				"version":    "sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func TestGetArtifactAnalysisSetup(t *testing.T) {
	component := &GetArtifactAnalysis{}

	t.Run("accepts empty configuration in url mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Integration:   &testcontexts.IntegrationContext{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("rejects select mode with missing location", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"inputMode": "select"},
			Integration:   &testcontexts.IntegrationContext{},
			Metadata:      &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "location is required")
	})

	t.Run("accepts valid resourceUrl in url mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":   "url",
				"resourceUrl": "https://us-central1-docker.pkg.dev/my-project/my-repo/my-image@sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("accepts expression in resourceUrl", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":   "url",
				"resourceUrl": "{{ $[\"trigger\"].data.resourceUri }}",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("accepts all four fields in select mode", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"inputMode":  "select",
				"location":   "us-central1",
				"repository": "my-repo",
				"package":    "my-image",
				"version":    "sha256:abc123",
			},
			Integration: &testcontexts.IntegrationContext{},
			Metadata:    &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}
