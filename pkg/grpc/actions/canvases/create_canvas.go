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
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ErrDuplicateCanvasName = "duplicate key value violates unique constraint"

func CreateCanvas(
	ctx context.Context,
	registry *registry.Registry,
	encryptor crypto.Encryptor,
	authService authorization.Authorization,
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

	if pbCanvas.Metadata.GetIsTemplate() {
		return nil, status.Error(codes.InvalidArgument, "templates cannot be created")
	}

	return CreateCanvasWithAutoLayoutAndUsage(
		ctx,
		usageService,
		registry,
		organizationID.String(),
		pbCanvas,
		autoLayout,
	)
}

func CreateCanvasWithAutoLayoutAndUsage(
	ctx context.Context,
	usageService usage.Service,
	registry *registry.Registry,
	organizationID string,
	pbCanvas *pb.Canvas,
	autoLayout *pb.CanvasAutoLayout,
) (*pb.CreateCanvasResponse, error) {
	if pbCanvas == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas is required")
	}

	if pbCanvas.GetMetadata() == nil {
		return nil, status.Error(codes.InvalidArgument, "canvas metadata is required")
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	createdBy := uuid.MustParse(userID)
	var err error
	if pbCanvas.Metadata.GetIsTemplate() {
		var canvas *models.Canvas
		err = database.Conn().Transaction(func(tx *gorm.DB) error {
			var txErr error
			canvas, txErr = CreatePublishedTemplateCanvasWithoutSetupInTransaction(
				tx,
				registry,
				pbCanvas,
				autoLayout,
				&createdBy,
				organizationID,
			)
			if txErr != nil {
				if strings.Contains(txErr.Error(), ErrDuplicateCanvasName) {
					return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
				}
				if errors.Is(txErr, errTemplateCanvasAutoLayout) {
					return status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", txErr)
				}
				return txErr
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return createCanvasResponse(canvas, organizationID)
	}

	nodes, edges, err := ParseCanvas(registry, organizationID, pbCanvas)
	if err != nil {
		return nil, err
	}

	nodes, edges, err = layout.ApplyLayout(nodes, edges, autoLayout)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to apply layout: %v", err)
	}

	expandedNodes, err := expandNodes(organizationID, nodes)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	targetOrganizationID := uuid.MustParse(organizationID)
	changeManagementEnabled, err := models.IsChangeManagementEnabled(targetOrganizationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load organization change management setting: %v", err)
	}

	canvasCount, err := models.CountCanvasesByOrganization(organizationID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count organization canvases: %v", err)
	}

	if err := usage.EnsureOrganizationWithinLimits(ctx, usageService, organizationID, &usagepb.OrganizationState{
		Canvases: int32(canvasCount + 1),
	}, &usagepb.CanvasState{
		Nodes: int32(len(expandedNodes)),
	}); err != nil {
		return nil, err
	}

	canvas := &models.Canvas{
		ID:                      uuid.New(),
		OrganizationID:          targetOrganizationID,
		LiveVersionID:           ptrUUID(uuid.New()),
		IsTemplate:              false,
		ChangeManagementEnabled: changeManagementEnabled,
		Name:                    pbCanvas.Metadata.Name,
		Description:             pbCanvas.Metadata.Description,
		CreatedBy:               &createdBy,
		CreatedAt:               &now,
		UpdatedAt:               &now,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Returning{}).Create(canvas).Error; err != nil {
			if strings.Contains(err.Error(), ErrDuplicateCanvasName) {
				return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
			}
			return err
		}

		// This helper persists validated nodes as data only and intentionally skips runtime setup.
		if err := persistCanvasNodesWithoutSetupInTransaction(tx, canvas.ID, expandedNodes, &now); err != nil {
			return err
		}

		version, err := models.CreatePublishedCanvasVersionInTransaction(
			tx,
			canvas.ID,
			&createdBy,
			expandedNodes,
			edges,
		)
		if err != nil {
			return err
		}
		canvas.LiveVersionID = &version.ID
		canvas.UpdatedAt = version.UpdatedAt

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createCanvasResponse(canvas, organizationID)
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func createCanvasResponse(canvas *models.Canvas, creatorOrganizationID string) (*pb.CreateCanvasResponse, error) {
	if publishErr := messages.NewCanvasCreatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishCreated(); publishErr != nil {
		log.Errorf("failed to publish canvas created RabbitMQ message: %v", publishErr)
	}

	userOrganizationID := canvas.OrganizationID.String()
	if canvas.IsTemplate {
		userOrganizationID = creatorOrganizationID
	}

	var user *models.User
	if canvas.CreatedBy != nil {
		var err error
		user, err = models.FindMaybeDeletedUserByID(userOrganizationID, canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	proto, err := SerializeCanvas(canvas, false, user)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
