package secrets

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func DescribeSecret(ctx context.Context, encryptor crypto.Encryptor, domainType, domainId, idOrName string) (*pb.DescribeSecretResponse, error) {
	err := actions.ValidateUUIDs(idOrName)
	var secret *models.Secret
	if err != nil {
		secret, err = models.FindSecretByName(domainType, uuid.MustParse(domainId), idOrName)
	} else {
		secret, err = models.FindSecretByID(domainType, uuid.MustParse(domainId), idOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "secret not found")
	}

	s, err := serializeSecret(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	return &pb.DescribeSecretResponse{
		Secret: s,
	}, nil
}

func serializeSecret(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) (*pb.Secret, error) {
	s := &pb.Secret{
		Metadata: &pb.Secret_Metadata{
			Id:         secret.ID.String(),
			Name:       secret.Name,
			DomainType: actions.DomainTypeToProto(secret.DomainType),
			DomainId:   secret.DomainID.String(),
			CreatedAt:  timestamppb.New(*secret.CreatedAt),
		},
		Spec: &pb.Secret_Spec{
			Provider: secretProviderToProto(secret.Provider),
		},
	}

	switch s.Spec.Provider {
	case pb.Secret_PROVIDER_LOCAL:
		local, err := serializeLocalSecretData(ctx, encryptor, secret)
		if err != nil {
			return nil, err
		}

		s.Spec.Local = local
		return s, nil

	default:
		return s, nil
	}
}

func serializeLocalSecretData(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) (*pb.Secret_Local, error) {
	data, err := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if err != nil {
		return nil, err
	}

	var values map[string]string
	err = json.Unmarshal(data, &values)
	if err != nil {
		return nil, err
	}

	local := &pb.Secret_Local{
		Data: map[string]string{},
	}

	//
	// We only show the keys, not the values.
	//
	for k := range values {
		local.Data[k] = "***"
	}

	return local, nil
}
