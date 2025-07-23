package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"github.com/superplanehq/superplane/pkg/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateSecret(ctx context.Context, encryptor crypto.Encryptor, domainType string, domainID string, spec *pb.Secret) (*pb.CreateSecretResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if spec == nil {
		return nil, status.Error(codes.InvalidArgument, "missing secret")
	}

	if spec.Metadata == nil || spec.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "empty secret name")
	}

	if spec.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "missing secret spec")
	}

	provider := protoToSecretProvider(spec.Spec.Provider)
	if provider == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid provider")
	}

	data, err := prepareSecretData(ctx, encryptor, spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secret, err := models.CreateSecret(spec.Metadata.Name, provider, userID, domainType, uuid.MustParse(domainID), data)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
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
		return secrets.ProviderLocal
	default:
		return ""
	}
}

func secretProviderToProto(provider string) pb.Secret_Provider {
	switch provider {
	case secrets.ProviderLocal:
		return pb.Secret_PROVIDER_LOCAL
	default:
		return pb.Secret_PROVIDER_UNKNOWN
	}
}

func prepareSecretData(ctx context.Context, encryptor crypto.Encryptor, secret *pb.Secret) ([]byte, error) {
	if secret.Spec == nil {
		return nil, fmt.Errorf("missing secret spec")
	}
	switch secret.Spec.Provider {
	case pb.Secret_PROVIDER_LOCAL:
		if secret.Spec.Local == nil || secret.Spec.Local.Data == nil {
			return nil, fmt.Errorf("missing data")
		}

		data, err := json.Marshal(secret.Spec.Local.Data)
		if err != nil {
			return nil, err
		}

		encrypted, err := encryptor.Encrypt(ctx, data, []byte(secret.Metadata.Name))
		if err != nil {
			return nil, err
		}

		return encrypted, nil

	default:
		return nil, fmt.Errorf("provider not supported")
	}
}
