package secrets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateSecretName(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName, name string) (*pb.UpdateSecretNameResponse, error) {
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	err := actions.ValidateUUIDs(idOrName)
	var secret *models.Secret
	if err != nil {
		secret, err = models.FindSecretByName(domainType, uuid.MustParse(domainID), idOrName)
	} else {
		secret, err = models.FindSecretByID(domainType, uuid.MustParse(domainID), idOrName)
	}
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "secret not found")
	}

	if secret.Name == name {
		s, err := serializeSecret(ctx, encryptor, *secret)
		if err != nil {
			return nil, err
		}
		return &pb.UpdateSecretNameResponse{Secret: s}, nil
	}

	updated, err := renameSecret(ctx, encryptor, secret, name)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	s, err := serializeSecret(ctx, encryptor, *updated)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSecretNameResponse{Secret: s}, nil
}

// renameSecret updates the secret's name. For providers that encrypt data with
// the secret name as additional authenticated data (e.g. LOCAL), the stored
// data must be re-encrypted with the new name so that subsequent reads can
// decrypt it.
func renameSecret(ctx context.Context, encryptor crypto.Encryptor, secret *models.Secret, name string) (*models.Secret, error) {
	if secret.Provider != secrets.ProviderLocal || len(secret.Data) == 0 {
		return secret.UpdateName(name)
	}

	plaintext, err := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if err != nil {
		return nil, err
	}

	reEncrypted, err := encryptor.Encrypt(ctx, plaintext, []byte(name))
	if err != nil {
		return nil, err
	}

	return secret.UpdateNameAndData(name, reEncrypted)
}
