package secrets

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		return nil, status.Error(codes.InvalidArgument, "secret not found")
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
