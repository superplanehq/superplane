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

func DeleteSecretKey(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName, keyName string) (*pb.DeleteSecretKeyResponse, error) {
	if keyName == "" {
		return nil, status.Error(codes.InvalidArgument, "key name is required")
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

	data, err := decryptSecretData(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	if _, ok := data[keyName]; !ok {
		return nil, status.Error(codes.InvalidArgument, "key not found")
	}
	delete(data, keyName)
	if len(data) == 0 {
		return nil, status.Error(codes.InvalidArgument, "secret must have at least one key")
	}

	encrypted, err := encryptSecretData(ctx, encryptor, secret.Name, data)
	if err != nil {
		return nil, err
	}

	updated, err := secret.UpdateData(encrypted)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	updated.Data = encrypted

	s, err := serializeSecret(ctx, encryptor, *updated)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteSecretKeyResponse{Secret: s}, nil
}
