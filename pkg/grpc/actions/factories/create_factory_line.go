package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func CreateFactoryLine(ctx context.Context, organizationID string, req *pb.CreateFactoryLineRequest) (*pb.CreateFactoryLineResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, factoryErrorToStatus(invalidArgument("name is required"), "failed to create factory line")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	var line *models.FactoryLine
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := factory.LockForUpdate(tx); err != nil {
			return err
		}
		steps, err := parseLineSteps(tx, orgID, factory.ID, req.GetSteps())
		if err != nil {
			return err
		}
		line, err = factory.CreateLine(tx, name, steps)
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory line")
	}

	return &pb.CreateFactoryLineResponse{
		Line: serializeFactoryLine(line),
	}, nil
}
