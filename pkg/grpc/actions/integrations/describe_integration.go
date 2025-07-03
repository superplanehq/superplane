package integrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func DescribeIntegration(ctx context.Context, req *pb.DescribeIntegrationRequest) (*pb.DescribeIntegrationResponse, error) {
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

	err = actions.ValidateUUIDs(req.IdOrName)
	var integration *models.Integration
	if err != nil {
		integration, err = models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, req.IdOrName)
	} else {
		integration, err = models.FindIntegrationByID(authorization.DomainCanvas, canvas.ID, uuid.MustParse(req.IdOrName))
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "integration not found")
	}

	response := &pb.DescribeIntegrationResponse{
		Integration: serializeIntegration(*integration),
	}

	return response, nil
}
