package secrets

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteSecret(ctx context.Context, domainType, domainID, idOrName string) (*pb.DeleteSecretResponse, error) {
	secret, err := findSecretInDomain(domainType, domainID, idOrName)
	if err != nil {
		return nil, err
	}

	err = secret.Delete()
	if err != nil {
		return nil, status.Error(codes.Internal, "error deleting secret")
	}

	return &pb.DeleteSecretResponse{}, nil
}
