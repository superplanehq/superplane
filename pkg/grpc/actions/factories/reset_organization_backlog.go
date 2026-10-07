package factories

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const resetBacklogDefaultsCommitMessage = "Reset Backlog defaults"

type ResetOrganizationBacklogResult struct {
	Reset    int                               `json:"reset"`
	Failures []ResetOrganizationBacklogFailure `json:"failures"`
}

type ResetOrganizationBacklogFailure struct {
	CanvasID string `json:"canvas_id"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

// ResetOrganizationBacklogDefaults publishes the current Backlog template onto
// every Backlog automation in the organization. Custom prompts and graph
// changes on those automations are replaced. Other automations stay the same.
func ResetOrganizationBacklogDefaults(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
) (*ResetOrganizationBacklogResult, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	if _, err := models.FindOrganizationByIDInTransaction(db, orgID.String()); err != nil {
		return nil, err
	}

	factories, err := models.ListFactories(db, orgID)
	if err != nil {
		return nil, err
	}

	result := &ResetOrganizationBacklogResult{}
	for i := range factories {
		if err := resetFactoryBacklogCanvases(ctx, db, deps, &factories[i], result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func resetFactoryBacklogCanvases(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	result *ResetOrganizationBacklogResult,
) error {
	factoryCanvases, err := factory.ListCanvases(db)
	if err != nil {
		return err
	}

	ids := make([]uuid.UUID, 0, len(factoryCanvases))
	byID := make(map[uuid.UUID]*models.Canvas, len(factoryCanvases))
	for i := range factoryCanvases {
		if factoryCanvases[i].LiveVersionID == nil {
			continue
		}
		ids = append(ids, factoryCanvases[i].ID)
		byID[factoryCanvases[i].ID] = &factoryCanvases[i]
	}
	if len(ids) == 0 {
		return nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, ids)
	if err != nil {
		return err
	}

	for canvasID, spec := range specs {
		if !models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
			continue
		}
		canvas := byID[canvasID]
		if err := resetBacklogCanvas(ctx, db, deps, factory, canvas); err != nil {
			result.Failures = append(result.Failures, ResetOrganizationBacklogFailure{
				CanvasID: canvas.ID.String(),
				Name:     canvas.Name,
				Error:    err.Error(),
			})
			continue
		}
		result.Reset++
	}
	return nil
}

func resetBacklogCanvas(
	ctx context.Context,
	db *gorm.DB,
	deps IntakeDependencies,
	factory *models.Factory,
	canvas *models.Canvas,
) error {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	if err != nil {
		return err
	}

	defaults, err := materializeBacklogDefaults(db, factory, canvas, version)
	if err != nil {
		return err
	}

	doc, err := yaml.CanvasFromYAML([]byte(defaults.canvasYAML))
	if err != nil {
		return err
	}

	nodes, edges, err := doc.Parse(deps.Registry, canvas.OrganizationID.String())
	if err != nil {
		return err
	}
	nodes = copyLiveNodeMetadata(version.Nodes, nodes)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := canvases.PublishGeneratedCanvasNodesWithOwner(
			ctx,
			tx,
			canvas,
			nil,
			resetBacklogDefaultsCommitMessage,
			nodes,
			edges,
			intakePublisherOptions(deps, canvas.OrganizationID),
		); err != nil {
			return err
		}
		return canvas.StampFactoryAppTemplate(
			tx,
			backlogTriggerNodeID,
			models.FactoryAppTemplateBacklogID,
			backlogTemplateVersion,
		)
	})
}

func copyLiveNodeMetadata(liveNodes, proposedNodes []models.Node) []models.Node {
	result := make([]models.Node, len(proposedNodes))
	copy(result, proposedNodes)
	for i, proposed := range result {
		for _, live := range liveNodes {
			if proposed.ID == live.ID {
				result[i].Metadata = live.Metadata
				break
			}
		}
	}
	return result
}
