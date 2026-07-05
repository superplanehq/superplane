package secrets

import (
	"context"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	secretstore "github.com/superplanehq/superplane/pkg/secrets"
)

func UpdateSecret(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName string, spec *pb.Secret) (*pb.UpdateSecretResponse, error) {
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

	if spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "missing secret")
	}

	if spec.Metadata == nil || spec.Metadata.Name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "empty secret name")
	}

	if spec.Spec == nil {
		return nil, grpcerrors.InvalidArgument(nil, "missing secret spec")
	}

	provider := protoToSecretProvider(spec.Spec.Provider)
	if provider != secret.Provider {
		return nil, grpcerrors.InvalidArgument(nil, "cannot update provider")
	}

	if spec.Spec.Local == nil || spec.Spec.Local.Data == nil {
		return nil, grpcerrors.InvalidArgument(nil, "missing data")
	}

	data, err := secretstore.EncryptLocalData(ctx, encryptor, *secret, spec.Spec.Local.Data)
	if err != nil {
		return nil, err
	}

	secret, err = secret.UpdateData(data)
	if err != nil {
		return nil, err
	}

	s, err := serializeSecret(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	return &pb.UpdateSecretResponse{Secret: s}, nil
}
