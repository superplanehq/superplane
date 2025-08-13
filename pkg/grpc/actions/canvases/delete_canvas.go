package canvases

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DeleteCanvas(ctx context.Context, orgID string, req *pb.DeleteCanvasRequest, authService authorization.Authorization) (*pb.DeleteCanvasResponse, error) {
	var canvas *models.Canvas
	err := actions.ValidateUUIDs(req.IdOrName)
	if err != nil {
		canvas, err = models.FindCanvasByName(req.IdOrName, uuid.MustParse(orgID))
	} else {
		canvas, err = models.FindCanvasByID(req.IdOrName, uuid.MustParse(orgID))
	}

	if err != nil {
		return nil, status.Error(codes.NotFound, "canvas not found")
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		err := canvas.DeleteInTransaction(tx)
		if err != nil {
			return status.Error(codes.Internal, "failed to delete canvas")
		}

		err = authService.DestroyCanvasRoles(canvas.ID.String())
		if err != nil {
			return status.Error(codes.Internal, "failed to destroy canvas roles")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &pb.DeleteCanvasResponse{}, nil
}
