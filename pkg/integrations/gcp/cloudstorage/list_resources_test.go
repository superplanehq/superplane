package cloudstorage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	projectID   string
	getURL      func(ctx context.Context, fullURL string) ([]byte, error)
	postURL     func(ctx context.Context, fullURL string, body any) ([]byte, error)
	execRequest func(ctx context.Context, method, url string, body io.Reader) ([]byte, error)
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

func (m *mockClient) ExecRequest(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	if m.execRequest != nil {
		return m.execRequest(ctx, method, url, body)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) ProjectID() string {
	return m.projectID
}

func TestListBucketResources(t *testing.T) {
	t.Run("lists buckets from project", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Contains(t, fullURL, "project=my-project")
				return json.Marshal(bucketListResponse{
					Items: []bucketItem{
						{Name: "bucket-a", Location: "US", StorageClass: "STANDARD"},
						{Name: "bucket-b", Location: "EU", StorageClass: "NEARLINE"},
					},
				})
			},
		}

		resources, err := ListBucketResources(context.Background(), client, "")
		require.NoError(t, err)
		require.Len(t, resources, 2)

		assert.Equal(t, "bucket-a", resources[0].ID)
		assert.Equal(t, "bucket-a (US)", resources[0].Name)
		assert.Equal(t, ResourceTypeBucket, resources[0].Type)

		assert.Equal(t, "bucket-b", resources[1].ID)
		assert.Equal(t, "bucket-b (EU)", resources[1].Name)
	})

	t.Run("uses provided project ID", func(t *testing.T) {
		client := &mockClient{
			projectID: "default-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				assert.Contains(t, fullURL, "project=override-project")
				return json.Marshal(bucketListResponse{})
			},
		}

		resources, err := ListBucketResources(context.Background(), client, "override-project")
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("skips buckets with empty name", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return json.Marshal(bucketListResponse{
					Items: []bucketItem{
						{Name: "", Location: "US"},
						{Name: "valid-bucket", Location: "US"},
					},
				})
			},
		}

		resources, err := ListBucketResources(context.Background(), client, "")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "valid-bucket", resources[0].ID)
	})

	t.Run("handles pagination", func(t *testing.T) {
		callCount := 0
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, fullURL string) ([]byte, error) {
				callCount++
				if callCount == 1 {
					return json.Marshal(bucketListResponse{
						Items:         []bucketItem{{Name: "bucket-1", Location: "US"}},
						NextPageToken: "token-abc",
					})
				}
				assert.Contains(t, fullURL, "pageToken=token-abc")
				return json.Marshal(bucketListResponse{
					Items: []bucketItem{{Name: "bucket-2", Location: "EU"}},
				})
			},
		}

		resources, err := ListBucketResources(context.Background(), client, "")
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "bucket-1", resources[0].ID)
		assert.Equal(t, "bucket-2", resources[1].ID)
		assert.Equal(t, 2, callCount)
	})

	t.Run("shows name without location when location is empty", func(t *testing.T) {
		client := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return json.Marshal(bucketListResponse{
					Items: []bucketItem{{Name: "bucket-no-loc"}},
				})
			},
		}

		resources, err := ListBucketResources(context.Background(), client, "")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "bucket-no-loc", resources[0].Name)
	})

	t.Run("returns nil for empty project", func(t *testing.T) {
		client := &mockClient{projectID: ""}
		resources, err := ListBucketResources(context.Background(), client, "")
		require.NoError(t, err)
		assert.Nil(t, resources)
	})
}
