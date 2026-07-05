package secrets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	secretstore "github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

func UpdateSecretName(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName, name string) (*pb.UpdateSecretNameResponse, error) {
	if name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "name is required")
	}

	err := actions.ValidateUUIDs(idOrName)
	var secret *models.Secret
	if err != nil {
		secret, err = models.FindSecretByName(domainType, uuid.MustParse(domainID), idOrName)
	} else {
		secret, err = models.FindSecretByID(domainType, uuid.MustParse(domainID), idOrName)
	}
	if err != nil {
		return nil, grpcerrors.InvalidArgument(nil, "secret not found")
	}

	if secret.Name == name {
		s, err := serializeSecret(ctx, encryptor, *secret)
		if err != nil {
			return nil, err
		}
		return &pb.UpdateSecretNameResponse{Secret: s}, nil
	}

	var reEncrypted []byte
	if len(secret.Data) > 0 {
		plainData, err := decryptSecretData(ctx, encryptor, *secret)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to decrypt secret data for re-encryption")
		}
		reEncrypted, err = secretstore.EncryptLocalData(ctx, encryptor, *secret, plainData)
		if err != nil {
			return nil, grpcerrors.Internal(err, "failed to re-encrypt secret data with new name")
		}
	}

	var updated *models.Secret
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		var txErr error
		updated, txErr = secret.UpdateNameAndData(tx, name, reEncrypted)
		return txErr
	})
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, grpcerrors.InvalidArgument(err, "invalid secret name")
		}
		return nil, grpcerrors.Internal(err, "failed to update secret name")
	}

	s, err := serializeSecret(ctx, encryptor, *updated)
	if err != nil {
		return nil, err
	}
	return &pb.UpdateSecretNameResponse{Secret: s}, nil
}
