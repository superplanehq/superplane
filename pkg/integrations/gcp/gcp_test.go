package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GCPIntegrationComponentsAndTriggers(t *testing.T) {
	g := &GCP{}

	components := g.Components()
	require.Len(t, components, 2)

	componentNames := make([]string, 0, len(components))
	for _, c := range components {
		componentNames = append(componentNames, c.Name())
	}
	assert.Contains(t, componentNames, "gcp.createVM")
	assert.Contains(t, componentNames, "gcp.pubsub.publishMessage")

	triggers := g.Triggers()
	require.Len(t, triggers, 2)

	triggerNames := make([]string, 0, len(triggers))
	for _, tr := range triggers {
		triggerNames = append(triggerNames, tr.Name())
	}
	assert.Contains(t, triggerNames, "gcp.compute.onVMInstance")
	assert.Contains(t, triggerNames, "gcp.pubsub.onTopicMessage")
}

func Test_topicSubscriptionApplies(t *testing.T) {
	g := &GCP{}

	t.Run("matches pubsub.topic pattern with correct topic", func(t *testing.T) {
		sub := &mockSubscription{config: map[string]any{
			"type":  "pubsub.topic",
			"topic": "my-topic",
		}}
		assert.True(t, g.topicSubscriptionApplies(sub, "my-topic"))
	})

	t.Run("does not match different topic", func(t *testing.T) {
		sub := &mockSubscription{config: map[string]any{
			"type":  "pubsub.topic",
			"topic": "my-topic",
		}}
		assert.False(t, g.topicSubscriptionApplies(sub, "other-topic"))
	})

	t.Run("does not match audit log pattern", func(t *testing.T) {
		sub := &mockSubscription{config: map[string]any{
			"serviceName": "compute.googleapis.com",
			"methodName":  "v1.compute.instances.insert",
		}}
		assert.False(t, g.topicSubscriptionApplies(sub, "my-topic"))
	})
}

type mockSubscription struct {
	config any
}

func (m *mockSubscription) Configuration() any {
	return m.config
}

func (m *mockSubscription) SendMessage(msg any) error {
	return nil
}

func Test_validateAndParseServiceAccountKey(t *testing.T) {
	t.Run("valid key returns metadata", func(t *testing.T) {
		key := []byte(`{
			"type": "service_account",
			"project_id": "my-project",
			"private_key_id": "abc",
			"private_key": "-----BEGIN PRIVATE KEY-----\nxyz\n-----END PRIVATE KEY-----",
			"client_email": "sa@my-project.iam.gserviceaccount.com",
			"client_id": "123"
		}`)
		meta, err := validateAndParseServiceAccountKey(key)
		require.NoError(t, err)
		assert.Equal(t, "my-project", meta.ProjectID)
		assert.Equal(t, "sa@my-project.iam.gserviceaccount.com", meta.ClientEmail)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		_, err := validateAndParseServiceAccountKey([]byte(`{invalid`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})

	t.Run("missing required field returns error", func(t *testing.T) {
		key := []byte(`{"type": "service_account", "project_id": "p"}`)
		_, err := validateAndParseServiceAccountKey(key)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required field")
	})

	t.Run("trims project_id and client_email", func(t *testing.T) {
		key := []byte(`{
			"type": "service_account",
			"project_id": "  proj  ",
			"private_key_id": "id",
			"private_key": "key",
			"client_email": "  sa@proj.iam.gserviceaccount.com  ",
			"client_id": "1"
		}`)
		meta, err := validateAndParseServiceAccountKey(key)
		require.NoError(t, err)
		assert.Equal(t, "proj", meta.ProjectID)
		assert.Equal(t, "sa@proj.iam.gserviceaccount.com", meta.ClientEmail)
	})
}
