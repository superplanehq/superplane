package agents

import (
	"context"

	"github.com/superplanehq/superplane/pkg/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func EnsureAgentTokensWithinLimits(ctx context.Context, usageService usage.Service, organizationID string) error {
	if usageService == nil || !usageService.Enabled() {
		return nil
	}

	response, err := usageService.DescribeOrganizationUsage(ctx, organizationID)
	if err != nil {
		return nil
	}

	bucket := response.GetUsage()
	if bucket == nil {
		return nil
	}

	capacity := bucket.AgentTokenBucketCapacity
	if capacity <= 0 {
		return nil
	}

	if bucket.AgentTokenBucketLevel >= capacity {
		return status.Error(codes.ResourceExhausted, "organization agent token limit exceeded")
	}

	return nil
}
