package canvases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateCanvas(
	ctx context.Context,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	gitProvider git.Provider,
	webhookBaseURL string,
	organizationID uuid.UUID,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	usageService usage.Service,
) (*pb.CreateCanvasResponse, error) {
	return CreateCanvasWithSeedFiles(
		ctx,
		registry,
		encryptor,
		authService,
		gitProvider,
		webhookBaseURL,
		organizationID,
		pbCanvas,
		autoLayout,
		usageService,
		nil,
	)
}

// CreateCanvasWithSeedFiles is the variant called by the app install flow. It
// persists the provided files alongside the canvas's pending repository row so
// the repository provisioner can commit them as the repo's initial content. A
// nil/empty seedFiles slice is equivalent to calling CreateCanvas.
func CreateCanvasWithSeedFiles(
	ctx context.Context,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
	gitProvider git.Provider,
	webhookBaseURL string,
	organizationID uuid.UUID,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
	usageService usage.Service,
	seedFiles []models.RepositorySeedFile,
) (*pb.CreateCanvasResponse, error) {
	if pbCanvas == nil {
		return nil, grpcerrors.InvalidArgument(nil, "canvas is required")
	}

	if pbCanvas.GetMetadata() == nil {
		return nil, grpcerrors.InvalidArgument(nil, "canvas metadata is required")
	}

	name := strings.TrimSpace(pbCanvas.GetMetadata().GetName())
	if name == "" {
		return nil, grpcerrors.InvalidArgument(nil, "canvas name is required")
	}
	pbCanvas.Metadata.Name = name

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID.String(), pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "failed to apply layout")
	}

	createdBy := uuid.MustParse(userID)
	canvasCount, err := models.CountCanvasesByOrganization(organizationID.String())
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to count organization canvases")
	}

	err = usage.EnsureOrganizationWithinLimits(
		ctx,
		usageService,
		organizationID.String(),
		&usagepb.OrganizationState{Canvases: int32(canvasCount + 1)},
		&usagepb.CanvasState{Nodes: int32(len(nodes))},
	)

	if err != nil {
		return nil, err
	}

	canvasID := uuid.New()
	versionID := uuid.New()

	now := time.Now()
	canvas := models.Canvas{
		ID:             canvasID,
		OrganizationID: organizationID,
		LiveVersionID:  &versionID,
		Name:           name,
		Description:    pbCanvas.Metadata.Description,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		findErr := ensureCanvasNameAvailableInTransaction(tx, organizationID, canvasID, name)
		if errors.Is(findErr, models.ErrCanvasNameAlreadyExists) {
			return grpcerrors.AlreadyExists(nil, "Canvas with the same name already exists")
		}
		if findErr != nil {
			return findErr
		}

		//
		// Create the workflow record
		//
		err := tx.Clauses(clause.Returning{}).Create(&canvas).Error
		if err != nil {
			return mapCanvasNameUniqueConstraintError(err)
		}

		//
		// Create initial commit on main branch
		//
		mainBranch, err := models.CreateWorkflowBranch(tx, canvasID, models.CanvasGitBranchMain, nil)
		if err != nil {
			return err
		}

		emptyVersion := models.CanvasVersion{
			ID:            versionID,
			WorkflowID:    canvasID,
			BranchID:      mainBranch.ID,
			OwnerID:       &createdBy,
			Nodes:         datatypes.NewJSONSlice([]models.Node{}),
			Edges:         datatypes.NewJSONSlice([]models.Edge{}),
			GitBranch:     models.CanvasGitBranchMain,
			CommitMessage: "Initial commit",
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}

		if err := tx.Create(&emptyVersion).Error; err != nil {
			return err
		}

		if err := models.UpdateWorkflowBranchHead(tx, mainBranch.ID, versionID); err != nil {
			return err
		}

		repository, err := canvas.CreatePendingRepositoryInTransaction(tx, gitProvider.Name(), gitProvider.GetRepositoryID(git.RepositoryOptions{
			OrganizationID: organizationID,
			CanvasID:       canvasID,
		}))

		if err != nil {
			return err
		}

		if len(seedFiles) > 0 {
			if err := models.CreateRepositorySeedFilesInTransaction(tx, repository.ID, seedFiles); err != nil {
				return err
			}
		}

		//
		// If this is a canvas creation with no nodes,
		// nothing else to do here.
		//
		if len(nodes) == 0 {
			return nil
		}

		//
		// Otherwise. we generate and apply changeset to the draft version
		//
		changeset, err := changesets.NewChangesetBuilder([]models.Node{}, []models.Edge{}, nodes, edges).Build()
		if err != nil {
			return err
		}

		patcher := changesets.NewCanvasPatcher(tx, organizationID, registry, &emptyVersion)
		if err := patcher.ApplyChangeset(changeset, nil); err != nil {
			return err
		}

		updatedVersion := patcher.GetVersion()
		if err := tx.Save(updatedVersion).Error; err != nil {
			return err
		}

		//
		// Publish the draft version as the live version
		//
		publisher, err := changesets.NewCanvasPublisher(tx, updatedVersion, &emptyVersion, changesets.CanvasPublisherOptions{
			Registry:       registry,
			OrgID:          organizationID,
			Encryptor:      encryptor,
			AuthService:    authService,
			WebhookBaseURL: webhookBaseURL,
			GitProvider:    gitProvider,
		})

		if err != nil {
			return err
		}

		return publisher.Publish(ctx)
	})

	if err != nil {
		return nil, err
	}

	if publishErr := messages.NewCanvasCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishCreated(); publishErr != nil {
		log.Errorf("failed to publish canvas created RabbitMQ message: %v", publishErr)
	}

	canvas.Description = pbCanvas.Metadata.Description

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	liveVersion, err := models.FindLiveCanvasVersionByCanvasInTransaction(database.DB(ctx), &canvas)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to load canvas spec")
	}

	proto, err := SerializeCanvas(&canvas, liveVersion, user, nil)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
