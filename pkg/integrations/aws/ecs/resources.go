package ecs

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListClusters(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	clusters, err := client.ListClusters()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS clusters: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(clusters))
	for _, cluster := range clusters {
		name := strings.TrimSpace(cluster.ClusterName)
		if name == "" {
			name = clusterNameFromArn(cluster.ClusterArn)
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: name,
			ID:   cluster.ClusterArn,
		})
	}

	return resources, nil
}

func ListServices(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	cluster := strings.TrimSpace(ctx.Parameters["cluster"])
	if cluster == "" {
		return nil, fmt.Errorf("cluster is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	serviceArns, err := client.ListServices(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS services: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(serviceArns))
	for _, arn := range serviceArns {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: serviceNameFromArn(arn),
			ID:   arn,
		})
	}

	return resources, nil
}

func ListTaskDefinitions(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	taskDefinitionArns, err := client.ListTaskDefinitions()
	if err != nil {
		return nil, fmt.Errorf("failed to list ECS task definitions: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(taskDefinitionArns))
	for _, arn := range taskDefinitionArns {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: taskDefinitionNameFromArn(arn),
			ID:   arn,
		})
	}

	return resources, nil
}
