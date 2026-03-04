package cloudbuild

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
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

func TestListBuildResources(t *testing.T) {
	ctx := context.Background()

	t.Run("returns parsed build resources from the integration project", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Equal(
					t,
					"https://cloudbuild.googleapis.com/v1/projects/integration-project/builds?pageSize=50",
					fullURL,
				)
				return []byte(`{
					"builds": [
						{
							"id": "9f7d716f-a898-424e-8bda-ac2dc2bf8247",
							"status": "SUCCESS",
							"createTime": "2026-03-03T04:35:20.025201Z"
						}
					]
				}`), nil
			},
		}

		resources, err := ListBuildResources(ctx, client, "")

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(
			t,
			core.IntegrationResource{
				Type: ResourceTypeBuild,
				ID:   "9f7d716f-a898-424e-8bda-ac2dc2bf8247",
				Name: "9f7d716f-a898-424e-8bda-ac2dc2bf8247",
			},
			resources[0],
		)
	})

	t.Run("uses the project override when provided", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Equal(
					t,
					"https://cloudbuild.googleapis.com/v1/projects/override-project/builds?pageSize=50",
					fullURL,
				)
				return []byte(`{"builds":[]}`), nil
			},
		}

		resources, err := ListBuildResources(ctx, client, "override-project")

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("falls back to the resource name when id is missing", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return []byte(`{
					"builds": [
						{
							"name": "projects/integration-project/locations/global/builds/build-from-name",
							"status": "FAILURE"
						}
					]
				}`), nil
			},
		}

		resources, err := ListBuildResources(ctx, client, "")

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "projects/integration-project/locations/global/builds/build-from-name", resources[0].ID)
		assert.Equal(t, "build-from-name", resources[0].Name)
	})

	t.Run("returns empty resources when cloud build is unavailable", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return nil, &gcpcommon.GCPAPIError{StatusCode: 404}
			},
		}

		resources, err := ListBuildResources(ctx, client, "")

		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func TestListLocationResources(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{
		projectID: "integration-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v2/projects/integration-project/locations?pageSize=100",
				fullURL,
			)
			return []byte(`{
				"locations": [
					{
						"name": "projects/integration-project/locations/us-central1",
						"locationId": "us-central1",
						"displayName": "Iowa"
					}
				]
			}`), nil
		},
	}

	resources, err := ListLocationResources(ctx, client, "")

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "us-central1", resources[0].ID)
	assert.Equal(t, "Iowa (us-central1)", resources[0].Name)
}

func TestListConnectionResources(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{
		projectID: "integration-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v2/projects/integration-project/locations/us-central1/connections?pageSize=50",
				fullURL,
			)
			return []byte(`{
				"connections": [
					{
						"name": "projects/integration-project/locations/us-central1/connections/github-main",
						"githubConfig": {}
					}
				]
			}`), nil
		},
	}

	resources, err := ListConnectionResources(ctx, client, "", "us-central1")

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(
		t,
		core.IntegrationResource{
			Type: ResourceTypeConnection,
			ID:   "projects/integration-project/locations/us-central1/connections/github-main",
			Name: "github-main",
		},
		resources[0],
	)
}

func TestListRepositoryResources(t *testing.T) {
	ctx := context.Background()
	client := &mockClient{
		projectID: "integration-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v2/projects/integration-project/locations/us-central1/connections/github-main/repositories?pageSize=50",
				fullURL,
			)
			return []byte(`{
				"repositories": [
					{
						"name": "projects/integration-project/locations/us-central1/connections/github-main/repositories/rtlbx",
						"remoteUri": "https://github.com/WashyKK/rtlbx.git"
					}
				]
			}`), nil
		},
	}

	resources, err := ListRepositoryResources(
		ctx,
		client,
		"projects/integration-project/locations/us-central1/connections/github-main",
	)

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(
		t,
		core.IntegrationResource{
			Type: ResourceTypeRepository,
			ID:   "projects/integration-project/locations/us-central1/connections/github-main/repositories/rtlbx",
			Name: "https://github.com/WashyKK/rtlbx.git",
		},
		resources[0],
	)
}

func TestListGitRefResources(t *testing.T) {
	ctx := context.Background()
	repository := "projects/integration-project/locations/us-central1/connections/github-main/repositories/rtlbx"
	client := &mockClient{
		projectID: "integration-project",
		getURL: func(_ context.Context, fullURL string) ([]byte, error) {
			assert.Equal(
				t,
				"https://cloudbuild.googleapis.com/v2/projects/integration-project/locations/us-central1/connections/github-main/repositories/rtlbx:fetchGitRefs?refType=BRANCH&pageSize=100",
				fullURL,
			)
			return []byte(`{"refNames":["refs/heads/main","refs/heads/release/v1"]}`), nil
		},
	}

	resources, err := ListBranchResources(ctx, client, repository)

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "refs/heads/main", resources[0].ID)
	assert.Equal(t, "main", resources[0].Name)
	assert.Equal(t, "release/v1", resources[1].Name)
}

func TestWithPageTokenEncodesReservedCharacters(t *testing.T) {
	assert.Equal(
		t,
		"https://example.com/resources?pageSize=50&pageToken=next%2Btoken%2Fwith%3Dchars",
		withPageToken("https://example.com/resources?pageSize=50", "next+token/with=chars"),
	)
	assert.Equal(
		t,
		"https://example.com/resources?pageToken=next%2Btoken%2Fwith%3Dchars",
		withPageToken("https://example.com/resources", "next+token/with=chars"),
	)
}
