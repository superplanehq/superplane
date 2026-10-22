package factories

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__ListFactories_IncludesLines(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	withLine, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	line, err := withLine.CreateLine(db, "plan-and-implement", nil)
	require.NoError(t, err)

	empty, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := ListFactories(ctx, r.Organization.ID.String())
	require.NoError(t, err)

	listedWithLine := listedFactoryByID(resp.Factories, withLine.ID.String())
	require.NotNil(t, listedWithLine)
	require.Len(t, listedWithLine.Lines, 1)
	assert.Equal(t, line.ID.String(), listedWithLine.Lines[0].Id)
	assert.Equal(t, "plan-and-implement", listedWithLine.Lines[0].Name)
	assert.Nil(t, listedWithLine.Lines[0].Metrics)

	listedEmpty := listedFactoryByID(resp.Factories, empty.ID.String())
	require.NotNil(t, listedEmpty)
	assert.Empty(t, listedEmpty.Lines)
}

func Test__ListFactories_ResolvesAgentSourcesFromOneConnectionLookup(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	olderClaude := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	newerClaude := createReadyOnboardingIntegration(t, r.Organization.ID, "claude")
	olderOpenRouter := createReadyOnboardingIntegration(t, r.Organization.ID, "openrouter")
	savedOpenRouter := createReadyOnboardingIntegration(t, r.Organization.ID, "openrouter")
	pendingOpenRouter, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "openrouter", support.RandomName("openrouter"), map[string]any{})
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", olderClaude).Update("created_at", time.Now().Add(-3*time.Hour)).Error)
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", newerClaude).Update("created_at", time.Now().Add(-2*time.Hour)).Error)
	require.NoError(t, db.Model(&models.Integration{}).Where("id = ?", olderOpenRouter).Update("created_at", time.Now().Add(-time.Hour)).Error)

	saved := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessClaudeCode, savedOpenRouter)
	fallback := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessClaudeCode, "")
	unspecified := createListedFactory(t, db, r.Organization.ID, "", "")
	missing := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessClaudeCode, uuid.NewString())
	pending := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessClaudeCode, pendingOpenRouter.ID.String())
	hosted := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessSuperPlane, "")
	cursor := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessCursor, "")
	codex := createListedFactory(t, db, r.Organization.ID, models.FactoryOnboardingAgentHarnessCodex, "")

	queries := 0
	const callback = "test:list_factories_agent_connections"
	require.NoError(t, database.Conn().Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if integrationQueryForOrganization(tx, r.Organization.ID) {
			queries++
		}
	}))
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Callback().Query().Remove(callback))
	})

	resp, err := ListFactories(ctx, r.Organization.ID.String())
	require.NoError(t, err)
	assert.Equal(t, 1, queries)

	assert.Equal(t, "openrouter", listedFactoryByID(resp.Factories, saved.ID.String()).GetAgentModelSource())
	assert.Equal(t, "anthropic", listedFactoryByID(resp.Factories, fallback.ID.String()).GetAgentModelSource())
	assert.Equal(t, "anthropic", listedFactoryByID(resp.Factories, unspecified.ID.String()).GetAgentModelSource())
	assert.Equal(t, "anthropic", listedFactoryByID(resp.Factories, missing.ID.String()).GetAgentModelSource())
	assert.Equal(t, "openrouter", listedFactoryByID(resp.Factories, pending.ID.String()).GetAgentModelSource())
	assert.Equal(t, "hosted", listedFactoryByID(resp.Factories, hosted.ID.String()).GetAgentModelSource())
	assert.Empty(t, listedFactoryByID(resp.Factories, cursor.ID.String()).GetAgentModelSource())
	assert.Equal(t, "openai", listedFactoryByID(resp.Factories, codex.ID.String()).GetAgentModelSource())
}

func createListedFactory(t *testing.T, db *gorm.DB, organizationID uuid.UUID, harness, integrationID string) *models.Factory {
	t.Helper()
	factory, err := models.CreateFactory(db, organizationID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	if harness == "" && integrationID == "" {
		return factory
	}
	patch := models.FactoryOnboardingPatch{}
	if harness != "" {
		patch.AgentHarness = &harness
	}
	if integrationID != "" {
		patch.AgentIntegrationID = &integrationID
	}
	require.NoError(t, factory.UpdateOnboarding(db, patch))
	return factory
}

func integrationQueryForOrganization(tx *gorm.DB, organizationID uuid.UUID) bool {
	if tx.Statement == nil || !strings.Contains(tx.Statement.SQL.String(), "app_installations") {
		return false
	}
	return varsContainOrganization(tx.Statement.Vars, organizationID)
}

func varsContainOrganization(vars []any, organizationID uuid.UUID) bool {
	for _, arg := range vars {
		switch value := arg.(type) {
		case uuid.UUID:
			if value == organizationID {
				return true
			}
		case []uuid.UUID:
			for _, id := range value {
				if id == organizationID {
					return true
				}
			}
		case string:
			if value == organizationID.String() {
				return true
			}
		}
	}
	return false
}

func listedFactoryByID(factories []*pb.Factory, id string) *pb.Factory {
	for _, factory := range factories {
		if factory.GetId() == id {
			return factory
		}
	}
	return nil
}
