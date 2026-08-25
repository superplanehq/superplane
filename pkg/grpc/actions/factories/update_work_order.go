package factories

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"

	log "github.com/sirupsen/logrus"
)

const (
	workOrderTitleMaxLength       = 256
	workOrderDescriptionMaxLength = 5000
)

func UpdateWorkOrder(
	ctx context.Context,
	organizationID string,
	req *pb.UpdateWorkOrderRequest,
) (*pb.UpdateWorkOrderResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	orderID, err := parseOrderID(req.GetOrderId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	title, description, err := workOrderContentFromRequest(req)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	db := database.DB(ctx)
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		order, err := factory.FindWorkOrder(tx, orderID)
		if err != nil {
			return err
		}

		return order.UpdateContent(tx, title, description)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	if err := messages.PublishFactoryWorkOrderUpdated(
		factoryID.String(),
		orderID.String(),
		factoryevents.EventTypeOrderUpdated,
	); err != nil {
		log.WithError(err).Warnf("Failed to publish factory work order updated for order %s", orderID)
	}

	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	order, err := factory.FindWorkOrder(db, orderID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update work order")
	}

	return &pb.UpdateWorkOrderResponse{Order: serialized}, nil
}

func workOrderContentFromRequest(req *pb.UpdateWorkOrderRequest) (*string, *string, error) {
	if req.Title == nil && req.Description == nil {
		return nil, nil, invalidArgument("title or description is required")
	}

	var title *string
	if req.Title != nil {
		trimmed := strings.TrimSpace(*req.Title)
		if trimmed == "" {
			return nil, nil, invalidArgument("title is required")
		}
		if utf8.RuneCountInString(trimmed) > workOrderTitleMaxLength {
			return nil, nil, invalidArgument("title is too long")
		}
		title = &trimmed
	}

	var description *string
	if req.Description != nil {
		if utf8.RuneCountInString(*req.Description) > workOrderDescriptionMaxLength {
			return nil, nil, invalidArgument("description is too long")
		}
		value := *req.Description
		description = &value
	}

	return title, description, nil
}
