package secrets

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	secretstore "github.com/superplanehq/superplane/pkg/secrets"
)

func CreateSecret(ctx context.Context, encryptor crypto.Encryptor, domainType string, domainID string, spec *pb.Secret) (*pb.CreateSecretResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
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
	if provider == "" {
		return nil, grpcerrors.InvalidArgument(nil, "invalid provider")
	}

	localData, err := prepareSecretData(spec)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid secret configuration")
	}

	secretID := uuid.New()
	data, err := secretstore.EncryptLocalData(ctx, encryptor, models.Secret{
		ID:   secretID,
		Name: spec.Metadata.Name,
	}, localData)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid secret configuration")
	}

	secret, err := models.CreateSecretWithID(secretID, spec.Metadata.Name, provider, userID, domainType, uuid.MustParse(domainID), data)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, grpcerrors.InvalidArgument(err, "name already used")
		}

		log.Errorf("failed to create secret %s: %v", spec.Metadata.Name, err)
		return nil, grpcerrors.Internal(err, "failed to create secret")
	}

	s, err := serializeSecret(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	return &pb.CreateSecretResponse{Secret: s}, nil
}

func protoToSecretProvider(provider pb.Secret_Provider) string {
	switch provider {
	case pb.Secret_PROVIDER_LOCAL:
		return secretstore.ProviderLocal
	default:
		return ""
	}
}

func secretProviderToProto(provider string) pb.Secret_Provider {
	switch provider {
	case secretstore.ProviderLocal:
		return pb.Secret_PROVIDER_LOCAL
	default:
		return pb.Secret_PROVIDER_UNKNOWN
	}
}

func prepareSecretData(secret *pb.Secret) (map[string]string, error) {
	if secret.Spec == nil {
		return nil, fmt.Errorf("missing secret spec")
	}
	switch secret.Spec.Provider {
	case pb.Secret_PROVIDER_LOCAL:
		if secret.Spec.Local == nil || secret.Spec.Local.Data == nil {
			return nil, fmt.Errorf("missing data")
		}

		return secret.Spec.Local.Data, nil

	default:
		return nil, fmt.Errorf("provider not supported")
	}
}

// decryptSecretData decrypts a secret's stored data and returns the key-value map.
func decryptSecretData(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) (map[string]string, error) {
	return secretstore.DecryptLocalData(ctx, encryptor, secret)
}
