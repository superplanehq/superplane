package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__ListIntegrations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	secret, err := support.CreateSecret(t, r, map[string]string{"key": "test"})
	require.NoError(t, err)
	integration, err := models.CreateIntegration(&models.Integration{
		Name:       support.RandomName("integration"),
		CreatedBy:  r.User,
		Type:       models.IntegrationTypeSemaphore,
		DomainType: models.DomainTypeOrganization,
		DomainID:   r.Organization.ID,
		URL:        "http://localhost:8800",
		AuthType:   models.IntegrationAuthTypeToken,
		Auth: datatypes.NewJSONType(models.IntegrationAuth{
			Token: &models.IntegrationAuthToken{
				ValueFrom: models.ValueDefinitionFrom{
					Secret: &models.ValueDefinitionFromSecret{
						Name: secret.Name,
						Key:  "key",
					},
				},
			},
		}),
	})

	require.NoError(t, err)

	t.Run("returns list of integrations", func(t *testing.T) {
		res, err := ListIntegrations(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Len(t, res.Integrations, 1)
		assert.Equal(t, integration.ID.String(), res.Integrations[0].Metadata.Id)
		assert.Equal(t, integration.Name, res.Integrations[0].Metadata.Name)
		assert.Equal(t, integration.DomainID.String(), res.Integrations[0].Metadata.DomainId)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_ORGANIZATION, res.Integrations[0].Metadata.DomainType)
		assert.NotEmpty(t, res.Integrations[0].Metadata.CreatedAt)
		assert.Equal(t, integration.CreatedBy.String(), res.Integrations[0].Metadata.CreatedBy)
		assert.Equal(t, models.IntegrationTypeSemaphore, res.Integrations[0].Spec.Type)
		assert.Equal(t, integration.URL, res.Integrations[0].Spec.Url)
		assert.Equal(t, protos.Integration_AUTH_TYPE_TOKEN, res.Integrations[0].Spec.Auth.Use)
		assert.NotNil(t, res.Integrations[0].Spec.Auth.Token)
	})
}
