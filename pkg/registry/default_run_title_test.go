package registry_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"

	// Import server package which imports all components, triggers, and applications.
	_ "github.com/superplanehq/superplane/pkg/server"
)

func TestTriggerDefaultRunTitlesResolveWithExampleData(t *testing.T) {
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	for _, trigger := range allRegistryTriggers(reg) {
		t.Run(trigger.Name(), func(t *testing.T) {
			template := strings.TrimSpace(trigger.DefaultRunTitle())
			if template == "" {
				return
			}

			resolved, err := contexts.NewNodeConfigurationBuilder(nil, uuid.Nil).
				WithRootPayload(rootPayloadForExampleData(trigger.ExampleData())).
				ResolveTemplateExpressions(template)

			require.NoError(t, err)
			require.NotEmpty(t, strings.TrimSpace(resolved.(string)))
		})
	}
}

func rootPayloadForExampleData(exampleData map[string]any) map[string]any {
	if _, ok := exampleData["data"]; ok {
		return exampleData
	}

	return map[string]any{
		"type": "default",
		"data": exampleData,
	}
}

func allRegistryTriggers(reg *registry.Registry) []core.Trigger {
	triggers := reg.ListTriggers()
	for _, integration := range reg.ListIntegrations() {
		triggers = append(triggers, integration.Triggers()...)
	}

	return triggers
}
