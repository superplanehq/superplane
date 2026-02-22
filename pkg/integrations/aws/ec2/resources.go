package ec2

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
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

func ListImages(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	var includeDisabled bool
	if ctx.Parameters["includeDisabled"] == "true" {
		includeDisabled = true
	} else {
		includeDisabled = false
	}

	integrationMetadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &integrationMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if integrationMetadata.Session == nil {
		return nil, fmt.Errorf("integration account ID is not configured")
	}

	accountID := strings.TrimSpace(integrationMetadata.Session.AccountID)
	if accountID == "" {
		return nil, fmt.Errorf("integration account ID is not configured")
	}

	client := NewClient(ctx.HTTP, creds, region)
	images, err := client.ListImages(accountID, includeDisabled)
	if err != nil {
		return nil, fmt.Errorf("failed to list EC2 images: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(images))
	for _, image := range images {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: imageResourceName(image),
			ID:   image.ImageID,
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

func imageResourceName(image Image) string {
	name := strings.TrimSpace(image.Name)
	if name == "" {
		return image.ImageID
	}

	return fmt.Sprintf("%s (%s)", name, image.ImageID)
}
