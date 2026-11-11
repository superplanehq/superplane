package factories

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// UpgradeGitHubIntakeAuthorGates upgrades enabled gates without resetting user settings or replaying events.
func UpgradeGitHubIntakeAuthorGates(ctx context.Context, deps IntakeDependencies) error {
	db := database.DB(ctx)
	after := uuid.Nil
	for {
		intakes, err := models.ListFactoryIntakesBySourceAfterID(db, models.FactoryIntakeSourceGitHubIssues, after, 100)
		if err != nil {
			return err
		}
		if len(intakes) == 0 {
			return nil
		}
		for _, intake := range intakes {
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
				if !intakeSettingsFromGraph(intake.Source, graph, spec).AuthorsWithAccess {
					return nil
				}
				permission := findIntakeNode(spec.Nodes, graph.AuthorPermissionNodeID)
				gate := findIntakeNode(spec.Nodes, graph.AuthorFilterNodeID)
				if permission == nil || gate == nil {
					return fmt.Errorf("enabled author gate is incomplete")
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
				return canvases.PublishGeneratedCanvasNodesWithOwner(ctx, tx, canvas, nil,
					"Upgrade GitHub intake author access", nodes, version.Edges, intakePublisherOptions(deps, canvas.OrganizationID))
			})
			if err != nil {
				return fmt.Errorf("upgrade GitHub intake %s: %w", intake.ID, err)
			}
			after = intake.ID
		}
	}
}
