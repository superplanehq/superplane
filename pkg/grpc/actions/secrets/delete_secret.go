package secrets

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
)

func DeleteSecret(ctx context.Context, domainType, domainID, idOrName string) (*pb.DeleteSecretResponse, error) {
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

	err = secret.Delete()
	if err != nil {
		return nil, grpcerrors.Internal(err, "error deleting secret")
	}

	return &pb.DeleteSecretResponse{}, nil
}
