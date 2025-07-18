package integrations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"

	protos "github.com/superplanehq/superplane/pkg/protos/superplane"
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
		_, err := CreateIntegration(context.Background(), r.Encryptor, &protos.CreateIntegrationRequest{})
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, s.Code())
		assert.Equal(t, "user not authenticated", s.Message())
	})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: uuid.New().String(),
		}

		_, err := CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "canvas not found", s.Message())
	})

	t.Run("missing name", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
				Metadata: &protos.Integration_Metadata{
					Name: "",
				},
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "integration name is required", s.Message())
	})

	t.Run("invalid integration type", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
				Metadata: &protos.Integration_Metadata{
					Name: "test",
				},
				Spec: &protos.Integration_Spec{},
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "invalid integration type", s.Message())
	})

	t.Run("invalid secret", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
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
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "error finding secret does-not-exist: record not found", s.Message())
	})

	t.Run("invalid secret key", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
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
			},
		}

		_, err = CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "key nope not found in secret "+secret.Name, s.Message())
	})

	t.Run("integration is created", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
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
			},
		}

		integration, err := CreateIntegration(ctx, r.Encryptor, req)
		require.NoError(t, err)
		assert.Equal(t, "test", integration.Integration.Metadata.Name)
	})

	t.Run("name already used -> error", func(t *testing.T) {
		req := &protos.CreateIntegrationRequest{
			CanvasIdOrName: r.Canvas.ID.String(),
			Integration: &protos.Integration{
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
			},
		}

		_, err := CreateIntegration(ctx, r.Encryptor, req)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Equal(t, "name already used", s.Message())
	})
}
