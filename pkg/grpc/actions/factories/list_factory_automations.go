package factories

import (
	"context"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
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

	return &pb.ListFactoryAutomationsResponse{
		Automations: serializeFactoryAutomations(canvases),
	}, nil
}
