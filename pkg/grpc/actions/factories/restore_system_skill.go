package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

func RestoreFactorySystemSkill(ctx context.Context, organizationID string, req *pb.RestoreFactorySystemSkillRequest) (*pb.RestoreFactorySystemSkillResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore system skill")
	}
	resourceID, err := uuid.Parse(strings.TrimSpace(req.GetResourceId()))
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid resource id"), "failed to restore system skill")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore system skill")
	}
	resource, err := factory.FindAgentResource(db, resourceID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore system skill")
	}
	if err := requireAgentResourceKindFeature(orgID, models.FactoryAgentResourceKindSkill); err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore system skill")
	}
	if !resource.IsSystemSkill() {
		return nil, factoryErrorToStatus(models.ErrFactorySystemSkillRequired, "failed to restore system skill")
	}
	template, ok := runner.SystemSkillTemplateByID(resource.Config.Data().SystemSkill.TemplateID)
	if !ok {
		return nil, grpcerrors.FailedPrecondition(nil, "This system skill template is unavailable in this SuperPlane version.")
	}
	if err := resource.RestoreSystemSkill(db, template.Version, template.Markdown(resource.Name)); err != nil {
		return nil, factoryErrorToStatus(err, "failed to restore system skill")
	}
	return &pb.RestoreFactorySystemSkillResponse{Resource: serializeFactoryAgentResource(resource)}, nil
}

func serializeSystemSkill(resource *models.FactoryAgentResource) *pb.FactoryAgentResource_SystemSkill {
	config := resource.Config.Data()
	metadata := config.SystemSkill
	out := &pb.FactoryAgentResource_SystemSkill{TemplateId: metadata.TemplateID, Version: metadata.Version, State: pb.FactoryAgentResource_SystemSkill_STATE_UNAVAILABLE}
	template, ok := runner.SystemSkillTemplateByID(metadata.TemplateID)
	if !ok {
		return out
	}
	out.LatestVersion = template.Version
	latest := models.FactoryAgentResourceConfig{Markdown: template.Markdown(resource.Name)}
	switch hash := config.SkillContentHash(); {
	case hash == latest.SkillContentHash():
		out.State = pb.FactoryAgentResource_SystemSkill_STATE_LATEST
	case hash != metadata.ContentHash:
		out.State = pb.FactoryAgentResource_SystemSkill_STATE_CUSTOM
	default:
		out.State = pb.FactoryAgentResource_SystemSkill_STATE_UPDATE_AVAILABLE
	}
	return out
}
