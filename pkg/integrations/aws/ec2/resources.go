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

	client := NewClient(ctx.HTTP, creds, region)

	imageOS := strings.TrimSpace(ctx.Parameters["imageOs"])
	if imageOS != "" {
		images, err := client.ListPublicImages(imageOS)
		if err != nil {
			return nil, fmt.Errorf("failed to list EC2 public images: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(images))
		for _, image := range images {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: publicImageResourceName(image),
				ID:   image.ImageID,
			})
		}

		return resources, nil
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

func ListInstanceTypes(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	instanceTypes, err := client.ListInstanceTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to list EC2 instance types: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(instanceTypes))
	for _, instanceType := range instanceTypes {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: instanceTypeResourceName(instanceType),
			ID:   instanceType.InstanceType,
		})
	}

	return resources, nil
}

func ListSubnets(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	subnets, err := client.ListSubnets()
	if err != nil {
		return nil, fmt.Errorf("failed to list EC2 subnets: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(subnets))
	for _, subnet := range subnets {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: subnetResourceName(subnet),
			ID:   subnet.SubnetID,
		})
	}

	return resources, nil
}

func ListSecurityGroups(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)

	var securityGroups []SecurityGroup
	subnetID := strings.TrimSpace(ctx.Parameters["subnetId"])
	if subnetID != "" {
		subnet, err := client.DescribeSubnet(subnetID)
		if err != nil {
			return nil, fmt.Errorf("failed to describe subnet: %w", err)
		}
		securityGroups, err = client.ListSecurityGroupsByVPC(subnet.VpcID)
		if err != nil {
			return nil, fmt.Errorf("failed to list EC2 security groups: %w", err)
		}
	} else {
		securityGroups, err = client.ListSecurityGroups()
		if err != nil {
			return nil, fmt.Errorf("failed to list EC2 security groups: %w", err)
		}
	}

	resources := make([]core.IntegrationResource, 0, len(securityGroups))
	for _, group := range securityGroups {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: securityGroupResourceName(group),
			ID:   group.GroupID,
		})
	}

	return resources, nil
}

func ListKeyPairs(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	keyPairs, err := client.ListKeyPairs()
	if err != nil {
		return nil, fmt.Errorf("failed to list EC2 key pairs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(keyPairs))
	for _, keyPair := range keyPairs {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: keyPair.KeyName,
			ID:   keyPair.KeyName,
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

func subnetResourceName(subnet Subnet) string {
	name := strings.TrimSpace(subnet.Name)
	if name == "" {
		return fmt.Sprintf("%s (%s)", subnet.SubnetID, subnet.CidrBlock)
	}

	return fmt.Sprintf("%s (%s)", name, subnet.SubnetID)
}

func securityGroupResourceName(group SecurityGroup) string {
	name := strings.TrimSpace(group.GroupName)
	if name == "" {
		return group.GroupID
	}

	return fmt.Sprintf("%s (%s)", name, group.GroupID)
}

func instanceTypeResourceName(instanceType InstanceTypeInfo) string {
	memoryGiB := float64(instanceType.MemoryMiB) / 1024
	if instanceType.MemoryMiB%1024 == 0 {
		return fmt.Sprintf("%s (%d vCPU, %d GiB Memory)", instanceType.InstanceType, instanceType.VCPUs, instanceType.MemoryMiB/1024)
	}

	return fmt.Sprintf("%s (%d vCPU, %.1f GiB Memory)", instanceType.InstanceType, instanceType.VCPUs, memoryGiB)
}
