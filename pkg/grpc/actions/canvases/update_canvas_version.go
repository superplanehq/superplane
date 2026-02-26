package canvases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpdateCanvasVersion(
	ctx context.Context,
	registry *registry.Registry,
	organizationID string,
	canvasID string,
	versionID string,
	pbCanvas *pb.Canvas,
) (*pb.UpdateCanvasVersionResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	versionUUID, err := uuid.Parse(versionID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	userUUID := uuid.MustParse(userID)
	var version *models.CanvasVersion

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		version, err = models.FindCanvasVersionInTransaction(tx, canvasUUID, versionUUID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return status.Error(codes.NotFound, "version not found")
			}
			return err
		}

		if version.OwnerID == nil || *version.OwnerID != userUUID {
			return status.Error(codes.PermissionDenied, "version owner mismatch")
		}

		if version.IsPublished {
			return status.Error(codes.FailedPrecondition, "published versions are immutable")
		}

		now := time.Now()
		version.Nodes = datatypes.NewJSONSlice(nodes)
		version.Edges = datatypes.NewJSONSlice(edges)
		version.UpdatedAt = &now

		if err := tx.Save(version).Error; err != nil {
			return err
		}

		draft := models.CanvasUserDraft{
			WorkflowID: canvasUUID,
			UserID:     userUUID,
			VersionID:  version.ID,
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}

		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "workflow_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"version_id": version.ID,
				"updated_at": now,
			}),
		}).Create(&draft).Error
	})
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}
		return nil, status.Errorf(codes.Internal, "failed to update canvas version: %v", err)
	}

	return &pb.UpdateCanvasVersionResponse{
		Version: SerializeCanvasVersion(version, organizationID),
	}, nil
}
