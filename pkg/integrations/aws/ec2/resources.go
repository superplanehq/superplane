package ec2

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

func ListInstances(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	instances, err := client.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list EC2 instances: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(instances))
	for _, instance := range instances {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: instanceResourceName(instance),
			ID:   instance.InstanceID,
		})
	}

	return resources, nil
}

func instanceResourceName(instance Instance) string {
	name := strings.TrimSpace(instance.Name)
	if name == "" {
		return instance.InstanceID
	}

	return fmt.Sprintf("%s (%s)", name, instance.InstanceID)
}
