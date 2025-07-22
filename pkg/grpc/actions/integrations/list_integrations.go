package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
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

	integrations, err := models.ListIntegrations(models.DomainCanvas, canvas.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list integrations")
	}

	return &pb.ListIntegrationsResponse{
		Integrations: serializeIntegrations(integrations),
	}, nil
}

func serializeIntegrations(integrations []*models.Integration) []*pb.Integration {
	out := []*pb.Integration{}
	for _, integration := range integrations {
		out = append(out, serializeIntegration(*integration))
	}
	return out
}
