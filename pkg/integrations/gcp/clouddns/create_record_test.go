package clouddns

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateRecord_Metadata(t *testing.T) {
	c := &CreateRecord{}
	assert.Equal(t, "gcp.clouddns.createRecord", c.Name())
	assert.Equal(t, "Cloud DNS • Create Record", c.Label())
	assert.NotEmpty(t, c.Description())
	assert.NotEmpty(t, c.Documentation())
	assert.Equal(t, "gcp", c.Icon())
	assert.Equal(t, "gray", c.Color())
}

func TestCreateRecord_ExampleOutput(t *testing.T) {
	c := &CreateRecord{}
	output := c.ExampleOutput()
	assert.NotEmpty(t, output["type"])
	assert.NotEmpty(t, output["timestamp"])
	payload, ok := output["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, payload["change"])
	assert.NotEmpty(t, payload["record"])
}

func TestCreateRecord_Setup(t *testing.T) {
	c := &CreateRecord{}

	t.Run("succeeds with valid config", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"1.2.3.4"},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("fails when managed zone is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "api.example.com",
				"type":    "A",
				"rrdatas": []string{"1.2.3.4"},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "managed zone is required")
	})

	t.Run("fails when record name is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"type":        "A",
				"rrdatas":     []string{"1.2.3.4"},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "record name is required")
	})

	t.Run("fails when record type is missing", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"rrdatas":     []string{"1.2.3.4"},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "record type is required")
	})

	t.Run("fails when rrdatas is empty", func(t *testing.T) {
		err := c.Setup(core.SetupContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"rrdatas":     []string{},
			},
			Metadata: &testcontexts.MetadataContext{},
		})
		require.ErrorContains(t, err, "at least one record value is required")
	})
}

func TestCreateRecord_Execute(t *testing.T) {
	t.Run("emits output when change is done immediately", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "1",
						"status":    "done",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		err := (&CreateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"1.2.3.4"},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.True(t, state.Passed)
		require.Len(t, state.Payloads, 1)
		wrapper := state.Payloads[0].(map[string]any)
		data := wrapper["data"].(map[string]any)
		change := data["change"].(map[string]any)
		assert.Equal(t, "done", change["status"])
		record := data["record"].(map[string]any)
		assert.Equal(t, "api.example.com.", record["name"])
		assert.Equal(t, "A", record["type"])
	})

	t.Run("schedules poll when change is pending", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "2",
						"status":    "pending",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		meta := &testcontexts.MetadataContext{}
		requests := &testcontexts.RequestContext{}
		err := (&CreateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"1.2.3.4"},
			},
			ExecutionState: &testcontexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       meta,
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.Equal(t, pollChangeActionName, requests.Action)
		stored := meta.Get().(RecordSetPollMetadata)
		assert.Equal(t, "2", stored.ChangeID)
		assert.Equal(t, "my-zone", stored.ManagedZone)
	})

	t.Run("fails when change status is unexpected", func(t *testing.T) {
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, _ any) ([]byte, error) {
					return json.Marshal(map[string]any{
						"id":        "3",
						"status":    "failed",
						"startTime": "2026-01-28T10:30:00.000Z",
					})
				},
			}, nil
		})

		state := &testcontexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &testcontexts.RequestContext{}
		err := (&CreateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"ttl":         300,
				"rrdatas":     []string{"1.2.3.4"},
			},
			ExecutionState: state,
			Metadata:       &testcontexts.MetadataContext{},
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.True(t, state.Finished)
		assert.False(t, state.Passed)
		assert.Contains(t, state.FailureMessage, "unexpected Cloud DNS change status")
		assert.Empty(t, requests.Action)
	})

	t.Run("normalizes record name to add trailing dot", func(t *testing.T) {
		var capturedBody any
		SetClientFactory(func(_ core.HTTPContext, _ core.IntegrationContext) (Client, error) {
			return &mockClient{
				projectID: "my-project",
				postURL: func(_ context.Context, _ string, body any) ([]byte, error) {
					capturedBody = body
					return json.Marshal(map[string]any{"id": "1", "status": "done"})
				},
			}, nil
		})

		err := (&CreateRecord{}).Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"managedZone": "my-zone",
				"name":        "api.example.com",
				"type":        "A",
				"rrdatas":     []string{"1.2.3.4"},
			},
			ExecutionState: &testcontexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &testcontexts.MetadataContext{},
		})

		require.NoError(t, err)
		bodyMap := capturedBody.(map[string]any)
		additions := bodyMap["additions"].([]ResourceRecordSet)
		assert.Equal(t, "api.example.com.", additions[0].Name)
	})
}

// mockClient is a test double for the Client interface.
type mockClient struct {
	projectID string
	getURL    func(ctx context.Context, fullURL string) ([]byte, error)
	postURL   func(ctx context.Context, fullURL string, body any) ([]byte, error)
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
