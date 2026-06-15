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

func requireAlarmName(value string) (string, error) {
	alarmName := strings.TrimSpace(value)
	if alarmName == "" {
		return "", fmt.Errorf("alarm name is required")
	}

	return alarmName, nil
}

func requireMetricName(value string) (string, error) {
	metricName := strings.TrimSpace(value)
	if metricName == "" {
		return "", fmt.Errorf("metric name is required")
	}

	return metricName, nil
}

func requireComparisonOperator(value string) (string, error) {
	comparisonOperator := strings.TrimSpace(value)
	if comparisonOperator == "" {
		return "", fmt.Errorf("comparison operator is required")
	}

	return comparisonOperator, nil
}

func requireStatistic(value string) (string, error) {
	statistic := strings.TrimSpace(value)
	if statistic == "" {
		return "", fmt.Errorf("statistic is required")
	}

	return statistic, nil
}

func hasConfigKey(configuration any, key string) bool {
	configurationMap, ok := configuration.(map[string]any)
	if !ok {
		return false
	}

	_, exists := configurationMap[key]
	return exists
}

func requireThreshold(configuration any, threshold float64) (float64, error) {
	if !hasConfigKey(configuration, "threshold") {
		return 0, fmt.Errorf("threshold is required")
	}

	return threshold, nil
}

func requireAllocationID(value string) (string, error) {
	allocationID := strings.TrimSpace(value)
	if allocationID == "" {
		return "", fmt.Errorf("allocation ID is required")
	}

	return allocationID, nil
}

func requireAssociationID(value string) (string, error) {
	associationID := strings.TrimSpace(value)
	if associationID == "" {
		return "", fmt.Errorf("association ID is required")
	}

	return associationID, nil
}

func requirePublicIPv4Pool(value string) (string, error) {
	poolID := strings.TrimSpace(value)
	if poolID == "" {
		return "", fmt.Errorf("public IPv4 pool is required")
	}

	return poolID, nil
}

func requireCustomerOwnedIPv4Pool(value string) (string, error) {
	poolID := strings.TrimSpace(value)
	if poolID == "" {
		return "", fmt.Errorf("customer-owned pool is required")
	}

	return poolID, nil
}

func requireIpamPoolID(value string) (string, error) {
	poolID := strings.TrimSpace(value)
	if poolID == "" {
		return "", fmt.Errorf("IPAM pool is required")
	}

	return poolID, nil
}

func requireLoadBalancerARN(value string) (string, error) {
	arn := strings.TrimSpace(value)
	if arn == "" {
		return "", fmt.Errorf("load balancer ARN is required")
	}

	return arn, nil
}
