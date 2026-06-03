package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockImageClient is a Client mock for the image components (create/update/delete).
type mockImageClient struct {
	projectID  string
	getFunc    func(ctx context.Context, path string) ([]byte, error)
	postFunc   func(ctx context.Context, path string, body any) ([]byte, error)
	deleteFunc func(ctx context.Context, path string) ([]byte, error)
}

func (m *mockImageClient) Get(ctx context.Context, path string) ([]byte, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, path)
	}
	return nil, fmt.Errorf("Get not implemented")
}

func (m *mockImageClient) Post(ctx context.Context, path string, body any) ([]byte, error) {
	if m.postFunc != nil {
		return m.postFunc(ctx, path, body)
	}
	return nil, fmt.Errorf("Post not implemented")
}

func (m *mockImageClient) Delete(ctx context.Context, path string) ([]byte, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, path)
	}
	return nil, fmt.Errorf("Delete not implemented")
}

func (m *mockImageClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	return nil, fmt.Errorf("GetURL not implemented")
}

func (m *mockImageClient) ProjectID() string {
	return m.projectID
}

// imageGetJSON builds an images.get response body matching imageGetResp.
func imageGetJSON(name, status, family string, labels map[string]string, fingerprint, deprecationState string) []byte {
	body := map[string]any{
		"id":                "1234567890123456789",
		"name":              name,
		"selfLink":          fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/my-project/global/images/%s", name),
		"family":            family,
		"status":            status,
		"diskSizeGb":        "10",
		"sourceDisk":        "https://www.googleapis.com/compute/v1/projects/my-project/zones/us-central1-a/disks/my-disk",
		"creationTimestamp": "2026-06-02T12:00:00.000-07:00",
		"labelFingerprint":  fingerprint,
	}
	if labels != nil {
		body["labels"] = labels
	}
	if deprecationState != "" {
		body["deprecated"] = map[string]any{"state": deprecationState}
	}
	b, _ := json.Marshal(body)
	return b
}

func Test__ParseImagePath(t *testing.T) {
	t.Run("bare name", func(t *testing.T) {
		project, name, err := parseImagePath("my-image")
		require.NoError(t, err)
		assert.Equal(t, "", project)
		assert.Equal(t, "my-image", name)
	})

	t.Run("relative global path", func(t *testing.T) {
		project, name, err := parseImagePath("global/images/my-image")
		require.NoError(t, err)
		assert.Equal(t, "", project)
		assert.Equal(t, "my-image", name)
	})

	t.Run("project-qualified path", func(t *testing.T) {
		project, name, err := parseImagePath("projects/elffie/global/images/my-image")
		require.NoError(t, err)
		assert.Equal(t, "elffie", project)
		assert.Equal(t, "my-image", name)
	})

	t.Run("full selfLink URL", func(t *testing.T) {
		selfLink := "https://www.googleapis.com/compute/v1/projects/elffie/global/images/web-base-01"
		project, name, err := parseImagePath(selfLink)
		require.NoError(t, err)
		assert.Equal(t, "elffie", project)
		assert.Equal(t, "web-base-01", name)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		project, name, err := parseImagePath("  my-image  ")
		require.NoError(t, err)
		assert.Equal(t, "", project)
		assert.Equal(t, "my-image", name)
	})

	t.Run("empty value is rejected", func(t *testing.T) {
		_, _, err := parseImagePath("")
		require.Error(t, err)
	})

	t.Run("non-image path is rejected", func(t *testing.T) {
		_, _, err := parseImagePath("zones/us-central1-a/instances/my-vm")
		require.Error(t, err)
	})
}

func Test__ImagePayloadFromGetResponse(t *testing.T) {
	t.Run("active image without deprecation", func(t *testing.T) {
		body := imageGetJSON("my-image", "READY", "my-app", map[string]string{"env": "prod"}, "abc", "")
		payload, err := ImagePayloadFromGetResponse(body)
		require.NoError(t, err)
		assert.Equal(t, "my-image", payload["name"])
		assert.Equal(t, "READY", payload["status"])
		assert.Equal(t, "my-app", payload["family"])
		assert.Equal(t, int64(10), payload["diskSizeGb"])
		assert.Equal(t, "my-disk", payload["sourceDisk"])
		assert.Equal(t, "ACTIVE", payload["deprecationState"])
		labels := payload["labels"].(map[string]string)
		assert.Equal(t, "prod", labels["env"])
	})

	t.Run("deprecated image surfaces state", func(t *testing.T) {
		body := imageGetJSON("my-image", "READY", "", nil, "abc", "DEPRECATED")
		payload, err := ImagePayloadFromGetResponse(body)
		require.NoError(t, err)
		assert.Equal(t, "DEPRECATED", payload["deprecationState"])
	})

	t.Run("unparseable body errors", func(t *testing.T) {
		_, err := ImagePayloadFromGetResponse([]byte("not-json"))
		require.Error(t, err)
	})
}

func Test__MergeImageLabels(t *testing.T) {
	t.Run("overwrites, adds, and preserves", func(t *testing.T) {
		existing := map[string]string{"env": "staging", "team": "core"}
		updates := map[string]string{"env": "prod", "owner": "platform"}
		merged := mergeImageLabels(existing, updates)
		assert.Equal(t, map[string]string{"env": "prod", "team": "core", "owner": "platform"}, merged)
	})

	t.Run("nil existing yields updates only", func(t *testing.T) {
		merged := mergeImageLabels(nil, map[string]string{"env": "prod"})
		assert.Equal(t, map[string]string{"env": "prod"}, merged)
	})

	t.Run("does not mutate the existing map", func(t *testing.T) {
		existing := map[string]string{"env": "staging"}
		mergeImageLabels(existing, map[string]string{"env": "prod"})
		assert.Equal(t, "staging", existing["env"])
	})
}

func Test__WaitForGlobalOperation(t *testing.T) {
	t.Run("DONE returns nil", func(t *testing.T) {
		mc := &mockImageClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				assert.True(t, strings.Contains(path, "/global/operations/op-1"))
				return opDone("op-1"), nil
			},
		}
		require.NoError(t, WaitForGlobalOperation(context.Background(), mc, "my-project", "op-1"))
	})

	t.Run("operation error is surfaced", func(t *testing.T) {
		mc := &mockImageClient{
			projectID: "my-project",
			getFunc: func(ctx context.Context, path string) ([]byte, error) {
				b, _ := json.Marshal(map[string]any{
					"name":   "op-1",
					"status": "DONE",
					"error":  map[string]any{"errors": []map[string]any{{"message": "boom"}}},
				})
				return b, nil
			},
		}
		err := WaitForGlobalOperation(context.Background(), mc, "my-project", "op-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})
}
