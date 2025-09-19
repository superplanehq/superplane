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

func Test__UpdateIntegration(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	canvasSecret, err := support.CreateCanvasSecret(t, r, map[string]string{"key": "value"})
	require.NoError(t, err)

	orgSecret, err := support.CreateOrganizationSecret(t, r, map[string]string{"key": "value"})
	require.NoError(t, err)

	createIntegrationSpec := &protos.Integration{
		Metadata: &protos.Integration_Metadata{
			Name: support.RandomName("integration"),
		},
		Spec: &protos.Integration_Spec{
			Type: models.IntegrationTypeSemaphore,
			Auth: &protos.Integration_Auth{
				Use: protos.Integration_AUTH_TYPE_TOKEN,
				Token: &protos.Integration_Auth_Token{
					ValueFrom: &protos.ValueFrom{
						Secret: &protos.ValueFromSecret{
							Name: canvasSecret.Name,
							Key:  "key",
						},
					},
				},
			},
		},
	}

	createdIntegration, err := CreateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createIntegrationSpec)
	require.NoError(t, err)

	t.Run("unauthenticated -> error", func(t *testing.T) {
		_, err := UpdateIntegration(context.Background(), r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Id, &protos.Integration{})
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

		_, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Id, integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration name is required", s.Message())
	})

	t.Run("integration not found", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "updated-name",
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: canvasSecret.Name,
								Key:  "key",
							},
						},
					},
				},
			},
		}

		_, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), "nonexistent-id", integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "integration nonexistent-id not found", s.Message())
	})

	t.Run("update integration by ID", func(t *testing.T) {
		newName := support.RandomName("updated-integration")
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: newName,
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: canvasSecret.Name,
								Key:  "key",
							},
						},
					},
				},
			},
		}

		response, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Id, integration)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, newName, response.Integration.Metadata.Name)
		assert.Equal(t, createdIntegration.Integration.Metadata.Id, response.Integration.Metadata.Id)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_CANVAS, response.Integration.Metadata.DomainType)
		assert.Equal(t, r.Canvas.ID.String(), response.Integration.Metadata.DomainId)
	})

	t.Run("update integration by name", func(t *testing.T) {
		newName := support.RandomName("updated-integration-name")
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: newName,
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								Name: canvasSecret.Name,
								Key:  "key",
							},
						},
					},
				},
			},
		}

		response, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Name, integration)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, newName, response.Integration.Metadata.Name)
		assert.Equal(t, createdIntegration.Integration.Metadata.Id, response.Integration.Metadata.Id)
	})

	t.Run("update canvas integration with organization secret", func(t *testing.T) {
		newName := support.RandomName("updated-integration-org-secret")
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: newName,
			},
			Spec: &protos.Integration_Spec{
				Type: models.IntegrationTypeSemaphore,
				Auth: &protos.Integration_Auth{
					Use: protos.Integration_AUTH_TYPE_TOKEN,
					Token: &protos.Integration_Auth_Token{
						ValueFrom: &protos.ValueFrom{
							Secret: &protos.ValueFromSecret{
								DomainType: authpb.DomainType_DOMAIN_TYPE_ORGANIZATION,
								Name:       orgSecret.Name,
								Key:        "key",
							},
						},
					},
				},
			},
		}

		response, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Id, integration)
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, newName, response.Integration.Metadata.Name)
		assert.Equal(t, createdIntegration.Integration.Metadata.Id, response.Integration.Metadata.Id)
	})

	t.Run("invalid secret", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test-invalid-secret",
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

		_, err := UpdateIntegration(ctx, r.Encryptor, r.Registry, models.DomainTypeCanvas, r.Canvas.ID.String(), createdIntegration.Integration.Metadata.Id, integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "error finding secret does-not-exist: record not found", s.Message())
	})
}