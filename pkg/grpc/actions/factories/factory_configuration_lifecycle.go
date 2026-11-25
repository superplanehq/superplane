package factories

import (
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

func validateFactoryConfigurationAutomationRemoval(target *factoryConfigurationTarget, current *factoryConfigurationAutomationTarget) error {
	canvas := current.canvas
	if canvas.OrganizationID != target.factory.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != target.factory.ID {
		return gorm.ErrRecordNotFound
	}
	if models.IsBacklogFactoryApp(current.version.Nodes, current.version.Edges) {
		return grpcerrors.FailedPrecondition(nil, "Keep the required Task analysis automation.")
	}
	for _, line := range target.reads.lines {
		if slices.ContainsFunc(line.Steps, func(step models.FactoryLineStep) bool { return step.AppID == canvas.ID }) {
			return grpcerrors.FailedPrecondition(nil, "Keep every phase runner automation.")
		}
	}
	intake := current.metadata.intake
	if intake != nil && (intake.OrganizationID != target.factory.OrganizationID || intake.FactoryID != target.factory.ID) {
		return gorm.ErrRecordNotFound
	}
	handler := current.metadata.handler
	if handler != nil && (handler.OrganizationID != target.factory.OrganizationID || handler.FactoryID != target.factory.ID) {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func removeFactoryConfigurationAutomation(tx *gorm.DB, target *factoryConfigurationTarget, current *factoryConfigurationAutomationTarget) (func(), error) {
	if err := validateFactoryConfigurationAutomationRemoval(target, current); err != nil {
		return nil, err
	}
	canvas := current.canvas
	if err := models.DeleteFactoryIntakesByCanvas(tx, canvas.ID); err != nil {
		return nil, err
	}
	if err := models.DeleteFactoryPRFeedbackHandlersByCanvas(tx, canvas.ID); err != nil {
		return nil, err
	}
	authored := current.metadata.source
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
