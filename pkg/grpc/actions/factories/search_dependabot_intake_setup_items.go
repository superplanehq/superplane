package factories

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// SearchDependabotIntakeSetupItems lists open alert packages for the
// workspace backlog repository before a Dependabot intake exists. The setup
// wizard uses this on step two so step one does not create a live intake.
func SearchDependabotIntakeSetupItems(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SearchDependabotIntakeSetupItemsRequest,
) (*pb.SearchFactoryIntakeItemsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search dependabot setup items")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search dependabot setup items")
	}

	severities := normalizeDependabotSeverities(req.GetDependabotSeverities())
	source, err := newDependabotIntakeItemSourceForFactory(ctx, deps, db, factory, severities)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search dependabot setup items")
	}

	items, err := source.Search(ctx, "", intakeItemLimit("", int(req.GetLimit())))
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search dependabot setup items")
	}

	serialized := make([]*pb.FactoryIntakeItem, 0, len(items))
	for _, item := range items {
		serialized = append(serialized, serializeFactoryIntakeItem(item))
	}

	return &pb.SearchFactoryIntakeItemsResponse{Items: serialized}, nil
}

func newDependabotIntakeItemSourceForFactory(
	_ context.Context,
	deps IntakeDependencies,
	tx *gorm.DB,
	factory *models.Factory,
	severities []string,
) (*dependabotIntakeItemSource, error) {
	binding, err := resolveIntakeBinding(tx, factory, models.FactoryIntakeSourceDependabotAlerts, "", "")
	if err != nil {
		return nil, err
	}
	if binding == nil || binding.Installation == nil {
		return nil, errIntakeNotConnected
	}

	repository, _ := binding.Configuration["repository"].(string)
	client, err := newIntakeGitHubClient(deps, tx, binding.Installation)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errIntakeNotConnected, err)
	}

	return &dependabotIntakeItemSource{
		github:     client,
		repository: strings.TrimSpace(repository),
		severities: severities,
	}, nil
}
