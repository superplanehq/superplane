package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestListActiveBindingGroups(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)
	canvas := newCanvas(t, org.ID)
	webhook := newWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps)

	// Two active bindings under different scope keys — should produce two groups.
	newBinding(t, org.ID, integration.ID, canvas.ID, "node-1", "repo:owner/a", &webhook.ID, true)
	newBinding(t, org.ID, integration.ID, canvas.ID, "node-2", "repo:owner/b", &webhook.ID, true)

	// Inactive binding — must NOT appear as a group.
	newBinding(t, org.ID, integration.ID, canvas.ID, "node-3", "repo:owner/c", &webhook.ID, false)

	groups, err := models.ListActiveBindingGroups()
	require.NoError(t, err)
	assert.Len(t, groups, 2)

	scopes := make([]string, len(groups))
	for i, g := range groups {
		assert.Equal(t, integration.ID, g.AppInstallationID)
		scopes[i] = g.ScopeKey
	}
	assert.ElementsMatch(t, []string{"repo:owner/a", "repo:owner/b"}, scopes)
}

func TestListActiveBindingsForGroup(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org := newOrg(t)
	integration := newIntegration(t, org.ID)
	canvas := newCanvas(t, org.ID)
	webhook := newWebhook(t, &integration.ID, models.WebhookStateReady, models.WebhookProvisioningModeOps)
	const scopeKey = "repo:owner/repo"

	b1 := newBinding(t, org.ID, integration.ID, canvas.ID, "node-1", scopeKey, &webhook.ID, true)
	b2 := newBinding(t, org.ID, integration.ID, canvas.ID, "node-2", scopeKey, &webhook.ID, true)
	// Inactive — must NOT be returned.
	_ = newBinding(t, org.ID, integration.ID, canvas.ID, "node-3", scopeKey, &webhook.ID, false)

	bindings, err := models.ListActiveBindingsForGroup(integration.ID, scopeKey)
	require.NoError(t, err)
	require.Len(t, bindings, 2)

	// Both active bindings must be present (inactive one excluded).
	resultIDs := []uuid.UUID{bindings[0].ID, bindings[1].ID}
	assert.Contains(t, resultIDs, b1.ID)
	assert.Contains(t, resultIDs, b2.ID)
}
