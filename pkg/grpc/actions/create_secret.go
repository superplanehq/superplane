package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"github.com/superplanehq/superplane/pkg/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func CreateSecret(ctx context.Context, encryptor crypto.Encryptor, req *pb.CreateSecretRequest) (*pb.CreateSecretResponse, error) {
	err := ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	if req.Secret == nil {
		return nil, status.Error(codes.InvalidArgument, "missing secret")
	}

	if req.Secret.Metadata == nil || req.Secret.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "empty secret name")
	}

	if req.Secret.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "missing secret spec")
	}

	provider := protoToSecretProvider(req.Secret.Spec.Provider)
	if provider == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid provider")
	}

	err = ValidateUUIDs(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester ID")
	}

	data, err := prepareSecretData(ctx, encryptor, req.Secret)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secret, err := models.CreateSecret(req.Secret.Metadata.Name, provider, req.RequesterId, canvas.ID, data)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	s, err := serializeSecret(ctx, encryptor, *secret)
	if err != nil {
		return nil, err
	}

	return &pb.CreateSecretResponse{Secret: s}, nil
}

func protoToSecretProvider(provider pb.Secret_Provider) string {
	switch provider {
	case pb.Secret_PROVIDER_LOCAL:
		return secrets.ProviderLocal
	default:
		return ""
	}
}

func secretProviderToProto(provider string) pb.Secret_Provider {
	switch provider {
	case secrets.ProviderLocal:
		return pb.Secret_PROVIDER_LOCAL
	default:
		return pb.Secret_PROVIDER_UNKNOWN
	}
}

func prepareSecretData(ctx context.Context, encryptor crypto.Encryptor, secret *pb.Secret) ([]byte, error) {
	if secret.Spec == nil {
		return nil, fmt.Errorf("missing secret spec")
	}
	switch secret.Spec.Provider {
	case pb.Secret_PROVIDER_LOCAL:
		if secret.Spec.Local == nil || secret.Spec.Local.Data == nil {
			return nil, fmt.Errorf("missing data")
		}

		data, err := json.Marshal(secret.Spec.Local.Data)
		if err != nil {
			return nil, err
		}

		encrypted, err := encryptor.Encrypt(ctx, data, []byte(secret.Metadata.Name))
		if err != nil {
			return nil, err
		}

		return encrypted, nil

	default:
		return nil, fmt.Errorf("provider not supported")
	}
}
