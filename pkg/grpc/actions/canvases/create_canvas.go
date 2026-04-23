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
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/layout"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	usagepb "github.com/superplanehq/superplane/pkg/protos/usage"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
	if pbCanvas.Spec != nil && pbCanvas.Spec.ChangeManagement != nil {
		changeManagementEnabled = pbCanvas.Spec.ChangeManagement.Enabled

		approvers, parseErr := parseCanvasChangeRequestApprovalConfig(pbCanvas.Spec.ChangeManagement)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", parseErr)
		}

		if approvers != nil {
			if validateErr := validateCanvasChangeRequestApprovers(
				authService,
				organizationID.String(),
				approvers,
			); validateErr != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", validateErr)
			}
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
	versionID := uuid.New()

	now := time.Now()
	canvas := models.Canvas{
		ID:             canvasID,
		OrganizationID: organizationID,
		LiveVersionID:  &versionID,
		IsTemplate:     false,
		CreatedBy:      &createdBy,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		findErr := ensureCanvasNameAvailableInTransaction(tx, organizationID, canvasID, name)
		if errors.Is(findErr, models.ErrCanvasNameAlreadyExists) {
			return status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
		}
		if findErr != nil {
			return findErr
		}

		//
		// Create the workflow record
		//
		err := tx.Clauses(clause.Returning{}).Create(&canvas).Error
		if err != nil {
			return err
		}

		//
		// Create new empty canvas version record
		//
		emptyVersion := models.CanvasVersion{
			ID:                      versionID,
			WorkflowID:              canvasID,
			OwnerID:                 &createdBy,
			State:                   models.CanvasVersionStatePublished,
			Name:                    name,
			Description:             pbCanvas.Metadata.Description,
			ChangeManagementEnabled: changeManagementEnabled,
			ChangeRequestApprovers:  datatypes.NewJSONSlice(changeRequestApprovers),
			PublishedAt:             &now,
			Nodes:                   datatypes.NewJSONSlice([]models.Node{}),
			Edges:                   datatypes.NewJSONSlice([]models.Edge{}),
			CreatedAt:               &now,
			UpdatedAt:               &now,
		}

		if err := tx.Create(&emptyVersion).Error; err != nil {
			return err
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

	canvas.ChangeManagementEnabled = changeManagementEnabled

	var user *models.User
	if canvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(canvas.OrganizationID.String(), canvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	proto, err := SerializeCanvas(&canvas, false, user)
	if err != nil {
		return nil, err
	}

	return &pb.CreateCanvasResponse{
		Canvas: proto,
	}, nil
}
