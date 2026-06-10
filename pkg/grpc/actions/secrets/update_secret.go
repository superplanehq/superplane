package secrets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateSecret(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName string, spec *pb.Secret) (*pb.UpdateSecretResponse, error) {
	secret, err := findSecretInDomain(domainType, domainID, idOrName)
	if err != nil {
		return nil, err
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
	if provider != secret.Provider {
		return nil, status.Error(codes.InvalidArgument, "cannot update provider")
	}

	data, err := prepareSecretData(ctx, encryptor, spec)
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
