package cloudstorage

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

func TestListBucketResources(t *testing.T) {
	ctx := context.Background()

	t.Run("returns parsed bucket resources", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Contains(t, fullURL, "project=my-project")
				return []byte(`{
					"items": [
						{"name": "bucket-one", "id": "bucket-one"},
						{"name": "bucket-two", "id": "bucket-two"}
					]
				}`), nil
			},
		}

		resources, err := ListBucketResources(ctx, client, "")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, core.IntegrationResource{
			Type: ResourceTypeBucket,
			ID:   "bucket-one",
			Name: "bucket-one",
		}, resources[0])
		assert.Equal(t, "bucket-two", resources[1].Name)
	})

	t.Run("uses project override when provided", func(t *testing.T) {
		client := &mockClient{
			projectID: "integration-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Contains(t, fullURL, "project=override-project")
				return []byte(`{"items":[]}`), nil
			},
		}

		resources, err := ListBucketResources(ctx, client, "override-project")
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("returns nil when project is empty", func(t *testing.T) {
		client := &mockClient{projectID: ""}
		resources, err := ListBucketResources(ctx, client, "")
		require.NoError(t, err)
		assert.Nil(t, resources)
	})

	t.Run("handles pagination", func(t *testing.T) {
		call := 0
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				call++
				if call == 1 {
					return []byte(`{
						"items": [{"name": "bucket-1", "id": "bucket-1"}],
						"nextPageToken": "token123"
					}`), nil
				}
				assert.Contains(t, fullURL, "pageToken=token123")
				return []byte(`{
					"items": [{"name": "bucket-2", "id": "bucket-2"}]
				}`), nil
			},
		}

		resources, err := ListBucketResources(ctx, client, "")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "bucket-1", resources[0].Name)
		assert.Equal(t, "bucket-2", resources[1].Name)
	})
}
