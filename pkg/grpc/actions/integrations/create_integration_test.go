package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"

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
		_, err := CreateIntegration(context.Background(), r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), &protos.Integration{})
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

		_, err := CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration name is required", s.Message())
	})

	t.Run("invalid integration type", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid integration type", s.Message())
	})

	t.Run("invalid secret", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: protos.Integration_TYPE_SEMAPHORE,
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

		_, err := CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
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
				Type: protos.Integration_TYPE_SEMAPHORE,
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

		_, err = CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "key nope not found in secret "+secret.Name, s.Message())
	})

	t.Run("integration is created", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: protos.Integration_TYPE_SEMAPHORE,
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

		response, err := CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
		require.NoError(t, err)
		assert.Equal(t, "test", response.Integration.Metadata.Name)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		integration := &protos.Integration{
			Metadata: &protos.Integration_Metadata{
				Name: "test",
			},
			Spec: &protos.Integration_Spec{
				Type: protos.Integration_TYPE_SEMAPHORE,
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

		_, err := CreateIntegration(ctx, r.Encryptor, models.DomainTypeCanvas, r.Canvas.ID.String(), integration)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
