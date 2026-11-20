package factories

import (
	"context"
	"maps"
	"slices"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// UpgradeGitHubIntakeAuthorGates upgrades enabled gates without resetting user settings or replaying events.
func UpgradeGitHubIntakeAuthorGates(ctx context.Context, deps IntakeDependencies) error {
	db := database.DB(ctx)
	after := uuid.Nil
	scanned, upgraded, failed := 0, 0, 0
	defer func() {
		log.WithFields(log.Fields{
			"scanned": scanned, "upgraded": upgraded, "skipped": scanned - upgraded - failed, "failed": failed,
		}).Info("GitHub intake author gate upgrade scan finished")
	}()
	for {
		intakes, err := models.ListFactoryIntakesBySourceAfterID(db, models.FactoryIntakeSourceGitHubIssues, after, 100)
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
			after = intake.ID
			scanned++
			changed := false
			logger := log.WithFields(log.Fields{
				"intake_id": intake.ID, "organization_id": intake.OrganizationID, "canvas_id": intake.CanvasID,
			})
			err := db.Transaction(func(tx *gorm.DB) error {
				canvas, err := models.LockCanvasForUpdate(tx, intake.OrganizationID, intake.CanvasID)
				if err != nil {
					return err
				}
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
				if err != nil {
					return err
				}
				spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
				graph := resolveIntakeGraph(intake.Source, spec)
				if graph.AuthorPermissionNodeID == "" && graph.AuthorFilterNodeID == "" {
					return nil
				}
				permission := findIntakeNode(spec.Nodes, graph.AuthorPermissionNodeID)
				gate := findIntakeNode(spec.Nodes, graph.AuthorFilterNodeID)
				if permission == nil || gate == nil {
					logger.Warn("skipping incomplete GitHub intake author gate")
					return nil
				}
				if !intakeSettingsFromGraph(intake.Source, graph, spec).AuthorsWithAccess {
					return nil
				}
				if permission.Configuration["fallbackUsername"] == intakeAuthorFallbackUsername && gate.Configuration["expression"] == intakeAuthorPermissionCondition {
					return nil
				}
				nodes := slices.Clone(version.Nodes)
				for i := range nodes {
					switch nodes[i].ID {
					case permission.ID:
						nodes[i].Configuration = maps.Clone(nodes[i].Configuration)
						nodes[i].Configuration["fallbackUsername"] = intakeAuthorFallbackUsername
					case gate.ID:
						nodes[i].Configuration = maps.Clone(nodes[i].Configuration)
						nodes[i].Configuration["expression"] = intakeAuthorPermissionCondition
					}
				}
				changed = true
				return canvases.PublishGeneratedCanvasNodesWithOwner(ctx, tx, canvas, nil,
					"Upgrade GitHub intake author access", nodes, version.Edges, intakePublisherOptions(deps, canvas.OrganizationID))
			})
			if err != nil {
				failed++
				logger.WithError(err).Warn("failed to upgrade GitHub intake author gate")
				continue
			}
			if changed {
				upgraded++
			}
		}
	}
}
