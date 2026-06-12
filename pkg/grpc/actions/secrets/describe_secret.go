package secrets

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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
	local := &pb.Secret_Local{
		Data: map[string]string{},
	}

	if len(secret.Data) == 0 {
		return local, nil
	}

	data, err := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if err != nil {
		//
		// A failure to decrypt one secret's payload (for example because of an
		// AAD mismatch caused by an older bug that renamed a secret without
		// re-encrypting its data) must not take down endpoints that fan-out
		// over many secrets, like ListSecrets. Log and degrade gracefully by
		// returning an empty key set instead of bubbling up an Internal error.
		//
		log.WithError(err).WithFields(log.Fields{
			"secret_id":   secret.ID.String(),
			"secret_name": secret.Name,
			"domain_id":   secret.DomainID.String(),
			"domain_type": secret.DomainType,
		}).Warn("failed to decrypt secret payload; returning empty key set")
		return local, nil
	}

	if len(data) == 0 {
		return local, nil
	}

	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		log.WithError(err).WithFields(log.Fields{
			"secret_id":   secret.ID.String(),
			"secret_name": secret.Name,
			"domain_id":   secret.DomainID.String(),
			"domain_type": secret.DomainType,
		}).Warn("failed to parse secret payload; returning empty key set")
		return local, nil
	}

	//
	// We only show the keys, not the values.
	//
	for k := range values {
		local.Data[k] = "***"
	}

	return local, nil
}
