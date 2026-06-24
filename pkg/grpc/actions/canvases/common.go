package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"gorm.io/gorm"
)

const canvasNameAlreadyExistsMessage = "Canvas with the same name already exists"

func checkCanvasExistence(ctx context.Context, db *gorm.DB, orgID, canvasID uuid.UUID) error {
	return telemetry.RunSpan(ctx, "canvases.check_canvas_existence", func(ctx context.Context) error {
		exists, err := models.CheckCanvasExistence(db, orgID, canvasID)
		if err != nil {
			return err
		}
		if !exists {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func loadCanvas(ctx context.Context, db *gorm.DB, orgID, canvasID uuid.UUID) (*models.Canvas, error) {
	var canvas *models.Canvas
	err := telemetry.RunSpan(ctx, "canvases.find_canvas", func(ctx context.Context) error {
		var findErr error
		canvas, findErr = models.FindCanvasInTransaction(db, orgID, canvasID)
		return findErr
	})
	if err != nil {
		return nil, err
	}

	return canvas, nil
}

func loadLiveCanvasVersion(ctx context.Context, db *gorm.DB, canvas *models.Canvas) (*models.CanvasVersion, error) {
	var liveVersion *models.CanvasVersion
	err := telemetry.RunSpan(ctx, "canvases.load_live_version", func(ctx context.Context) error {
		var loadErr error
		liveVersion, loadErr = models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		return loadErr
	})
	if err != nil {
		return nil, err
	}

	return liveVersion, nil
}

func loadCanvasStatus(ctx context.Context, db *gorm.DB, canvasID uuid.UUID) (*pb.Canvas_Status, error) {
	var canvasStatus *pb.Canvas_Status
	err := telemetry.RunSpan(ctx, "canvases.load_status", func(ctx context.Context) error {
		lastExecutions, loadErr := models.FindLastExecutionPerNode(db, canvasID)
		if loadErr != nil {
			return loadErr
		}

		executionResources, loadErr := LoadNodeExecutionResources(db, lastExecutions)
		if loadErr != nil {
			return loadErr
		}

		serializedExecutions, loadErr := SerializeNodeExecutions(lastExecutions, executionResources)
		if loadErr != nil {
			return loadErr
		}

		lastEvents, loadErr := models.FindLastEventPerNode(db, canvasID)
		if loadErr != nil {
			return loadErr
		}

		serializedEvents, loadErr := SerializeCanvasEvents(lastEvents)
		if loadErr != nil {
			return loadErr
		}

		canvasStatus = &pb.Canvas_Status{
			LastExecutions: serializedExecutions,
			LastEvents:     serializedEvents,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return canvasStatus, nil
}

func ensureCanvasNameAvailableInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	name string,
) error {
	existingCanvas, err := models.FindCanvasByNameInTransaction(tx, name, organizationID)
	if err == nil && existingCanvas.ID != canvasID {
		return models.ErrCanvasNameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
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
