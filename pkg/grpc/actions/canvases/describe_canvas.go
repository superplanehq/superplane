package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func DescribeCanvas(
	ctx context.Context,
	registry *registry.Registry,
	organizationID string,
	userID string,
	id string,
	draft bool,
) (*pb.DescribeCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			template, templateErr := models.FindCanvasTemplate(canvasID)
			if templateErr != nil {
				return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
			}
			canvas = template
		} else {
			return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
		}
	}

	var proto *pb.Canvas
	if !draft {
		proto, err = SerializeCanvas(canvas, true)
	} else {
		versioningEnabled, versioningErr := isCanvasVersioningEnabledForCanvas(canvas)
		if versioningErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to determine canvas versioning mode: %v", versioningErr)
		}
		if !versioningEnabled {
			return nil, status.Error(codes.FailedPrecondition, "canvas versioning is not enabled for this canvas; no draft version is available")
		}

		userUUID, parseErr := uuid.Parse(userID)
		if parseErr != nil {
			return nil, status.Error(codes.Internal, "failed to identify current user")
		}

		draftRef, draftErr := models.FindCanvasDraftInTransaction(database.Conn(), canvas.ID, userUUID)
		if draftErr != nil {
			if errors.Is(draftErr, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "draft version not found for current user")
			}
			return nil, status.Errorf(codes.Internal, "failed to load draft version: %v", draftErr)
		}

		version, versionErr := models.FindCanvasVersionInTransaction(database.Conn(), canvas.ID, draftRef.VersionID)
		if versionErr != nil {
			if errors.Is(versionErr, gorm.ErrRecordNotFound) {
				return nil, status.Error(codes.NotFound, "draft version not found for current user")
			}
			return nil, status.Errorf(codes.Internal, "failed to load draft version: %v", versionErr)
		}

		proto, err = SerializeCanvasFromVersion(canvas, version, true)
	}
	if err != nil {
		log.Errorf("failed to serialize canvas %s: %v", canvas.ID.String(), err)
		return nil, status.Error(codes.Internal, "failed to serialize workflow")
	}

	return &pb.DescribeCanvasResponse{
		Canvas: proto,
	}, nil
}
