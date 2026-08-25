package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DeleteFactoryIntake(ctx context.Context, organizationID string, req *pb.DeleteFactoryIntakeRequest) (*pb.DeleteFactoryIntakeResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory intake")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory intake")
	}

	intakeID, err := parseIntakeID(req.GetIntakeId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory intake")
	}

	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := models.FindFactory(tx, orgID, factoryID)
		if err != nil {
			return err
		}

		intake, err := factory.FindIntake(tx, intakeID)
		if err != nil {
			return err
		}

		// The canvas exists only to run this intake, so it goes with it. The
		// cleanup worker hard-deletes it once its runs are gone.
		canvas, err := models.FindCanvasInTransaction(tx, orgID, intake.CanvasID)
		if err != nil {
			return err
		}

		if err := intake.Delete(tx); err != nil {
			return err
		}

		return canvas.SoftDeleteInTransaction(tx)
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory intake")
	}

	return &pb.DeleteFactoryIntakeResponse{}, nil
}

func parseIntakeID(intakeID string) (uuid.UUID, error) {
	id, err := uuid.Parse(intakeID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid intake id")
	}

	return id, nil
}
