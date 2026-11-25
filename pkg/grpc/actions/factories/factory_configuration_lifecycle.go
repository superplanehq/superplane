package factories

import (
	"errors"
	"slices"

	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func configurationStagePlacement(stage *factoryConfigurationStageTarget) string {
	if stage.stage.Kind == models.FactoryLineStagePhase && stage.stage.StepIndex != nil && *stage.stage.StepIndex == 0 {
		return models.CanvasColumnKeyImplement
	}
	return stage.stage.Key
}

func validateFactoryConfigurationAutomationRemoval(tx *gorm.DB, target *factoryConfigurationTarget, current *factoryConfigurationAutomationTarget) error {
	canvas := current.canvas
	if canvas.OrganizationID != target.factory.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != target.factory.ID {
		return gorm.ErrRecordNotFound
	}
	reserved := rejectReservedFactoryAutomation(tx, target.factory, canvas)
	if !errors.Is(reserved, errFactoryAutomationReserved) {
		return reserved
	}
	if models.IsBacklogFactoryApp(current.version.Nodes, current.version.Edges) {
		return grpcerrors.FailedPrecondition(nil, "Keep the required Task analysis automation.")
	}
	lines, err := target.factory.ListLines(tx)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if slices.ContainsFunc(line.Steps, func(step models.FactoryLineStep) bool { return step.AppID == canvas.ID }) {
			return grpcerrors.FailedPrecondition(nil, "Keep every phase runner automation.")
		}
	}
	intake, err := models.FindFactoryIntakeByCanvasID(tx, canvas.ID)
	if err != nil && !errors.Is(err, models.ErrFactoryIntakeNotFound) {
		return err
	}
	if intake != nil && (intake.OrganizationID != target.factory.OrganizationID || intake.FactoryID != target.factory.ID) {
		return gorm.ErrRecordNotFound
	}
	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, canvas.ID)
	if err != nil {
		return err
	}
	if handler != nil && (handler.OrganizationID != target.factory.OrganizationID || handler.FactoryID != target.factory.ID) {
		return gorm.ErrRecordNotFound
	}
	if intake != nil || handler != nil {
		return nil
	}
	return reserved
}

func removeFactoryConfigurationAutomation(tx *gorm.DB, target *factoryConfigurationTarget, current *factoryConfigurationAutomationTarget) (func(), error) {
	if err := validateFactoryConfigurationAutomationRemoval(tx, target, current); err != nil {
		return nil, err
	}
	canvas := current.canvas
	if err := models.DeleteFactoryIntakesByCanvas(tx, canvas.ID); err != nil {
		return nil, err
	}
	if err := models.DeleteFactoryPRFeedbackHandlersByCanvas(tx, canvas.ID); err != nil {
		return nil, err
	}
	authored, err := findLegacyAutomationSource(tx, canvas)
	if err != nil {
		return nil, err
	}
	if authored != nil {
		if authored.FactoryID != target.factory.ID {
			return nil, gorm.ErrRecordNotFound
		}
		if err := authored.Delete(tx); err != nil {
			return nil, err
		}
	}
	if err := canvas.SoftDeleteInTransaction(tx); err != nil {
		return nil, err
	}
	return func() { canvases.PublishCanvasDeleted(canvas) }, nil
}
