package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"

	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateIntegration(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	secret, err := support.CreateSecret(t, r, map[string]string{"key": "value"})
	require.NoError(t, err)

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := CreateIntegration(context.Background(), r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), &protos.Integration{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("missing name", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "",
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration name is required", s.Message())
	})

	t.Run("missing integration type", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration type is required", s.Message())
	})

	t.Run("integration type not available", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: "does-not-exist",
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration type does-not-exist not available", s.Message())
	})

	t.Run("invalid secret", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: "does-not-exist",
								Key:  "nope",
							},
						},
					},
				},
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "error finding secret does-not-exist: record not found", s.Message())
	})

	t.Run("invalid secret key", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: secret.Name,
								Key:  "nope",
							},
						},
					},
				},
			},
		}

		_, err = CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "key nope not found in secret "+secret.Name, s.Message())
	})

	t.Run("integration is created", func(t *testing.T) {
		name := support.RandomName("integration")
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: name,
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: secret.Name,
								Key:  "key",
							},
						},
					},
				},
			},
		}

		response, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, name, response.Integration.Metadata.Name)
		assert.NotEmpty(t, response.Integration.Metadata.Id)
		assert.NotEmpty(t, response.Integration.Metadata.CreatedAt)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_ORGANIZATION, response.Integration.Metadata.DomainType)
		assert.Equal(t, r.Organization.ID.String(), response.Integration.Metadata.DomainId)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		name := support.RandomName("integration")
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: name,
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: secret.Name,
								Key:  "key",
							},
						},
					},
				},
			},
		}

		//
		// No organization integration with this name yet, so this works.
		//
		integration.Spec.Auth.Token.ValueFrom.Secret.Name = secret.Name
		_, err = CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		require.NoError(t, err)

		//
		// Name already taken
		//
		integration.Spec.Auth.Token.ValueFrom.Secret.Name = secret.Name
		_, err = CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeOrganization, r.Organization.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
