package canvases

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

const canvasNameAlreadyExistsMessage = "Canvas with the same name already exists"

func checkCanvasExistence(ctx context.Context, db *gorm.DB, orgID, canvasID uuid.UUID) (err error) {
	ctx, done := telemetry.Span(ctx, "canvases.check_canvas_existence")
	defer done(&err)

	exists, err := models.CheckCanvasExistence(db.WithContext(ctx), orgID, canvasID)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func loadCanvas(ctx context.Context, db *gorm.DB, orgID, canvasID uuid.UUID) (canvas *models.Canvas, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.find_canvas")
	defer done(&err)

	return models.FindCanvasInTransaction(db.WithContext(ctx), orgID, canvasID)
}

func loadLiveCanvasVersion(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (liveVersion *models.CanvasVersion, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.load_live_version")
	defer done(&err)

	return models.FindLiveCanvasVersionByCanvasInTransaction(db.WithContext(ctx), canvas)
}

func resolveLiveCanvasVersionID(tx *gorm.DB, canvas *models.Canvas, requestedVersionID string) (uuid.UUID, error) {
	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, grpcerrors.NotFound(err, "canvas live version not found")
		}
		return uuid.Nil, err
	}

	requestedVersionID = strings.TrimSpace(requestedVersionID)
	if requestedVersionID == "" {
		return liveVersion.ID, nil
	}

	requestedUUID, err := uuid.Parse(requestedVersionID)
	if err != nil {
		return uuid.Nil, grpcerrors.InvalidArgument(err, "invalid version id")
	}

	if requestedUUID != liveVersion.ID {
		return uuid.Nil, grpcerrors.InvalidArgument(nil, "version_id is not the current live version")
	}

	return liveVersion.ID, nil
}

func loadCanvasStatus(ctx context.Context, db *gorm.DB, canvasID uuid.UUID) (canvasStatus *pb.Canvas_Status, err error) {
	ctx, done := telemetry.Span(ctx, "canvases.load_status")
	defer done(&err)

	lastExecutions, err := models.FindLastExecutionPerNode(db.WithContext(ctx), canvasID)
	if err != nil {
		return nil, err
	}

	executionResources, err := LoadNodeExecutionResources(db.WithContext(ctx), lastExecutions)
	if err != nil {
		return nil, err
	}

	serializedExecutions, err := SerializeNodeExecutions(lastExecutions, executionResources)
	if err != nil {
		return nil, err
	}

	lastEvents, err := models.FindLastEventPerNode(db.WithContext(ctx), canvasID)
	if err != nil {
		return nil, err
	}

	serializedEvents, err := SerializeCanvasEvents(lastEvents)
	if err != nil {
		return nil, err
	}

	return &pb.Canvas_Status{
		LastExecutions: serializedExecutions,
		LastEvents:     serializedEvents,
	}, nil
}

func mapCanvasNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	err = models.MapCanvasNameUniqueConstraintError(err)
	if errors.Is(err, models.ErrCanvasNameAlreadyExists) {
		return grpcerrors.AlreadyExists(err, canvasNameAlreadyExistsMessage)
	}

	return err
}

func publishCanvasVersionInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	liveVersion *models.CanvasVersion,
	nextVersion *models.CanvasVersion,
	options changesets.CanvasPublisherOptions,
) error {
	changeset, err := changesets.NewChangesetBuilder(
		liveVersion.Nodes,
		liveVersion.Edges,
		nextVersion.Nodes,
		nextVersion.Edges,
	).Build()
	if err != nil {
		return err
	}

	if len(changeset.Changes) == 0 {
		return mapCanvasNameUniqueConstraintError(
			models.PromoteToLiveInTransaction(tx, nextVersion, nextVersion.Nodes, nextVersion.Edges),
		)
	}

	publisher, err := changesets.NewCanvasPublisher(tx, nextVersion, liveVersion, options)
	if err != nil {
		return err
	}

	return mapCanvasNameUniqueConstraintError(publisher.Publish(ctx))
}
