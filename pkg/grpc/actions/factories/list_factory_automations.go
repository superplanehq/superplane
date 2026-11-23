package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func ListFactoryAutomations(ctx context.Context, organizationID string, req *pb.ListFactoryAutomationsRequest) (*pb.ListFactoryAutomationsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	var canvases []models.Canvas
	if command := strings.TrimSpace(req.GetSkillName()); command != "" {
		if err := requireAgentResourceKindFeature(orgID, models.FactoryAgentResourceKindSkill); err != nil {
			return nil, factoryErrorToStatus(err, "failed to check skill usage")
		}
		canvases, err = factory.ListCanvasesMentioningSkill(db, command)
	} else {
		canvases, err = factory.ListCanvases(db)
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	automations := serializeFactoryAutomations(canvases)
	if err := applyAutomationTemplateIDs(db, canvases, automations); err != nil {
		return nil, factoryErrorToStatus(err, "failed to list factory automations")
	}

	return &pb.ListFactoryAutomationsResponse{
		Automations: automations,
	}, nil
}

func applyAutomationTemplateIDs(db *gorm.DB, canvases []models.Canvas, automations []*pb.Factory_Automation) error {
	ids := make([]uuid.UUID, 0, len(canvases))
	for i := range canvases {
		if canvases[i].LiveVersionID == nil {
			continue
		}
		ids = append(ids, canvases[i].ID)
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(db, ids)
	if err != nil {
		return err
	}
	byID := make(map[string]*pb.Factory_Automation, len(automations))
	for _, automation := range automations {
		byID[automation.GetId()] = automation
	}
	for id, spec := range specs {
		automation := byID[id.String()]
		if automation == nil {
			continue
		}
		automation.TemplateId = factoryAutomationTemplateID(spec)
	}
	return nil
}

func factoryAutomationTemplateID(spec models.LiveCanvasSpec) string {
	if id := models.FactoryAppTemplateID(spec.Nodes); id != "" {
		return id
	}
	if models.IsBacklogFactoryApp(spec.Nodes, spec.Edges) {
		return models.FactoryAppTemplateBacklogID
	}
	return ""
}
