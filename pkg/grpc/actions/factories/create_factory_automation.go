package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

const factoryAutomationDefaultName = "Custom automation"

func CreateFactoryAutomation(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.CreateFactoryAutomationRequest,
) (*pb.CreateFactoryAutomationResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}

	db := database.DB(ctx)
	organization, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}
	templateID := strings.TrimSpace(req.GetTemplateId())
	if templateID != "" && templateID != scheduledIntakeTemplateID {
		return nil, factoryErrorToStatus(invalidArgument("unknown factory automation template"), "failed to create factory automation")
	}
	if templateID == scheduledIntakeTemplateID {
		if err := requireScheduledIntake(organization); err != nil {
			return nil, err
		}
	} else if !organization.HasExperimentalFeature(features.FeatureFactoryCustomAutomations) {
		return nil, factoryErrorToStatus(errCustomAutomationsDisabled, "failed to create factory automation")
	}

	columnKey := strings.TrimSpace(req.GetColumnKey())
	if templateID == scheduledIntakeTemplateID {
		columnKey = models.CanvasColumnKeyBacklog
	}
	if columnKey == models.CanvasColumnKeyBacklog && templateID != scheduledIntakeTemplateID {
		return nil, factoryErrorToStatus(invalidArgument("Use the scheduled intake template for Backlog automations."), "failed to create factory automation")
	}
	if columnKey != "" && !models.ValidCanvasColumnKey(columnKey) {
		return nil, factoryErrorToStatus(invalidArgument("column key must be backlog, verify, or done"), "failed to create factory automation")
	}

	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}
	factoryID := factory.ID

	name := strings.TrimSpace(req.GetName())
	if name == "" {
		name = factoryAutomationDefaultName
		if templateID == scheduledIntakeTemplateID {
			name = "Scheduled intake"
		}
	}
	name, err = models.AvailableCanvasName(db, orgID, &factoryID, name)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}

	var nodes []models.Node
	var edges []models.Edge
	description := ""
	if templateID == scheduledIntakeTemplateID {
		doc, buildErr := scheduledIntakeCanvas(db, factory, name, req.GetInstallParams())
		if buildErr != nil {
			return nil, factoryErrorToStatus(buildErr, "failed to configure scheduled intake")
		}
		nodes, edges, err = doc.Parse(deps.Registry, orgID.String())
		if err != nil {
			return nil, factoryErrorToStatus(err, "failed to build scheduled intake")
		}
		description = doc.Metadata.Description
	}
	response, err := canvases.CreateCanvas(
		ctx,
		deps.Registry,
		deps.Encryptor,
		deps.AuthService,
		deps.WebhookBaseURL,
		orgID,
		name,
		description,
		&factoryID,
		nodes,
		edges,
	)
	if err != nil {
		return nil, err
	}

	canvasID, err := uuid.Parse(response.GetCanvas().GetMetadata().GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}

	canvas, err := models.FindCanvasInTransaction(db, orgID, canvasID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to create factory automation")
	}
	if columnKey != "" {
		if err := canvas.SetColumnKey(db, columnKey); err != nil {
			discardIntakeCanvas(db, orgID, canvasID)
			return nil, factoryErrorToStatus(err, "failed to create factory automation")
		}
	}

	return &pb.CreateFactoryAutomationResponse{
		Automation: serializeFactoryAutomation(*canvas),
	}, nil
}
