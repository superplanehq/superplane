package ec2

import (
	"fmt"
	"strings"
)

func requireRegion(value string) (string, error) {
	region := strings.TrimSpace(value)
	if region == "" {
		return "", fmt.Errorf("region is required")
	}

	return region, nil
}

func requireImageID(value string) (string, error) {
	imageID := strings.TrimSpace(value)
	if imageID == "" {
		return "", fmt.Errorf("image ID is required")
	}

	return imageID, nil
}

func requireInstanceID(value string) (string, error) {
	instanceID := strings.TrimSpace(value)
	if instanceID == "" {
		return "", fmt.Errorf("instance ID is required")
	}

	return instanceID, nil
}

func requireInstanceType(value string) (string, error) {
	instanceType := strings.TrimSpace(value)
	if instanceType == "" {
		return "", fmt.Errorf("instance type is required")
	}

	return instanceType, nil
}

func requireName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", fmt.Errorf("name is required")
	}

	return name, nil
}

func requireSubnetID(value string) (string, error) {
	subnetID := strings.TrimSpace(value)
	if subnetID == "" {
		return "", fmt.Errorf("subnet is required")
	}

	return subnetID, nil
}

func requireSecurityGroupID(value string) (string, error) {
	securityGroupID := strings.TrimSpace(value)
	if securityGroupID == "" {
		return "", fmt.Errorf("security group is required")
	}

	return securityGroupID, nil
}
