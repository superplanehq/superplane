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
)

const maskedValuePlaceholder = "***"

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

	data, err := prepareSecretDataForUpdate(ctx, encryptor, *secret, spec)
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

// prepareSecretDataForUpdate merges incoming spec with existing secret data: keys sent as "***"
// keep their existing value so the UI can update a single key without sending others.
func prepareSecretDataForUpdate(ctx context.Context, encryptor crypto.Encryptor, existing models.Secret, spec *pb.Secret) ([]byte, error) {
	if spec.Spec == nil || spec.Spec.Local == nil || spec.Spec.Local.Data == nil {
		return nil, status.Error(codes.InvalidArgument, "missing data")
	}
	incoming := spec.Spec.Local.Data

	existingDecrypted, err := encryptor.Decrypt(ctx, existing.Data, []byte(existing.Name))
	if err != nil {
		return nil, err
	}
	var merged map[string]string
	if err := json.Unmarshal(existingDecrypted, &merged); err != nil {
		merged = make(map[string]string)
	}
	if merged == nil {
		merged = make(map[string]string)
	}

	result := make(map[string]string)
	for k, v := range incoming {
		if v != maskedValuePlaceholder {
			result[k] = v
		} else if existingVal, ok := merged[k]; ok {
			result[k] = existingVal
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return encryptor.Encrypt(ctx, data, []byte(spec.Metadata.Name))
}
