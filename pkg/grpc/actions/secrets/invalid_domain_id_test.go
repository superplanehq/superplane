package secrets

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	secretpb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__SecretsRejectInvalidDomainID(t *testing.T) {
	encryptor := &crypto.NoOpEncryptor{}
	ctx := authentication.SetUserIdInMetadata(context.Background(), uuid.NewString())
	secret := &secretpb.Secret{
		Metadata: &secretpb.Secret_Metadata{
			Name: "test",
		},
		Spec: &secretpb.Secret_Spec{
			Provider: secretpb.Secret_PROVIDER_LOCAL,
			Local: &secretpb.Secret_Local{
				Data: map[string]string{"token": "value"},
			},
		},
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "list",
			run: func() error {
				_, err := ListSecrets(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization")
				return err
			},
		},
		{
			name: "describe",
			run: func() error {
				_, err := DescribeSecret(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", "test")
				return err
			},
		},
		{
			name: "create",
			run: func() error {
				_, err := CreateSecret(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", secret)
				return err
			},
		},
		{
			name: "update",
			run: func() error {
				_, err := UpdateSecret(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", "test", secret)
				return err
			},
		},
		{
			name: "delete",
			run: func() error {
				_, err := DeleteSecret(ctx, models.DomainTypeOrganization, "renamed-organization", "test")
				return err
			},
		},
		{
			name: "set key",
			run: func() error {
				_, err := SetSecretKey(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", "test", "token", "value")
				return err
			},
		},
		{
			name: "delete key",
			run: func() error {
				_, err := DeleteSecretKey(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", "test", "token")
				return err
			},
		},
		{
			name: "update name",
			run: func() error {
				_, err := UpdateSecretName(ctx, encryptor, models.DomainTypeOrganization, "renamed-organization", "test", "new-name")
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.run()
			s, ok := status.FromError(err)
			assert.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, s.Code())
			assert.Equal(t, "invalid domain id", s.Message())
		})
	}
}
