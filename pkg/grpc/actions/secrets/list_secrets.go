package secrets

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

func ListSecrets(ctx context.Context, encryptor crypto.Encryptor, domainType string, domainID uuid.UUID) (*pb.ListSecretsResponse, error) {
	secrets, err := models.ListSecrets(domainType, domainID)
	if err != nil {
		return nil, err
	}

	s, err := serializeSecrets(ctx, encryptor, secrets)
	if err != nil {
		return nil, err
	}

	return &pb.ListSecretsResponse{
		Secrets: s,
	}, nil
}

func serializeSecrets(ctx context.Context, encryptor crypto.Encryptor, secrets []models.Secret) ([]*pb.Secret, error) {
	out := []*pb.Secret{}

	for _, s := range secrets {
		secret, err := serializeSecret(ctx, encryptor, s)
		if err != nil {
			return nil, err
		}

		out = append(out, secret)
	}

	return out, nil
}
