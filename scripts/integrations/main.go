package main

import (
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/registryimports"
)

var _ = registryimports.Loaded

func main() {
	reg, err := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	integrations := reg.ListIntegrations()
	coreActions := reg.ListActions()
	coreTriggers := reg.ListTriggers()

	actionsCount := len(coreActions)
	integrationTriggers := 0
	for _, integration := range integrations {
		actionsCount += len(integration.Actions())
		integrationTriggers += len(integration.Triggers())
	}

	triggersCount := len(coreTriggers) + integrationTriggers
	componentsCount := actionsCount + triggersCount

	// +1 counts built-in "Core" components alongside registered integrations.
	integrationsCount := len(integrations) + 1

	fmt.Printf("Integrations: %d\n", integrationsCount)
	fmt.Printf("Components: %d\n", componentsCount)
	fmt.Printf("Triggers: %d\n", triggersCount)
	fmt.Printf("Actions: %d\n", actionsCount)
}
