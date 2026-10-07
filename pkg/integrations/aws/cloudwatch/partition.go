package cloudwatch

import "strings"

// AWS partitions differ in more than ARNs: the API endpoint and the console
// live on different domains too. AllRegions offers cn-north-1 and
// cn-northwest-1, so the China partition is reachable and handled explicitly.
// GovCloud is not currently in AllRegions; its ARN partition is handled, but
// its console host would need adding here if those regions are ever offered.

const (
	partitionDefault  = "aws"
	partitionChina    = "aws-cn"
	partitionGovCloud = "aws-us-gov"
)

// PartitionForRegion returns the ARN partition a region belongs to.
func PartitionForRegion(region string) string {
	switch {
	case strings.HasPrefix(strings.TrimSpace(region), "cn-"):
		return partitionChina
	case strings.HasPrefix(strings.TrimSpace(region), "us-gov-"):
		return partitionGovCloud
	default:
		return partitionDefault
	}
}

// APIDNSSuffix returns the domain AWS service endpoints hang off in a region.
// GovCloud shares the default suffix; only China differs.
func APIDNSSuffix(region string) string {
	if PartitionForRegion(region) == partitionChina {
		return "amazonaws.com.cn"
	}

	return "amazonaws.com"
}

// ConsoleHost returns the AWS console host serving a region.
func ConsoleHost(region string) string {
	if PartitionForRegion(region) == partitionChina {
		return "console.amazonaws.cn"
	}

	return "console.aws.amazon.com"
}
