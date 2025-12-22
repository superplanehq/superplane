package integrations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func Test__DescribeIntegration(t *testing.T) {
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
		URL:        "http://localhost:8000",
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

	t.Run("integration that does not exist -> error", func(t *testing.T) {
		_, err := DescribeIntegration(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), uuid.New().String())
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "integration not found", s.Message())
	})

	t.Run("using id", func(t *testing.T) {
		response, err := DescribeIntegration(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), integration.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, integration.ID.String(), response.Integration.Metadata.Id)
		assert.Equal(t, r.Organization.ID.String(), response.Integration.Metadata.DomainId)
		assert.Equal(t, *integration.CreatedAt, response.Integration.Metadata.CreatedAt.AsTime())
		assert.Equal(t, integration.Name, response.Integration.Metadata.Name)
	})

	t.Run("using name", func(t *testing.T) {
		response, err := DescribeIntegration(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), integration.Name)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Integration)
		assert.Equal(t, integration.ID.String(), response.Integration.Metadata.Id)
		assert.Equal(t, r.Organization.ID.String(), response.Integration.Metadata.DomainId)
		assert.Equal(t, *integration.CreatedAt, response.Integration.Metadata.CreatedAt.AsTime())
		assert.Equal(t, integration.Name, response.Integration.Metadata.Name)
	})
}
