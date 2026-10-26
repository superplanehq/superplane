package factories

import (
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

var intakeSourcesWithDefaultOwner = []string{
	models.FactoryIntakeSourceGitHubIssues,
	models.FactoryIntakeSourceSentryExceptions,
}

func intakeSourceSupportsDefaultOwner(source string) bool {
	return slices.Contains(intakeSourcesWithDefaultOwner, source)
}

func validateIntakeDefaultOwner(tx *gorm.DB, organizationID uuid.UUID, source, currentOwner string, settings *pb.FactoryIntake_Settings) error {
	if settings == nil || settings.DefaultOwnerId == nil {
		return nil
	}
	if !intakeSourceSupportsDefaultOwner(source) {
		return invalidArgument("default task owner is not supported for this intake")
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
