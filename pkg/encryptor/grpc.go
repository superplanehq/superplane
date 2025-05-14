package encryptor

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/superplanehq/superplane/pkg/protos/encryptor"
)

type GrpcEncryptor struct {
	endpoint string
}

func NewGrpcEncryptor(endpoint string) *GrpcEncryptor {
	return &GrpcEncryptor{
		endpoint: endpoint,
	}
}

func (e *GrpcEncryptor) Encrypt(ctx context.Context, data []byte, associatedData []byte) ([]byte, error) {
	conn, err := grpc.NewClient(e.endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	client := pb.NewEncryptorClient(conn)
	req := pb.EncryptRequest{
		Raw:            data,
		AssociatedData: associatedData,
	}

	res, err := client.Encrypt(ctx, &req)
	if err != nil {
		return nil, err
	}

	return res.Cypher, nil
}

func (e *GrpcEncryptor) Decrypt(ctx context.Context, cypher []byte, associatedData []byte) ([]byte, error) {
	conn, err := grpc.NewClient(e.endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	client := pb.NewEncryptorClient(conn)
	req := pb.DecryptRequest{
		Cypher:         cypher,
		AssociatedData: associatedData,
	}

	res, err := client.Decrypt(ctx, &req)
	if err != nil {
		return nil, err
	}

	return res.Raw, nil
}
