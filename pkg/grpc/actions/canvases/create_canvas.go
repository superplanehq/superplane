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
	"github.com/superplanehq/superplane/pkg/canvas/materialize"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	git "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	if pbCanvas == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas is required")
	}

	if pbCanvas.GetMetadata() == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas metadata is required")
	}

	name := strings.TrimSpace(pbCanvas.GetMetadata().GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas name is required")
	}
	pbCanvas.Metadata.Name = name

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	nodes, edges, err := ParseCanvas(registry, organizationID.String(), pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", err)
	}

	createdBy := uuid.MustParse(userID)
	organizationChangeManagementEnabled, err := models.IsChangeManagementEnabled(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load organization change management setting: %v", err)
	}

	changeManagementEnabled := organizationChangeManagementEnabled
	changeRequestApprovers := models.DefaultCanvasChangeRequestApprovers()
	if changeManagement := pbCanvas.GetSpec().GetChangeManagement(); changeManagement != nil {
		changeManagementEnabled = changeManagement.Enabled

		approvers, approversErr := parseAndValidateCanvasChangeRequestApprovers(
			authService,
			organizationID.String(),
			changeManagement,
		)
		if approversErr != nil {
			return nil, approversErr
		}
		if approvers != nil {
			changeRequestApprovers = approvers
		}
	}

	canvasCount, err := models.CountCanvasesByOrganization(organizationID.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count organization canvases: %v", err)
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
	liveVersionID := uuid.New()
	now := time.Now()
	canvas := models.Canvas{
		ID:             canvasID,
		OrganizationID: organizationID,
		Name:           name,
		LiveVersionID:  &liveVersionID,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		findErr := ensureCanvasNameAvailableInTransaction(tx, organizationID, canvasID, name)
		if errors.Is(findErr, models.ErrCanvasNameAlreadyExists) {
			return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
		}
		return findErr
	})
	if err != nil {
		return nil, err
	}

	repoID := gitProvider.GetRepositoryID(git.RepositoryOptions{
		OrganizationID: organizationID,
		CanvasID:       canvasID,
	})
	repository := &models.Repository{
		CanvasID:       canvasID,
		OrganizationID: organizationID,
		Provider:       gitProvider.Name(),
		RepoID:         repoID,
		Status:         models.RepositoryStatusPending,
	}

	user, err := models.FindActiveUserByID(organizationID.String(), userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find user: %v", err)
	}

	commitSHA, seedErr := materialize.SeedMainRepository(ctx, gitProvider, repository, materialize.SeedRepositoryInput{
		Canvas: &materialize.CanvasYAML{
			APIVersion: "v1",
			Kind:       "Canvas",
			Metadata: materialize.CanvasYAMLMetadata{
				Name:        name,
				Description: pbCanvas.Metadata.Description,
			},
			Spec: materialize.CanvasYAMLSpec{
				Nodes:                   nodes,
				Edges:                   edges,
				ChangeManagementEnabled: changeManagementEnabled,
				ChangeRequestApprovers:  changeRequestApprovers,
			},
		},
		Author: git.CommitAuthor{
			Name:  user.Name,
			Email: user.GetEmail(),
		},
	})
	if seedErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to seed canvas repository: %v", seedErr)
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Clauses(clause.Returning{}).Create(&canvas).Error; createErr != nil {
			return mapCanvasNameUniqueConstraintError(createErr)
		}

		// The placeholder live version is intentionally created empty (no nodes/
		// edges). It only exists to satisfy the non-null workflows.live_version_id
		// FK before materialization runs. Materializing the seeded git commit then
		// diffs this empty live against the snapshot, so every node is treated as an
		// add and the publisher creates the corresponding workflow_nodes records.
		// It must stay empty for the worker, so the optimistic response below is
		// built from the parsed spec rather than from this row.
		placeholderVersion := models.CanvasVersion{
			ID:                      liveVersionID,
			WorkflowID:              canvasID,
			State:                   models.CanvasVersionStatePublished,
			Name:                    name,
			Description:             pbCanvas.Metadata.Description,
			ChangeManagementEnabled: changeManagementEnabled,
			ChangeRequestApprovers:  datatypes.NewJSONSlice(changeRequestApprovers),
			Nodes:                   datatypes.NewJSONSlice([]models.Node{}),
			Edges:                   datatypes.NewJSONSlice([]models.Edge{}),
			CommitSHA:               commitSHA,
			GitBranch:               models.CanvasGitBranchMain,
			MaterializationStatus:   models.MaterializationStatusPending,
			PublishedAt:             &now,
			CreatedAt:               &now,
			UpdatedAt:               &now,
		}
		if versionErr := tx.Create(&placeholderVersion).Error; versionErr != nil {
			return versionErr
		}

		if repoErr := canvas.CreatePendingRepositoryInTransaction(tx, gitProvider.Name(), repoID); repoErr != nil {
			return repoErr
		}

		return tx.Model(&models.Repository{}).
			Where("canvas_id = ?", canvasID).
			Updates(map[string]any{
				"status":     models.RepositoryStatusReady,
				"updated_at": time.Now(),
			}).Error
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create canvas: %v", err)
	}

	// Worker-authoritative materialization: git main holds the seeded commit and
	// the canvas row exists, so the materializer worker (inline in tests) loads the
	// snapshot, creates workflow_nodes and webhooks, and promotes the live version.
	if err := materialize.RequestBranchMaterialization(ctx, canvasID, models.CanvasGitBranchMain, commitSHA, &createdBy); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to request canvas materialization: %v", err)
	}

	canvas.ChangeManagementEnabled = changeManagementEnabled
	canvas.ChangeRequestApprovers = datatypes.NewJSONSlice(changeRequestApprovers)
	canvas.Description = pbCanvas.Metadata.Description

	if publishErr := messages.NewCanvasCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishCreated(); publishErr != nil {
		log.Errorf("failed to publish canvas created RabbitMQ message: %v", publishErr)
	}

	var createdByUser *models.User
	if canvas.CreatedBy != nil {
		createdByUser, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	var createdByRef *pb.UserRef
	if createdByUser != nil {
		createdByRef = &pb.UserRef{Id: createdByUser.ID.String(), Name: createdByUser.Name}
	}

	canvasFolderID := ""
	if canvas.CanvasFolderID != nil {
		canvasFolderID = canvas.CanvasFolderID.String()
	}

	// Optimistic response: the live projection is materialized asynchronously, so
	// the canvas spec is echoed from the just-committed (parsed + laid-out) nodes
	// and edges rather than read back from the still-empty placeholder live row.
	return &pb.CreateCanvasResponse{
		Canvas: &pb.Canvas{
			Metadata: &pb.Canvas_Metadata{
				Id:             canvas.ID.String(),
				OrganizationId: canvas.OrganizationID.String(),
				Name:           name,
				Description:    pbCanvas.Metadata.Description,
				CreatedAt:      timestamppb.New(*canvas.CreatedAt),
				UpdatedAt:      timestamppb.New(*canvas.UpdatedAt),
				CreatedBy:      createdByRef,
				FolderId:       canvasFolderID,
			},
			Spec: &pb.Canvas_Spec{
				Nodes:            actions.NodesToProto(nodes),
				Edges:            actions.EdgesToProto(edges),
				ChangeManagement: serializeChangeManagement(changeManagementEnabled, changeRequestApprovers),
			},
		},
	}, nil
}
