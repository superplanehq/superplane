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
	backlogDefaultDescription = "Score new work orders for how well an agent can complete them."
	backlogTriggerNodeID      = "trigger"
	backlogTriggerName        = "On Work Order"
	backlogAnalysisSubject    = "work order"
)

// ensureBacklogCanvas creates the factory Backlog automation when the factory
// has no On Work Order trigger yet. A second intake must not add a second
// scorer.
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
	name, err := models.AvailableCanvasName(db, factoryModel.OrganizationID, backlogDefaultName)
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create Backlog automation")
	}

	canvasDoc := buildBacklogCanvas(backlogCanvasRequest{
		Name:  name,
		Agent: resolveIntakeAgent(db, factoryModel),
	})

	nodes, edges, err := canvasDoc.Parse(deps.Registry, factoryModel.OrganizationID.String())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to build Backlog automation")
	}

	response, err := canvases.CreateCanvasWithSeedFiles(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.GitProvider,
		deps.WebhookBaseURL,
		factoryModel.OrganizationID,
		canvasDoc.Metadata.Name,
		canvasDoc.Metadata.Description,
		&factoryModel.ID,
		nodes,
		edges,
		deps.UsageService,
		nil,
	)
	if err != nil {
		return uuid.Nil, err
	}

	canvasID, err := uuid.Parse(response.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return uuid.Nil, factoryErrorToStatus(err, "failed to create Backlog automation")
	}

	return canvasID, nil
}

type backlogCanvasRequest struct {
	Name  string
	Agent *intakeAgent
}

func buildBacklogCanvas(request backlogCanvasRequest) *yaml.Canvas {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = backlogDefaultName
	}

	spec := intakeSpec{analysisSubject: backlogAnalysisSubject}

	return &yaml.Canvas{
		APIVersion: yaml.APIVersion,
		Kind:       yaml.KindCanvas,
		Metadata: &yaml.CanvasMetadata{
			Name:        name,
			Description: backlogDefaultDescription,
		},
		Spec: &yaml.CanvasSpec{
			Edges: []yaml.Edge{
				{Channel: "default", SourceID: backlogTriggerNodeID, TargetID: intakeAnalysisNodeID},
				{Channel: "passed", SourceID: intakeAnalysisNodeID, TargetID: intakeReportConfidenceNodeID},
			},
			Nodes: []yaml.Node{
				{
					ID:        backlogTriggerNodeID,
					Name:      backlogTriggerName,
					Type:      yaml.NodeTypeTrigger,
					Component: factory.OnWorkOrderTriggerName,
					Position:  yaml.Position{X: 160, Y: 80},
				},
				{
					ID:            intakeAnalysisNodeID,
					Name:          intakeAnalysisNodeName,
					Type:          yaml.NodeTypeAction,
					Component:     request.Agent.component(),
					Configuration: intakeAnalysisConfiguration(spec, request.Agent),
					Concurrency:   intakeConcurrency(),
					Position:      yaml.Position{X: 160, Y: 260},
				},
				{
					ID:            intakeReportConfidenceNodeID,
					Name:          "Report Confidence",
					Type:          yaml.NodeTypeAction,
					Component:     intakeReportConfidenceComponent,
					Configuration: intakeConfidenceReportConfiguration(backlogAnalysisSubject),
					Concurrency:   intakeConcurrency(),
					Position:      yaml.Position{X: 160, Y: 440},
				},
			},
		},
	}
}
