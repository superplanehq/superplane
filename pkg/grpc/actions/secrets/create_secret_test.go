package secrets

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	authpb "github.com/superplanehq/superplane/pkg/protos/authorization"
	protos "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__CreateSecret(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	encryptor := &crypto.NoOpEncryptor{}

	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("unauthenticated user", func(t *testing.T) {
		secret := &protos.Secret{
			Metadata: &protos.Secret_Metadata{
				Name: "test",
			},
			Spec: &protos.Secret_Spec{
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		_, err := CreateSecret(context.Background(), encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), secret)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("canvas secret is created", func(t *testing.T) {
		secret := &protos.Secret{
			Metadata: &protos.Secret_Metadata{
				Name: support.RandomName("secret"),
			},
			Spec: &protos.Secret_Spec{
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		response, err := CreateSecret(ctx, encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), secret)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.NotEmpty(t, response.Secret.Metadata.Id)
		assert.NotEmpty(t, response.Secret.Metadata.CreatedAt)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_CANVAS, response.Secret.Metadata.DomainType)
		assert.Equal(t, r.Canvas.ID.String(), response.Secret.Metadata.DomainId)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, response.Secret.Spec.Provider)
		require.NotNil(t, response.Secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***"}, response.Secret.Spec.Local.Data)
	})

	t.Run("organization secret is created", func(t *testing.T) {
		secret := &protos.Secret{
			Metadata: &protos.Secret_Metadata{
				Name: support.RandomName("secret"),
			},
			Spec: &protos.Secret_Spec{
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		response, err := CreateSecret(ctx, encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Secret)
		assert.NotEmpty(t, response.Secret.Metadata.Id)
		assert.NotEmpty(t, response.Secret.Metadata.CreatedAt)
		assert.Equal(t, authpb.DomainType_DOMAIN_TYPE_ORGANIZATION, response.Secret.Metadata.DomainType)
		assert.Equal(t, r.Organization.ID.String(), response.Secret.Metadata.DomainId)
		assert.Equal(t, protos.Secret_PROVIDER_LOCAL, response.Secret.Spec.Provider)
		require.NotNil(t, response.Secret.Spec.Local)
		require.Equal(t, map[string]string{"test": "***"}, response.Secret.Spec.Local.Data)
	})

	t.Run("name already used", func(t *testing.T) {
		name := support.RandomName("secret")
		ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
		secret := &protos.Secret{
			Metadata: &protos.Secret_Metadata{
				Name: name,
			},
			Spec: &protos.Secret_Spec{
				Provider: protos.Secret_PROVIDER_LOCAL,
				Local: &protos.Secret_Local{
					Data: map[string]string{
						"test": "test",
					},
				},
			},
		}

		//
		// This works since there's no canvas secret with this name
		//
		_, err := CreateSecret(ctx, encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), secret)
		require.NoError(t, err)

		//
		// This also works since there's no organization secret with this name too.
		//
		_, err = CreateSecret(ctx, encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret)
		require.NoError(t, err)

		//
		// Name is already taken, so we cannot create another canvas secret with it.
		//
		_, err = CreateSecret(ctx, encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), secret)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())

		//
		// Same thing on the organization level.
		//
		_, err = CreateSecret(ctx, encryptor, models.DomainTypeOrganization, r.Organization.ID.String(), secret)
		s, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
