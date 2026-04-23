package canvases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpdateCanvas(
	_ context.Context,
	authService authorization.Authorization,
	organizationID string,
	id string,
	name *string,
	description *string,
	changeManagement *pb.Canvas_ChangeManagement,
) (*pb.UpdateCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	organizationUUID := uuid.MustParse(organizationID)

	canvas, err := findWritableCanvas(organizationUUID, canvasID)
	if err != nil {
		return nil, err
	}

	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		return updateCanvasInTransaction(tx, authService, organizationID, organizationUUID, canvasID, name, description, changeManagement)
	})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			return nil, s.Err()
		}
		return nil, err
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String(), canvas.OrganizationID.String()).PublishUpdated(); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	refreshedCanvas, err := models.FindCanvas(organizationUUID, canvasID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to load updated canvas: %v", err)
	}

	var user *models.User
	if refreshedCanvas.CreatedBy != nil {
		user, err = models.FindMaybeDeletedUserByID(refreshedCanvas.OrganizationID.String(), refreshedCanvas.CreatedBy.String())
		if err != nil {
			return nil, err
		}
	}

	serializedCanvas, err := SerializeCanvas(refreshedCanvas, false, user)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasResponse{Canvas: serializedCanvas}, nil
}

func findWritableCanvas(organizationUUID, canvasID uuid.UUID) (*models.Canvas, error) {
	canvas, err := models.FindCanvas(organizationUUID, canvasID)
	if err == nil {
		if canvas.IsTemplate {
			return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
		}

		return canvas, nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if _, templateErr := models.FindCanvasTemplate(canvasID); templateErr == nil {
			return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
		}
	}

	return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
}

func updateCanvasInTransaction(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	name *string,
	description *string,
	changeManagement *pb.Canvas_ChangeManagement,
) error {
	lockedCanvas, err := lockCanvasForUpdate(tx, organizationUUID, canvasID)
	if err != nil {
		return err
	}

	liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return err
	}

	changed, err := applyCanvasLiveVersionUpdates(
		tx,
		authService,
		organizationID,
		organizationUUID,
		canvasID,
		liveVersion,
		name,
		description,
		changeManagement,
	)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	return saveCanvasMetadataUpdate(tx, lockedCanvas, liveVersion)
}

func lockCanvasForUpdate(tx *gorm.DB, organizationUUID, canvasID uuid.UUID) (*models.Canvas, error) {
	lockedCanvas := &models.Canvas{}
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("organization_id = ?", organizationUUID).
		Where("id = ?", canvasID).
		First(lockedCanvas).
		Error
	if err != nil {
		return nil, err
	}

	return lockedCanvas, nil
}

func applyCanvasLiveVersionUpdates(
	tx *gorm.DB,
	authService authorization.Authorization,
	organizationID string,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	liveVersion *models.CanvasVersion,
	name *string,
	description *string,
	changeManagement *pb.Canvas_ChangeManagement,
) (bool, error) {
	nameChanged, err := applyCanvasNameUpdate(tx, organizationUUID, canvasID, liveVersion, name)
	if err != nil {
		return false, err
	}

	descriptionChanged := applyCanvasDescriptionUpdate(liveVersion, description)

	changeManagementChanged, err := applyCanvasChangeManagementUpdate(authService, organizationID, liveVersion, changeManagement)
	if err != nil {
		return false, err
	}

	return nameChanged || descriptionChanged || changeManagementChanged, nil
}

func applyCanvasNameUpdate(
	tx *gorm.DB,
	organizationUUID uuid.UUID,
	canvasID uuid.UUID,
	liveVersion *models.CanvasVersion,
	name *string,
) (bool, error) {
	if name == nil {
		return false, nil
	}

	nextName := strings.TrimSpace(*name)
	if nextName == "" {
		return false, status.Error(codes.InvalidArgument, "canvas name is required")
	}

	nameErr := ensureCanvasNameAvailableInTransaction(tx, organizationUUID, canvasID, nextName)
	if errors.Is(nameErr, models.ErrCanvasNameAlreadyExists) {
		return false, status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
	}
	if nameErr != nil {
		return false, nameErr
	}

	if liveVersion.Name == nextName {
		return false, nil
	}

	liveVersion.Name = nextName
	return true, nil
}

func applyCanvasDescriptionUpdate(liveVersion *models.CanvasVersion, description *string) bool {
	if description == nil || liveVersion.Description == *description {
		return false
	}

	liveVersion.Description = *description
	return true
}

func applyCanvasChangeManagementUpdate(
	authService authorization.Authorization,
	organizationID string,
	liveVersion *models.CanvasVersion,
	changeManagement *pb.Canvas_ChangeManagement,
) (bool, error) {
	if changeManagement == nil {
		return false, nil
	}

	nextApprovers, err := parseCanvasChangeRequestApprovalConfig(changeManagement)
	if err != nil {
		return false, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", err)
	}

	if nextApprovers != nil {
		if err := validateCanvasChangeRequestApprovers(authService, organizationID, nextApprovers); err != nil {
			return false, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", err)
		}
	}

	approversChanged := applyCanvasApproversUpdate(liveVersion, nextApprovers)
	changeManagementEnabledChanged := applyCanvasChangeManagementEnabledUpdate(liveVersion, changeManagement.Enabled)

	return approversChanged || changeManagementEnabledChanged, nil
}

func applyCanvasApproversUpdate(
	liveVersion *models.CanvasVersion,
	nextApprovers []models.CanvasChangeRequestApprover,
) bool {
	if nextApprovers == nil {
		return false
	}

	currentApprovers := liveVersion.EffectiveChangeRequestApprovers()
	if slices.EqualFunc(currentApprovers, nextApprovers, func(left, right models.CanvasChangeRequestApprover) bool {
		return left.Type == right.Type && left.User == right.User && left.Role == right.Role
	}) {
		return false
	}

	liveVersion.ChangeRequestApprovers = datatypes.NewJSONSlice(nextApprovers)
	return true
}

func applyCanvasChangeManagementEnabledUpdate(liveVersion *models.CanvasVersion, enabled bool) bool {
	if liveVersion.ChangeManagementEnabled == enabled {
		return false
	}

	liveVersion.ChangeManagementEnabled = enabled
	return true
}

func saveCanvasMetadataUpdate(tx *gorm.DB, lockedCanvas *models.Canvas, liveVersion *models.CanvasVersion) error {
	now := time.Now()
	liveVersion.UpdatedAt = &now
	lockedCanvas.UpdatedAt = &now

	if err := tx.Save(liveVersion).Error; err != nil {
		return err
	}

	return tx.Save(lockedCanvas).Error
}
