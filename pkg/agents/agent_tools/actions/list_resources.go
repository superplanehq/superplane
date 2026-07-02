package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	organizationActions "github.com/superplanehq/superplane/pkg/grpc/actions/organizations"
	organizationpb "github.com/superplanehq/superplane/pkg/protos/organizations"
)

const (
	listResourcesActionName = "list_resources"
	defaultResourceLimit    = 100
	maxResourceLimit        = 200
)

type listResourcesAction struct {
	deps Dependencies
}

func newListResourcesAction(deps Dependencies) listResourcesAction {
	return listResourcesAction{deps: deps}
}

func (a listResourcesAction) Name() string {
	return listResourcesActionName
}

func (a listResourcesAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
	integrationID := strings.TrimSpace(input.IntegrationID)
	if integrationID == "" {
		return resourcesResult{}, fmt.Errorf("integration_id is required for list_resources")
	}

	parameters, resourceType, err := resourceParameters(input)
	if err != nil {
		return resourcesResult{}, err
	}

	response, err := organizationActions.ListIntegrationResources(
		ctx,
		a.deps.Registry,
		session.OrganizationID,
		integrationID,
		parameters,
	)
	if err != nil {
		return resourcesResult{}, err
	}

	limit := resourceLimit(input.Limit)
	resources, truncated := serializeAgentIntegrationResources(response.GetResources(), limit)

	return resourcesResult{
		Action:        listResourcesActionName,
		CanvasID:      session.CanvasID,
		IntegrationID: integrationID,
		ResourceType:  resourceType,
		Count:         len(response.GetResources()),
		Truncated:     truncated,
		Resources:     resources,
	}, nil
}

func resourceParameters(input Input) (map[string]string, string, error) {
	parameters := make(map[string]string, len(input.Parameters)+1)
	for key, value := range input.Parameters {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		parameters[key] = value
	}

	resourceType := strings.TrimSpace(input.ResourceType)
	if resourceType == "" {
		resourceType = strings.TrimSpace(parameters["type"])
	}
	if resourceType == "" {
		return nil, "", fmt.Errorf("resource_type is required for list_resources")
	}

	parameters["type"] = resourceType
	return parameters, resourceType, nil
}

func resourceLimit(input uint32) int {
	if input == 0 {
		return defaultResourceLimit
	}
	if input > maxResourceLimit {
		return maxResourceLimit
	}
	return int(input)
}

func serializeAgentIntegrationResources(resources []*organizationpb.IntegrationResourceRef, limit int) ([]integrationResourceResult, bool) {
	if limit < 0 {
		limit = 0
	}

	truncated := len(resources) > limit
	if truncated {
		resources = resources[:limit]
	}

	result := make([]integrationResourceResult, 0, len(resources))
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		result = append(result, integrationResourceResult{
			Type: resource.GetType(),
			ID:   resource.GetId(),
			Name: resource.GetName(),
		})
	}
	return result, truncated
}
