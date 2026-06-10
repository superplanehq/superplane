package secrets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func SetSecretKey(ctx context.Context, encryptor crypto.Encryptor, domainType, domainID, idOrName, keyName, value string) (*pb.SetSecretKeyResponse, error) {
	if keyName == "" {
		return nil, status.Error(codes.InvalidArgument, "key name is required")
	}

	secret, err := findSecretInDomain(domainType, domainID, idOrName)
	if err != nil {
		return nil, err
	}

	data, err := decryptSecretData(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	if data == nil {
		data = make(map[string]string)
	}
	data[keyName] = value

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
	return &pb.SetSecretKeyResponse{Secret: s}, nil
}
