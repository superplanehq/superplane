package canvases

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
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

func promoteMainCanvasVersionInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	canvas *models.Canvas,
	previousLive *models.CanvasVersion,
	nextVersion *models.CanvasVersion,
	options changesets.CanvasPublisherOptions,
) error {
	liveVersion := previousLive
	if liveVersion == nil {
		liveVersion = &models.CanvasVersion{WorkflowID: canvas.ID}
	}
	if nextVersion.ID == liveVersion.ID {
		return nil
	}
	return publishCanvasVersionInTransaction(ctx, tx, liveVersion, nextVersion, options)
}

func canvasPublisherOptions(
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	gitProvider gitprovider.Provider,
	orgID uuid.UUID,
	authService authorization.Authorization,
	webhookBaseURL string,
) changesets.CanvasPublisherOptions {
	return changesets.CanvasPublisherOptions{
		Registry:       registry,
		GitProvider:    gitProvider,
		OrgID:          orgID,
		Encryptor:      encryptor,
		AuthService:    authService,
		WebhookBaseURL: webhookBaseURL,
	}
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
		outOfSync, syncErr := canvasRuntimeNodesOutOfSync(tx, nextVersion)
		if syncErr != nil {
			return syncErr
		}
		if outOfSync {
			syntheticLive := &models.CanvasVersion{
				WorkflowID: nextVersion.WorkflowID,
				Nodes:      append([]models.Node(nil), liveVersion.Nodes...),
				Edges:      append([]models.Edge(nil), liveVersion.Edges...),
			}
			if len(syntheticLive.Nodes) == len(nextVersion.Nodes) {
				syntheticLive.Nodes = nil
				syntheticLive.Edges = nil
			}
			publisher, publisherErr := changesets.NewCanvasPublisher(tx, nextVersion, syntheticLive, options)
			if publisherErr != nil {
				return publisherErr
			}
			return mapCanvasNameUniqueConstraintError(publisher.Publish(ctx))
		}

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

func canvasRuntimeNodesOutOfSync(tx *gorm.DB, version *models.CanvasVersion) (bool, error) {
	runtimeNodes, err := models.FindCanvasNodesInTransaction(tx, version.WorkflowID)
	if err != nil {
		return false, err
	}

	runtimeIDs := make(map[string]struct{}, len(runtimeNodes))
	for _, node := range runtimeNodes {
		runtimeIDs[node.NodeID] = struct{}{}
	}

	for _, node := range version.Nodes {
		if _, ok := runtimeIDs[node.ID]; !ok {
			return true, nil
		}
	}

	return len(runtimeIDs) != len(version.Nodes), nil
}

func repairCanvasRuntimeNodesIfOutOfSync(
	ctx context.Context,
	canvas *models.Canvas,
	options changesets.CanvasPublisherOptions,
) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
		if err != nil {
			return err
		}

		outOfSync, err := canvasRuntimeNodesOutOfSync(tx, liveVersion)
		if err != nil || !outOfSync {
			return err
		}

		return promoteMainCanvasVersionInTransaction(
			ctx,
			tx,
			canvas,
			&models.CanvasVersion{WorkflowID: canvas.ID},
			liveVersion,
			options,
		)
	})
}
