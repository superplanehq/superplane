package factories

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	workersctx "github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func DuplicateWorkOrder(ctx context.Context, organizationID string, req *pb.DuplicateWorkOrderRequest) (*pb.DuplicateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	createdByID, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to duplicate work order")
	}

	enabled, err := models.HasExperimentalFeature(orgID, features.FeatureFactoryCreateWithAgent)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}
	if !enabled {
		return nil, grpcerrors.PermissionDenied(nil, "task refinement is not enabled")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	source, err := findWorkOrder(db, factory, req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	assigneeIDs := []uuid.UUID{createdByID}
	var order *models.FactoryWorkOrder
	var bound storedfiles.BindResult
	err = db.Transaction(func(tx *gorm.DB) error {
		created, createErr := createDuplicateDraft(tx, factory, source, createdByID, assigneeIDs)
		if createErr != nil {
			return createErr
		}
		order = created

		nextDescription, result, cloneErr := storedfiles.CloneDescriptionFiles(
			ctx,
			tx,
			blob.Current(),
			orgID,
			factory.ID,
			source.ID,
			order.ID,
			createdByID,
			order.Description,
		)
		bound = result
		if cloneErr != nil {
			return cloneErr
		}
		if nextDescription != order.Description {
			if err := order.UpdateContent(tx, nil, &nextDescription); err != nil {
				return err
			}
		}
		if err := copyRepositorySnapshot(tx, order, source); err != nil {
			return err
		}
		return copyPlanningSpec(tx, source, order)
	})
	if delErr := storedfiles.ApplyBindResult(ctx, db, blob.Current(), orgID, factory.ID, bound, err); delErr != nil {
		log.WithError(delErr).Warn("Failed to delete file objects after clone")
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	workersctx.EmitWorkOrderCreated(db, factory, order)

	if err := messages.PublishFactoryWorkOrderUpdated(
		factory.ID.String(),
		order.ID.String(),
		factoryevents.EventTypeOrderStatusUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", order.ID)
	}

	if assignedIDs := newAssigneeIDs(nil, assigneeIDs); len(assignedIDs) > 0 {
		notification := messages.FactoryWorkOrderNotificationMessage{
			OrganizationID:  orgID.String(),
			FactoryID:       factory.ID.String(),
			OrderID:         order.ID.String(),
			EventType:       factoryevents.EventTypeOrderAssigneesUpdated,
			ActorUserID:     createdByID.String(),
			AssignedUserIDs: assignedIDs,
		}
		if err := notification.Publish(); err != nil {
			log.WithError(err).Warnf("Failed to publish work order notification for order %s", order.ID)
		}
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to duplicate work order")
	}

	return &pb.DuplicateWorkOrderResponse{Order: serialized}, nil
}

func createDuplicateDraft(
	tx *gorm.DB,
	factory *models.Factory,
	source *models.FactoryWorkOrder,
	createdByID uuid.UUID,
	assigneeIDs []uuid.UUID,
) (*models.FactoryWorkOrder, error) {
	if origin := source.Origin(); origin != nil {
		return factory.CreateWorkOrderWithOrigin(
			tx,
			source.Title,
			source.Description,
			&createdByID,
			assigneeIDs,
			nil,
			*origin,
		)
	}
	return factory.CreateWorkOrder(tx, source.Title, source.Description, &createdByID, assigneeIDs, nil)
}

func copyRepositorySnapshot(tx *gorm.DB, dest, source *models.FactoryWorkOrder) error {
	now := time.Now()
	dest.Repository = source.Repository
	dest.DefaultBranch = source.DefaultBranch
	dest.UpdatedAt = now
	return tx.Model(dest).Omit(clause.Associations).Updates(map[string]any{
		"repository":     source.Repository,
		"default_branch": source.DefaultBranch,
		"updated_at":     now,
	}).Error
}

func copyPlanningSpec(tx *gorm.DB, source, dest *models.FactoryWorkOrder) error {
	artifact, err := source.FindArtifactByKey(tx, models.PlanningSpecArtifactKey+":"+source.ID.String())
	if errors.Is(err, models.ErrFactoryWorkOrderArtifactNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	data := map[string]any{}
	if len(artifact.Data) > 0 {
		if err := json.Unmarshal(artifact.Data, &data); err != nil {
			return err
		}
	}
	body, _ := data["body"].(string)
	if strings.TrimSpace(body) == "" {
		return nil
	}

	_, err = dest.CreateArtifact(tx, models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeMarkdown,
		Key:  models.PlanningSpecArtifactKey + ":" + dest.ID.String(),
		Data: map[string]any{
			"name":  models.PlanningSpecArtifactTitle,
			"title": models.PlanningSpecArtifactTitle,
			"body":  body,
		},
	})
	return err
}
