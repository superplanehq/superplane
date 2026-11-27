package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func DeleteFactoryAutomation(
	ctx context.Context,
	organizationID string,
	req *pb.DeleteFactoryAutomationRequest,
) (*pb.DeleteFactoryAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	automationID, err := parseFactoryAutomationID(req.GetAutomationId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, automationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
		return nil, factoryErrorToStatus(gorm.ErrRecordNotFound, "failed to delete factory automation")
	}

	var cancellations []*models.RunCancellationResult
	err = db.Transaction(func(tx *gorm.DB) error {
		factory, err = models.LockAutomationFactory(tx, orgID, factory.ID)
		if err != nil {
			return err
		}
		if err := rejectManagedFactoryAutomation(tx, canvas); err != nil {
			return err
		}
		if canvas.LiveVersionID != nil {
			version, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
			if err != nil {
				return err
			}
			if models.IsBacklogFactoryApp(version.Nodes, version.Edges) {
				if err := factory.SetBacklogAutomationRemoved(tx, true); err != nil {
					return err
				}
			}
		}
		cancellations, err = deleteInstalledAutomation(tx, factory, canvas)
		return err
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to delete factory automation")
	}
	publishDeletedAutomation(canvas, cancellations)

	return &pb.DeleteFactoryAutomationResponse{}, nil
}

func rejectReservedFactoryAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas) error {
	if err := rejectManagedFactoryAutomation(tx, canvas); err != nil {
		return err
	}
	liveVersion, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	if models.IsBacklogFactoryApp(liveVersion.Nodes, liveVersion.Edges) {
		return errFactoryAutomationReserved
	}
	lines, err := factory.ListLines(tx)
	if err != nil {
		return err
	}
	for _, line := range lines {
		for _, step := range line.Steps {
			if step.AppID == canvas.ID {
				return errFactoryAutomationReserved
			}
		}
	}
	return nil
}

func rejectManagedFactoryAutomation(tx *gorm.DB, canvas *models.Canvas) error {
	if _, err := models.FindFactoryIntakeByCanvasID(tx, canvas.ID); err == nil {
		return errFactoryAutomationReserved
	} else if !errors.Is(err, models.ErrFactoryIntakeNotFound) {
		return err
	}

	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, canvas.ID)
	if err != nil {
		return err
	}
	if handler != nil {
		return errFactoryAutomationReserved
	}

	return nil
}

// Lifecycle owners delete their records separately, but share canvas cleanup.
func deleteInstalledAutomation(tx *gorm.DB, factory *models.Factory, canvas *models.Canvas) ([]*models.RunCancellationResult, error) {
	lines, err := factory.ListLines(tx)
	if err != nil {
		return nil, err
	}
	if len(lines) > 0 {
		line, err := factory.FindLineForUpdate(tx, lines[0].ID)
		if err != nil {
			return nil, err
		}
		stages := line.StagesValue()
		steps, err := line.RemoveDeletedAutomationBindings(tx, stages, []models.Canvas{*canvas})
		if err != nil {
			return nil, err
		}
		if err := line.UpdateConfiguration(tx, nil, steps, nil, stages); err != nil {
			return nil, err
		}
	}
	locked, err := models.LockCanvasForUpdate(tx, factory.OrganizationID, canvas.ID)
	if err != nil {
		return nil, err
	}
	*canvas = *locked
	if canvas.FactoryID == nil || *canvas.FactoryID != factory.ID {
		return nil, gorm.ErrRecordNotFound
	}
	if factory.OnboardingConfigValue().ProvisionedAppID == canvas.ID.String() {
		empty := ""
		if err := factory.UpdateOnboarding(tx, models.FactoryOnboardingPatch{ProvisionedAppID: &empty, ProvisionedLineID: &empty}); err != nil {
			return nil, err
		}
	}
	cancellations, err := canvas.RequestUnfinishedRunCancellation(tx, nil)
	if err != nil {
		return nil, err
	}
	if err := canvas.CancelQueuedFactoryDispatches(tx); err != nil {
		return nil, err
	}
	return cancellations, canvases.DeleteCanvasInTransaction(tx, canvas)
}

func publishDeletedAutomation(canvas *models.Canvas, cancellations []*models.RunCancellationResult) {
	canvases.PublishCanvasDeleted(canvas)
	for _, cancellation := range cancellations {
		run := cancellation.Run
		if cancellation.NewlyCancelling {
			if err := messages.NewCanvasRunMessage(run.WorkflowID.String(), run.ID.String()).Publish(); err != nil {
				log.Errorf("failed to publish run cancellation: %v", err)
			}
		}
		messages.PublishRunCancellationDrain(run.WorkflowID, cancellation.Drain)
	}
}

func parseFactoryAutomationID(value string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid factory automation ID")
	}
	return id, nil
}
