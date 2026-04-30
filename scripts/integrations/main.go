package main

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"

	// Import server to auto-register all integrations, components, and triggers via init().
	_ "github.com/superplanehq/superplane/pkg/server"
)

func main() {
	reg, _ := registry.NewRegistry(crypto.NewNoOpEncryptor(), registry.HTTPOptions{})
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

	integrationsCount := len(integrations) + 1

	fmt.Printf("Integrations: %d\n", integrationsCount)
	fmt.Printf("Components: %d\n", componentsCount)
	fmt.Printf("Triggers: %d\n", triggersCount)
	fmt.Printf("Actions: %d\n", actionsCount)
}
