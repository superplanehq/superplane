package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__SSH__Name(t *testing.T) {
	s := &SSH{}
	assert.Equal(t, "ssh", s.Name())
}

func Test__SSH__Label(t *testing.T) {
	s := &SSH{}
	assert.Equal(t, "SSH", s.Label())
}

func Test__SSH__Icon(t *testing.T) {
	s := &SSH{}
	assert.Equal(t, "server", s.Icon())
}

func Test__SSH__Description(t *testing.T) {
	s := &SSH{}
	assert.NotEmpty(t, s.Description())
}

func Test__SSH__Configuration(t *testing.T) {
	s := &SSH{}
	fields := s.Configuration()

	require.Len(t, fields, 5)
	assert.Equal(t, "host", fields[0].Name)
	assert.Equal(t, "port", fields[1].Name)
	assert.Equal(t, "username", fields[2].Name)
	assert.Equal(t, "privateKey", fields[3].Name)
	assert.Equal(t, "passphrase", fields[4].Name)
	assert.True(t, fields[3].Sensitive)
	assert.True(t, fields[4].Sensitive)
}

func Test__SSH__Components(t *testing.T) {
	s := &SSH{}
	components := s.Components()

	require.Len(t, components, 3)
	assert.IsType(t, &ExecuteCommand{}, components[0])
	assert.IsType(t, &ExecuteScript{}, components[1])
	assert.IsType(t, &HostMetadata{}, components[2])
}

func Test__SSH__Triggers(t *testing.T) {
	s := &SSH{}
	triggers := s.Triggers()

	assert.Len(t, triggers, 0)
}

func Test__SSH__CompareWebhookConfig(t *testing.T) {
	s := &SSH{}
	equal, err := s.CompareWebhookConfig(nil, nil)

	require.NoError(t, err)
	assert.False(t, equal)
}

func Test__SSH__SetupWebhook(t *testing.T) {
	s := &SSH{}
	ctx := core.SetupWebhookContext{}
	result, err := s.SetupWebhook(ctx)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support webhooks")
}

func Test__SSH__CleanupWebhook(t *testing.T) {
	s := &SSH{}
	ctx := core.CleanupWebhookContext{}
	err := s.CleanupWebhook(ctx)

	assert.NoError(t, err)
}

func Test__SSH__ListResources(t *testing.T) {
	s := &SSH{}

	t.Run("returns empty for non-host resource type", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Hosts: []HostInfo{
						{Host: "example.com", Port: 22, Username: "user"},
					},
				},
			},
		}

		resources, err := s.ListResources("other", ctx)
		require.NoError(t, err)
		assert.Len(t, resources, 0)
	})

	t.Run("returns hosts for host resource type", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Hosts: []HostInfo{
						{Host: "example.com", Port: 22, Username: "user"},
						{Host: "test.com", Port: 2222, Username: "admin"},
					},
				},
			},
		}

		resources, err := s.ListResources("host", ctx)
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "user@example.com:22", resources[0].ID)
		assert.Equal(t, "admin@test.com:2222", resources[1].ID)
	})
}
