package compute

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/azure/common"
)

func ListVirtualMachines(ctx core.ListResourcesContext, resourceType string) ([]core.IntegrationResource, error) {
	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	subscriptionID := common.SubscriptionIDFromInstallation(ctx.Integration)
	if strings.TrimSpace(subscriptionID) == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}

	client := NewClient(ctx.HTTP, credentials, subscriptionID)
	vms, err := client.ListVMsBySubscription()
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	resources := make([]core.IntegrationResource, 0, len(vms))
	for _, vm := range vms {
		resources = append(resources, core.IntegrationResource{
			Type: resourceType,
			Name: vm.Name,
			ID:   vm.ID,
		})
	}

	return resources, nil
}
