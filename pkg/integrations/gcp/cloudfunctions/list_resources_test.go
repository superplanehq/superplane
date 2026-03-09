package cloudfunctions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
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

func TestListLocationResources(t *testing.T) {
	ctx := context.Background()

	t.Run("returns parsed location resources", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Equal(t,
					"https://cloudfunctions.googleapis.com/v2/projects/my-project/locations?pageSize=100",
					fullURL,
				)
				return []byte(`{
					"locations": [
						{"locationId": "us-central1", "displayName": "Iowa"},
						{"locationId": "europe-west1", "displayName": "Belgium"}
					]
				}`), nil
			},
		}

		resources, err := ListLocationResources(ctx, client, "")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, core.IntegrationResource{
			Type: ResourceTypeLocation,
			ID:   "us-central1",
			Name: "Iowa (us-central1)",
		}, resources[0])
		assert.Equal(t, "Belgium (europe-west1)", resources[1].Name)
	})

	t.Run("uses project override when provided", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Contains(t, fullURL, "projects/override-project/")
				return []byte(`{"locations":[]}`), nil
			},
		}

		resources, err := ListLocationResources(ctx, client, "override-project")
		require.NoError(t, err)
		assert.Empty(t, resources)
	})
}

func TestListFunctionResources(t *testing.T) {
	ctx := context.Background()

	t.Run("returns Cloud Functions and Cloud Run services combined", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				if fullURL == "https://cloudfunctions.googleapis.com/v2/projects/my-project/locations/us-central1/functions?pageSize=100" {
					return []byte(`{
						"functions": [
							{
								"name": "projects/my-project/locations/us-central1/functions/hello-world",
								"state": "ACTIVE"
							}
						]
					}`), nil
				}
				if fullURL == "https://run.googleapis.com/v2/projects/my-project/locations/us-central1/services?pageSize=100" {
					return []byte(`{
						"services": [
							{
								"name": "projects/my-project/locations/us-central1/services/my-service",
								"uri": "https://my-service-abc123-uc.a.run.app"
							}
						]
					}`), nil
				}
				t.Errorf("unexpected URL: %s", fullURL)
				return nil, nil
			},
		}

		resources, err := ListFunctionResources(ctx, client, "", "us-central1")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, core.IntegrationResource{
			Type: ResourceTypeFunction,
			ID:   "projects/my-project/locations/us-central1/functions/hello-world",
			Name: "hello-world",
		}, resources[0])
		assert.Equal(t, core.IntegrationResource{
			Type: ResourceTypeFunction,
			ID:   "projects/my-project/locations/us-central1/services/my-service",
			Name: "my-service",
		}, resources[1])
	})

	t.Run("returns nil when location is empty", func(t *testing.T) {
		client := &mockClient{projectID: "my-project"}
		resources, err := ListFunctionResources(ctx, client, "", "")
		require.NoError(t, err)
		assert.Nil(t, resources)
	})
}
