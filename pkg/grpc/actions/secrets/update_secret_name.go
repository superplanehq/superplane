package secrets

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
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

	updated, err := secret.UpdateName(name)
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
