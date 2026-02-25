package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetAgentSettings(t *testing.T) {
	r := support.Setup(t)

	resp, err := GetAgentSettings(r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.AgentSettings)
	require.NotNil(t, resp.AgentSettings.OpenaiKey)

	assert.Equal(t, r.Organization.ID.String(), resp.AgentSettings.OrganizationId)
	assert.False(t, resp.AgentSettings.AgentModeEnabled)
	assert.False(t, resp.AgentSettings.AgentModeEffective)
	assert.False(t, resp.AgentSettings.OpenaiKey.Configured)
	assert.Equal(t, models.OrganizationAgentOpenAIKeyStatusNotConfigured, resp.AgentSettings.OpenaiKey.Status)
}

func Test__UpdateAgentSettings_AllowsPendingEnablementWithoutKey(t *testing.T) {
	r := support.Setup(t)

	resp, err := UpdateAgentSettings(r.Organization.ID.String(), true, r.User.String())
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.AgentSettings)

	assert.True(t, resp.AgentSettings.AgentModeEnabled)
	assert.False(t, resp.AgentSettings.AgentModeEffective)
	assert.False(t, resp.AgentSettings.OpenaiKey.Configured)
	assert.Equal(t, models.OrganizationAgentOpenAIKeyStatusNotConfigured, resp.AgentSettings.OpenaiKey.Status)
}

func Test__SetAgentOpenAIKey(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	t.Run("empty key returns error", func(t *testing.T) {
		_, err := SetAgentOpenAIKey(
			ctx,
			r.Encryptor,
			r.Organization.ID.String(),
			r.User.String(),
			"   ",
			false,
		)
		require.Error(t, err)

		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("save with validate=false stores unchecked status", func(t *testing.T) {
		resp, err := SetAgentOpenAIKey(
			ctx,
			r.Encryptor,
			r.Organization.ID.String(),
			r.User.String(),
			"sk-this-is-a-valid-format-test-key-12345",
			false,
		)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.AgentSettings)
		require.NotNil(t, resp.AgentSettings.OpenaiKey)

		assert.True(t, resp.AgentSettings.OpenaiKey.Configured)
		assert.Equal(t, models.OrganizationAgentOpenAIKeyStatusUnchecked, resp.AgentSettings.OpenaiKey.Status)
		assert.Equal(t, "2345", resp.AgentSettings.OpenaiKey.Last4)
		assert.False(t, resp.AgentSettings.AgentModeEffective)

		settings, lookupErr := models.FindOrganizationAgentSettingsByOrganizationID(r.Organization.ID.String())
		require.NoError(t, lookupErr)
		assert.NotEmpty(t, settings.OpenAIApiKeyCiphertext)
		assert.NotNil(t, settings.OpenAIKeyEncryptionKeyID)
	})
}

func Test__DeleteAgentOpenAIKey(t *testing.T) {
	r := support.Setup(t)
	ctx := context.Background()

	_, err := SetAgentOpenAIKey(
		ctx,
		r.Encryptor,
		r.Organization.ID.String(),
		r.User.String(),
		"sk-this-is-a-valid-format-test-key-12345",
		false,
	)
	require.NoError(t, err)

	resp, err := DeleteAgentOpenAIKey(r.Organization.ID.String(), r.User.String())
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.AgentSettings)
	require.NotNil(t, resp.AgentSettings.OpenaiKey)

	assert.False(t, resp.AgentSettings.OpenaiKey.Configured)
	assert.Equal(t, models.OrganizationAgentOpenAIKeyStatusNotConfigured, resp.AgentSettings.OpenaiKey.Status)
	assert.Empty(t, resp.AgentSettings.OpenaiKey.Last4)

	settings, err := models.FindOrganizationAgentSettingsByOrganizationID(r.Organization.ID.String())
	require.NoError(t, err)
	assert.Empty(t, settings.OpenAIApiKeyCiphertext)
	assert.Nil(t, settings.OpenAIKeyEncryptionKeyID)
}

func Test__UpdateAgentSettings_InvalidUserID(t *testing.T) {
	r := support.Setup(t)

	_, err := UpdateAgentSettings(r.Organization.ID.String(), true, uuid.NewString()+"-invalid")
	require.Error(t, err)

	s, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, s.Code())
	assert.Equal(t, "invalid user id", s.Message())
}
