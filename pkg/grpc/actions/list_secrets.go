package actions

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListSecrets(ctx context.Context, encryptor crypto.Encryptor, req *pb.ListSecretsRequest) (*pb.ListSecretsResponse, error) {
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

	secrets, err := models.ListSecrets(canvas.ID.String())
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
