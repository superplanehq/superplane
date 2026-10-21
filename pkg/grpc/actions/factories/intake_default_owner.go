package factories

import (
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func validateIntakeDefaultOwner(tx *gorm.DB, organizationID uuid.UUID, source, currentOwner string, settings *pb.FactoryIntake_Settings) error {
	if settings == nil || settings.DefaultOwnerId == nil {
		return nil
	}
	if source != models.FactoryIntakeSourceGitHubIssues {
		return invalidArgument("default task owner is only supported for GitHub issues")
	}
	ownerID := settings.GetDefaultOwnerId()
	if ownerID == "" || ownerID == currentOwner {
		return nil
	}
	id, err := uuid.Parse(ownerID)
	if err != nil {
		return invalidArgument("invalid default task owner id")
	}
	user, err := models.FindActiveUserByIDInTransaction(tx, organizationID.String(), id.String())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return invalidArgument("default task owner must be an active organization member")
	}
	if err != nil {
		return err
	}
	if !user.IsHuman() {
		return invalidArgument("default task owner must be a person")
	}
	return nil
}
