package factories

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// RetireProductiveIntakes publishes empty graphs in bounded transactions. Run only
// after all API and worker instances have the Productive setup and execution blocks.
// No remote API calls run here; webhook cleanup starts after each commit.
func RetireProductiveIntakes(ctx context.Context, deps IntakeDependencies) error {
	db := database.DB(ctx)
	after := uuid.Nil
	for {
		intakes, err := models.ListProductiveIntakesForRetirement(db, after, 100)
		if err != nil {
			return err
		}
		if len(intakes) == 0 {
			return nil
		}
		for _, intake := range intakes {
			if err := ctx.Err(); err != nil {
				return err
			}
			var cancellations []*models.RunCancellationResult
			err := db.Transaction(func(tx *gorm.DB) error {
				canvas, err := models.LockUnscopedCanvas(tx, intake.CanvasID)
				if err != nil {
					return err
				}
				nodes, err := models.FindCanvasNodesUnscopedInTransaction(tx, canvas.ID)
				if err != nil {
					return err
				}
				if err := intake.RecordRetirement(tx, nodes); err != nil {
					return err
				}
				// Ownership validation no longer applies to the disabled graph.
				// The retirement identity continues to block publication and execution.
				if err := intake.Delete(tx); err != nil {
					return err
				}
				// Old DeleteFactoryIntake requests soft-deleted the canvas. Retain its
				// remaining history and prevent CanvasCleanupWorker from removing runs.
				if canvas.DeletedAt.Valid {
					if err := tx.Unscoped().Model(canvas).Update("deleted_at", nil).Error; err != nil {
						return err
					}
					canvas.DeletedAt = gorm.DeletedAt{}
				}

				cancellations, err = cancelProductiveIntakeRuns(tx, canvas.ID)
				if err != nil {
					return err
				}
				if err := canvases.PublishGeneratedCanvasNodesWithOwner(ctx, tx, canvas, nil,
					"Retire Productive Factory intake", []models.Node{}, []models.Edge{}, intakePublisherOptions(deps, canvas.OrganizationID)); err != nil {
					return err
				}
				// Also drain nodes absent from the current version, including previously
				// deleted nodes with pending work left by an interrupted retirement.
				for _, node := range nodes {
					if _, err := models.DeleteCanvasNodeWithResult(tx, node); err != nil {
						return err
					}
					if err := tx.Unscoped().Model(&node).Update("webhook_id", nil).Error; err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("retire Productive intake %s (canvas %s): %w", intake.ID, intake.CanvasID, err)
			}
			for _, cancellation := range cancellations {
				messages.PublishRunCancellationDrain(intake.CanvasID, cancellation.Drain)
			}
			log.WithFields(log.Fields{"intake_id": intake.ID, "canvas_id": intake.CanvasID, "organization_id": intake.OrganizationID}).Info("Retired Productive Factory intake")
			after = intake.ID
		}
	}
}

func cancelProductiveIntakeRuns(tx *gorm.DB, canvasID uuid.UUID) ([]*models.RunCancellationResult, error) {
	var cancellations []*models.RunCancellationResult
	for {
		runs, err := models.ListCanvasRuns(tx, canvasID, 100, nil, models.CanvasRunFilters{States: []string{models.CanvasRunStatePending, models.CanvasRunStateStarted}})
		if err != nil {
			return nil, err
		}
		if len(runs) == 0 {
			return cancellations, nil
		}
		for i := range runs {
			cancellation, err := runs[i].RequestCancellation(tx, nil)
			if err != nil {
				return nil, err
			}
			cancellations = append(cancellations, cancellation)
		}
	}
}
