package registry_test

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/registry"

	// Import server package which imports all components, triggers, and applications
	_ "github.com/superplanehq/superplane/pkg/server"
)

// isCamelCase checks if a string follows camelCase naming convention
func isCamelCase(s string) bool {
	if len(s) == 0 || !unicode.IsLower(rune(s[0])) {
		return false
	}
	return !strings.ContainsAny(s, "_-")
}

// isValidName validates component/trigger names (simple or dotted like "app.name" or "app.sub.name")
func isValidName(name string) bool {
	parts := strings.Split(name, ".")
	if len(parts) > 3 {
		return false
	}
	for _, part := range parts {
		if !isCamelCase(part) {
			return false
		}
	}
	return true
}

func TestComponentsAndTriggersUseCamelCaseNames(t *testing.T) {
	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	for _, c := range reg.ListComponents() {
		assert.True(t, isValidName(c.Name()), "Component %q is not camelCase", c.Name())
	}

	for _, tr := range reg.ListTriggers() {
		assert.True(t, isValidName(tr.Name()), "Trigger %q is not camelCase", tr.Name())
	}

	for _, integration := range reg.ListIntegrations() {
		assert.True(t, isCamelCase(integration.Name()), "Integration %q is not camelCase", integration.Name())

		for _, c := range integration.Components() {
			assert.True(t, isValidName(c.Name()), "Component %q is not camelCase", c.Name())
		}
		for _, tr := range integration.Triggers() {
			assert.True(t, isValidName(tr.Name()), "Trigger %q is not camelCase", tr.Name())
		}
	}
}
