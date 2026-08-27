package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DeleteFactoryPRFeedbackHandler(
	ctx context.Context,
	organizationID string,
	req *pb.DeleteFactoryPRFeedbackHandlerRequest,
) (*pb.DeleteFactoryPRFeedbackHandlerResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory PR feedback handler")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory PR feedback handler")
	}

	handlerID, err := parsePRFeedbackHandlerID(req.GetHandlerId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory PR feedback handler")
	}

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		handler, err := factory.FindPRFeedbackHandler(tx, handlerID)
		if err != nil {
			return err
		}

		canvas, err := models.FindCanvasInTransaction(tx, orgID, handler.CanvasID)
		if err != nil {
			return err
		}

		if err := handler.Delete(tx); err != nil {
			return err
		}

		return canvas.SoftDeleteInTransaction(tx)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory PR feedback handler")
	}

	return &pb.DeleteFactoryPRFeedbackHandlerResponse{}, nil
}

func parsePRFeedbackHandlerID(handlerID string) (uuid.UUID, error) {
	id, err := uuid.Parse(handlerID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid handler id")
	}

	return id, nil
}
