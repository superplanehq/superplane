package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const (
	backlogDefaultName        = "Backlog"
	backlogDefaultDescription = "Plan new draft tasks."
	backlogTemplateVersion    = 3
	backlogTriggerNodeID      = "trigger"
	backlogTriggerName        = "On Task"

	backlogRefinementFilterNodeID = "task-refinement-enabled"
	backlogRefinementNodeID       = "refine-task"
)

// ensureBacklogCanvas creates the factory Backlog automation when the factory
// has no On Work Order trigger yet. A second intake must not add a second
// planner.
func ensureBacklogCanvas(
	ctx context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	factoryModel *models.Factory,
) error {
	has, err := factoryHasOnWorkOrderCanvas(tx, factoryModel)
	if err != nil {
		return err
	}
	if has {
		return nil
	}

	_, err = createBacklogCanvas(ctx, deps, factoryModel)
	return err
}

func factoryHasOnWorkOrderCanvas(tx *gorm.DB, factoryModel *models.Factory) (bool, error) {
	canvases, err := factoryModel.ListCanvases(tx)
	if err != nil {
		return false, err
	}

	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID == nil {
			continue
		}
		ids = append(ids, canvases[i].ID)
	}
	if len(ids) == 0 {
		return false, nil
	}

	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(tx, ids)
	if err != nil {
		return false, err
	}

	for _, spec := range specs {
		if onWorkOrderNodeIDFromSpec(spec) != "" {
			return true, nil
		}
	}

	return false, nil
}

func onWorkOrderNodeIDFromSpec(spec models.LiveCanvasSpec) string {
	for i := range spec.Nodes {
		if spec.Nodes[i].Type != models.NodeTypeTrigger {
			continue
		}
		if spec.Nodes[i].ComponentName() == factory.OnWorkOrderTriggerName {
			return spec.Nodes[i].ID
		}
	}
	return ""
}

func createBacklogCanvas(
	ctx context.Context,
	deps IntakeDependencies,
	factoryModel *models.Factory,
) (uuid.UUID, error) {
	db := database.DB(ctx)
	name, err := models.AvailableCanvasName(db, factoryModel.OrganizationID, &factoryModel.ID, backlogDefaultName)
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create Backlog automation")
	}

	canvasDoc := buildBacklogCanvas(backlogCanvasRequest{
		Name:       name,
		Agent:      resolveIntakeAgent(db, factoryModel),
		GitHubName: resolveGitHubInstallationName(db, factoryModel),
	})

	nodes, edges, err := canvasDoc.Parse(deps.Registry, factoryModel.OrganizationID.String())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to build Backlog automation")
	}

	response, err := canvases.CreateCanvas(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.WebhookBaseURL,
		factoryModel.OrganizationID,
		canvasDoc.Metadata.Name,
		canvasDoc.Metadata.Description,
		&factoryModel.ID,
		nodes,
		edges,
	)
	if err != nil {
		return uuid.Nil, err
	}

	canvasID, err := uuid.Parse(response.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create Backlog automation")
	}
	if err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		canvasModel, err := models.FindCanvasInTransaction(tx, factoryModel.OrganizationID, canvasID)
		if err != nil {
			return err
		}
		return canvasModel.StampFactoryAppTemplate(
			tx,
			backlogTriggerNodeID,
			models.FactoryAppTemplateBacklogID,
			backlogTemplateVersion,
		)
	}); err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to mark Backlog automation version")
	}

	return canvasID, nil
}

type backlogCanvasRequest struct {
	Name       string
	Agent      *intakeAgent
	GitHubName string
}

func buildBacklogCanvas(request backlogCanvasRequest) *yaml.Canvas {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = backlogDefaultName
	}

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: backlogDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: backlogTriggerNodeID, TargetID: backlogRefinementFilterNodeID},
				{Channel: "true", SourceID: backlogRefinementFilterNodeID, TargetID: backlogRefinementNodeID},
				{Channel: "failed", SourceID: backlogRefinementNodeID, TargetID: intakeAddRunErrorNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:        backlogTriggerNodeID,
					Name:      backlogTriggerName,
					Type:      yaml.NodeTypeTrigger,
					Component: factory.OnWorkOrderTriggerName,
					Metadata:  models.FactoryAppTemplateMetadata(models.FactoryAppTemplateBacklogID, backlogTemplateVersion),
					Position:  yaml.Position{X: 160, Y: 80},
				},
				{
					ID:        backlogRefinementFilterNodeID,
					Name:      "Refine task?",
					Type:      yaml.NodeTypeAction,
					Component: intakeFilterComponent,
					Configuration: map[string]any{
						"expression": "{{ root().data.taskRefinementEnabled == true }}",
					},
					Concurrency: intakeConcurrency(),
					Position:    yaml.Position{X: 160, Y: 260},
				},
				{
					ID:            backlogRefinementNodeID,
					Name:          "Refine Task",
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: intakeRefinementConfiguration(request.Agent, request.GitHubName),
					Concurrency:   intakeConcurrency(),
					Position:      yaml.Position{X: 160, Y: 440},
				},
				{
					ID:        intakeAddRunErrorNodeID,
					Name:      intakeAddRunErrorNodeName,
					Type:      yaml.NodeTypeAction,
					Component: intakeAddRunErrorComponent,
					Configuration: map[string]any{
						"message": intakeAddRunErrorMessage,
					},
					Position: yaml.Position{X: 160, Y: 620},
				},
			},
		},
	}
}
