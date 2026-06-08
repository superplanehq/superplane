package tpu

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient is a test double for the Client interface.
type mockClient struct {
	projectID string
	getURL    func(ctx context.Context, fullURL string) ([]byte, error)
	postURL   func(ctx context.Context, fullURL string, body any) ([]byte, error)
	deleteURL func(ctx context.Context, fullURL string) ([]byte, error)
}

func (m *mockClient) ProjectID() string { return m.projectID }

func (m *mockClient) GetURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.getURL != nil {
		return m.getURL(ctx, fullURL)
	}
	return nil, nil
}

func (m *mockClient) PostURL(ctx context.Context, fullURL string, body any) ([]byte, error) {
	if m.postURL != nil {
		return m.postURL(ctx, fullURL, body)
	}
	return nil, nil
}

func (m *mockClient) DeleteURL(ctx context.Context, fullURL string) ([]byte, error) {
	if m.deleteURL != nil {
		return m.deleteURL(ctx, fullURL)
	}
	return nil, nil
}

func isOperationURL(url string) bool {
	return strings.Contains(url, "/operations/")
}

func opStartedJSON(name string) []byte {
	b, _ := json.Marshal(map[string]any{"name": name})
	return b
}

func opDoneJSON(name string) []byte {
	b, _ := json.Marshal(map[string]any{"name": name, "done": true})
	return b
}

func nodeJSON(resourceName, state string, labels map[string]string) []byte {
	node := map[string]any{
		"name":            resourceName,
		"acceleratorType": "v2-8",
		"runtimeVersion":  "tpu-vm-tf-2.16.1",
		"state":           state,
		"health":          "HEALTHY",
		"createTime":      "2026-01-28T10:30:00Z",
		"networkEndpoints": []map[string]any{
			{"ipAddress": "10.128.0.5", "port": 8470},
		},
	}
	if len(labels) > 0 {
		node["labels"] = labels
	}
	b, _ := json.Marshal(node)
	return b
}

func Test__ParseNodeName(t *testing.T) {
	t.Run("full resource name", func(t *testing.T) {
		project, location, node, err := parseNodeName("projects/my-project/locations/us-central1-b/nodes/my-tpu")
		require.NoError(t, err)
		assert.Equal(t, "my-project", project)
		assert.Equal(t, "us-central1-b", location)
		assert.Equal(t, "my-tpu", node)
	})

	t.Run("missing node errors", func(t *testing.T) {
		_, _, _, err := parseNodeName("projects/my-project/locations/us-central1-b")
		require.Error(t, err)
	})
}

func Test__ResolveNodeSelection(t *testing.T) {
	t.Run("full resource name derives location", func(t *testing.T) {
		location, nodeID, err := resolveNodeSelection("projects/my-project/locations/us-central1-b/nodes/my-tpu", "my-project")
		require.NoError(t, err)
		assert.Equal(t, "us-central1-b", location)
		assert.Equal(t, "my-tpu", nodeID)
	})

	t.Run("cross-project resource name fails", func(t *testing.T) {
		_, _, err := resolveNodeSelection("projects/other-project/locations/us-central1-b/nodes/my-tpu", "my-project")
		require.ErrorContains(t, err, "cross-project")
	})

	t.Run("empty node fails", func(t *testing.T) {
		_, _, err := resolveNodeSelection("", "my-project")
		require.ErrorContains(t, err, "node is required")
	})

	t.Run("bare name (not a resource path) fails", func(t *testing.T) {
		_, _, err := resolveNodeSelection("my-tpu", "my-project")
		require.ErrorContains(t, err, "select a TPU node from the list")
	})
}

func Test__LabelsFromEntries(t *testing.T) {
	t.Run("builds map and drops empty keys", func(t *testing.T) {
		out := labelsFromEntries([]LabelEntry{
			{Key: "env", Value: "prod"},
			{Key: "  ", Value: "x"},
			{Key: "team", Value: "core"},
		})
		assert.Equal(t, map[string]string{"env": "prod", "team": "core"}, out)
	})

	t.Run("empty list returns nil", func(t *testing.T) {
		assert.Nil(t, labelsFromEntries(nil))
	})
}

func Test__NodePayloadFromResponse(t *testing.T) {
	payload, err := nodePayloadFromResponse(nodeJSON("projects/my-project/locations/us-central1-b/nodes/my-tpu", "READY", map[string]string{"env": "prod"}))
	require.NoError(t, err)
	assert.Equal(t, "my-tpu", payload["name"])
	assert.Equal(t, "us-central1-b", payload["location"])
	assert.Equal(t, "v2-8", payload["acceleratorType"])
	assert.Equal(t, "READY", payload["state"])
	assert.Equal(t, "HEALTHY", payload["health"])
	assert.Equal(t, []string{"10.128.0.5"}, payload["ipAddresses"])
}

func Test__WaitForOperation(t *testing.T) {
	t.Run("done returns body", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return opDoneJSON("projects/my-project/locations/us-central1-b/operations/op1"), nil
			},
		}
		_, err := waitForOperation(context.Background(), mc, "projects/my-project/locations/us-central1-b/operations/op1")
		require.NoError(t, err)
	})

	t.Run("error surfaces failure message", func(t *testing.T) {
		mc := &mockClient{
			projectID: "my-project",
			getURL: func(_ context.Context, _ string) ([]byte, error) {
				return []byte(`{"name":"op1","done":true,"error":{"code":3,"message":"boom"}}`), nil
			},
		}
		_, err := waitForOperation(context.Background(), mc, "op1")
		require.ErrorContains(t, err, "boom")
	})
}
