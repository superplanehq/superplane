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

func ListPublicIPv4Pools(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	pools, err := client.ListPublicIPv4Pools()
	if err != nil {
		return nil, fmt.Errorf("failed to list public IPv4 pools: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(pools))
	for _, pool := range pools {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: publicIPv4PoolResourceName(pool),
			ID:   pool.PoolID,
		})
	}

	return resources, nil
}

func ListCustomerOwnedIPv4Pools(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	pools, err := client.ListCustomerOwnedIPv4Pools()
	if err != nil {
		return nil, fmt.Errorf("failed to list customer-owned IPv4 pools: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(pools))
	for _, pool := range pools {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: customerOwnedIPv4PoolResourceName(pool),
			ID:   pool.PoolID,
		})
	}

	return resources, nil
}

func ListIpamPools(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	pools, err := client.ListIpamPoolsForElasticIP()
	if err != nil {
		return nil, fmt.Errorf("failed to list IPAM pools: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(pools))
	for _, pool := range pools {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: ipamPoolResourceName(pool),
			ID:   pool.PoolID,
		})
	}

	return resources, nil
}

func ListElasticIPs(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	addresses, err := client.ListAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to list Elastic IPs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(addresses))
	for _, address := range addresses {
		if address.Domain != "vpc" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: elasticIPResourceName(address),
			ID:   address.AllocationID,
		})
	}

	return resources, nil
}

func ListUnassociatedElasticIPs(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	addresses, err := client.ListAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to list Elastic IPs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(addresses))
	for _, address := range addresses {
		if address.Domain != "vpc" {
			continue
		}

		if strings.TrimSpace(address.AssociationID) != "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: elasticIPResourceName(address),
			ID:   address.AllocationID,
		})
	}

	return resources, nil
}

func ListElasticIPAssociations(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	addresses, err := client.ListAddresses()
	if err != nil {
		return nil, fmt.Errorf("failed to list Elastic IP associations: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(addresses))
	for _, address := range addresses {
		if address.Domain != "vpc" {
			continue
		}

		associationID := strings.TrimSpace(address.AssociationID)
		if associationID == "" {
			continue
		}

		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: elasticIPAssociationResourceName(address),
			ID:   associationID,
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

func ListInstanceProfiles(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		region = "us-east-1"
	}

	client := NewClient(ctx.HTTP, creds, region)
	profiles, err := client.ListInstanceProfiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list IAM instance profiles: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(profiles))
	for _, profile := range profiles {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: profile.Name,
			ID:   profile.Name,
		})
	}

	return resources, nil
}

func ListAlarms(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	alarms, err := client.ListAlarms()
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch alarms: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(alarms))
	for _, alarm := range alarms {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: alarm.AlarmName,
			ID:   alarm.AlarmName,
		})
	}

	return resources, nil
}

func ListInstanceAlarms(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	instanceID := strings.TrimSpace(ctx.Parameters["instanceId"])
	if instanceID == "" {
		return nil, fmt.Errorf("instanceId is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	alarms, err := client.ListAlarmsForInstance(instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list CloudWatch alarms for instance %s: %w", instanceID, err)
	}

	resources := make([]core.IntegrationResource, 0, len(alarms))
	for _, alarm := range alarms {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: alarm.AlarmName,
			ID:   alarm.AlarmName,
		})
	}

	return resources, nil
}

func publicIPv4PoolResourceName(pool PublicIPv4Pool) string {
	description := strings.TrimSpace(pool.Description)
	if description == "" {
		return pool.PoolID
	}

	return fmt.Sprintf("%s (%s)", description, pool.PoolID)
}

func customerOwnedIPv4PoolResourceName(pool CustomerOwnedIPv4Pool) string {
	if routeTable := strings.TrimSpace(pool.LocalGatewayRouteTableID); routeTable != "" {
		return fmt.Sprintf("%s (%s)", pool.PoolID, routeTable)
	}

	return pool.PoolID
}

func ipamPoolResourceName(pool IpamPool) string {
	description := strings.TrimSpace(pool.Description)
	if description == "" {
		if locale := strings.TrimSpace(pool.Locale); locale != "" {
			return fmt.Sprintf("%s (%s)", pool.PoolID, locale)
		}

		return pool.PoolID
	}

	return fmt.Sprintf("%s (%s)", description, pool.PoolID)
}

func elasticIPResourceName(address ElasticIP) string {
	publicIP := strings.TrimSpace(address.PublicIP)
	if publicIP == "" {
		return address.AllocationID
	}

	return fmt.Sprintf("%s (%s)", publicIP, address.AllocationID)
}

func elasticIPAssociationResourceName(address ElasticIP) string {
	publicIP := strings.TrimSpace(address.PublicIP)
	instanceID := strings.TrimSpace(address.InstanceID)

	switch {
	case publicIP != "" && instanceID != "":
		return fmt.Sprintf("%s → %s (%s)", publicIP, instanceID, address.AssociationID)
	case publicIP != "":
		return fmt.Sprintf("%s (%s)", publicIP, address.AssociationID)
	case instanceID != "":
		return fmt.Sprintf("%s (%s)", instanceID, address.AssociationID)
	default:
		return address.AssociationID
	}
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

func ListLoadBalancers(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	loadBalancers, err := client.ListLoadBalancers()
	if err != nil {
		return nil, fmt.Errorf("failed to list load balancers: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(loadBalancers))
	for _, lb := range loadBalancers {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: loadBalancerResourceName(lb),
			ID:   lb.LoadBalancerARN,
		})
	}

	return resources, nil
}

func loadBalancerResourceName(lb LoadBalancer) string {
	name := strings.TrimSpace(lb.Name)
	if name == "" {
		return lb.LoadBalancerARN
	}

	return fmt.Sprintf("%s (%s)", name, lb.Type)
}

func ListTargetGroups(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(ctx.Parameters["region"])
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := NewClient(ctx.HTTP, creds, region)
	targetGroups, err := client.ListTargetGroups()
	if err != nil {
		return nil, fmt.Errorf("failed to list target groups: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(targetGroups))
	for _, tg := range targetGroups {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: targetGroupResourceName(tg),
			ID:   tg.TargetGroupARN,
		})
	}

	return resources, nil
}

func targetGroupResourceName(tg TargetGroup) string {
	name := strings.TrimSpace(tg.Name)
	if name == "" {
		return tg.TargetGroupARN
	}
	if tg.Protocol != "" && tg.Port > 0 {
		return fmt.Sprintf("%s (%s:%d)", name, tg.Protocol, tg.Port)
	}
	return name
}
