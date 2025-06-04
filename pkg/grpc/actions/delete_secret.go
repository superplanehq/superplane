package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
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

	err = ValidateUUIDs(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid requester ID")
	}

	err = ValidateUUIDs(req.IdOrName)
	var secret *models.Secret
	if err != nil {
		secret, err = models.FindSecretByName(canvas.ID.String(), req.IdOrName)
	} else {
		secret, err = models.FindSecretByID(canvas.ID.String(), req.IdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "secret not found")
	}

	err = secret.Delete()
	if err != nil {
		return nil, status.Error(codes.Internal, "error deleting secret")
	}

	return &pb.DeleteSecretResponse{}, nil
}
