package secrets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UpdateSecret(ctx context.Context, encryptor crypto.Encryptor, req *pb.UpdateSecretRequest) (*pb.UpdateSecretResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester ID")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var secret *models.Secret
	if err != nil {
		secret, err = models.FindSecretByName(canvas.ID.String(), req.IdOrName)
	} else {
		secret, err = models.FindSecretByID(canvas.ID.String(), req.IdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "secret not found")
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
	if provider != secret.Provider {
		return nil, status.Error(codes.InvalidArgument, "cannot update provider")
	}

	data, err := prepareSecretData(ctx, encryptor, req.Secret)
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
