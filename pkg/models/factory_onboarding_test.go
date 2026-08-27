package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__FactoryOnboarding(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	t.Run("new factory starts incomplete with empty config", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		assert.Nil(t, factory.OnboardingCompletedAt)
		assert.False(t, factory.IsOnboardingComplete())
		assert.Equal(t, models.FactoryOnboardingConfig{}, factory.OnboardingConfigValue())

		reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Nil(t, reloaded.OnboardingCompletedAt)
		assert.Equal(t, models.FactoryOnboardingConfig{}, reloaded.OnboardingConfigValue())
	})

	t.Run("partial update merges fields", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		vcsID := uuid.New().String()
		appRepo := "acme/api"
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			VCSIntegrationID: &vcsID,
			AppRepository:    &appRepo,
		}))

		assert.Nil(t, factory.OnboardingCompletedAt)
		assert.Equal(t, vcsID, factory.OnboardingConfigValue().VCSIntegrationID)
		assert.Equal(t, "acme/api", factory.OnboardingConfigValue().AppRepository)

		backlogRepo := "acme/backlog"
		issuesSource := models.FactoryOnboardingIssuesSourceVCS
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			BacklogRepository: &backlogRepo,
			IssuesSource:      &issuesSource,
		}))

		config := factory.OnboardingConfigValue()
		assert.Equal(t, vcsID, config.VCSIntegrationID)
		assert.Equal(t, "acme/api", config.AppRepository)
		assert.Equal(t, "acme/backlog", config.BacklogRepository)
		assert.Equal(t, models.FactoryOnboardingIssuesSourceVCS, config.IssuesSource)
	})

	t.Run("rejects invalid enum and uuid values", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		badSource := "github-issues"
		err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{IssuesSource: &badSource})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingInvalidIssuesSource)

		badHarness := "windsurf"
		err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AgentHarness: &badHarness})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingInvalidAgentHarness)

		badID := "not-a-uuid"
		err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &badID})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingInvalidIntegrationID)

		err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{ProvisionedAppID: &badID})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingInvalidAppID)

		badRepository := "missing-owner"
		err = factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: &badRepository})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingInvalidRepository)
	})

	t.Run("complete requires ready config and is idempotent", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		err = factory.CompleteOnboarding(db, models.FactoryOnboardingPatch{})
		assert.ErrorIs(t, err, models.ErrFactoryOnboardingAppRepositoryRequired)

		ready := readyOnboardingPatch()
		require.NoError(t, factory.CompleteOnboarding(db, ready))
		require.NotNil(t, factory.OnboardingCompletedAt)
		firstCompletedAt := *factory.OnboardingCompletedAt

		time.Sleep(2 * time.Millisecond)
		require.NoError(t, factory.CompleteOnboarding(db, models.FactoryOnboardingPatch{}))
		require.NotNil(t, factory.OnboardingCompletedAt)
		assert.Equal(t, firstCompletedAt, *factory.OnboardingCompletedAt)
		assert.True(t, factory.IsOnboardingComplete())
	})

	t.Run("complete allows empty agent integration", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		ready := readyOnboardingPatch()
		ready.AgentIntegrationID = nil
		require.NoError(t, factory.CompleteOnboarding(db, ready))
		assert.Empty(t, factory.OnboardingConfigValue().AgentIntegrationID)
	})
}

func readyOnboardingPatch() models.FactoryOnboardingPatch {
	vcsID := uuid.New().String()
	agentID := uuid.New().String()
	appRepo := "acme/api"
	backlogRepo := "acme/backlog"
	issuesSource := models.FactoryOnboardingIssuesSourceSkip
	agentHarness := models.FactoryOnboardingAgentHarnessClaudeCode
	appID := uuid.New().String()
	lineID := uuid.New().String()

	return models.FactoryOnboardingPatch{
		VCSIntegrationID:   &vcsID,
		AgentIntegrationID: &agentID,
		AppRepository:      &appRepo,
		BacklogRepository:  &backlogRepo,
		IssuesSource:       &issuesSource,
		AgentHarness:       &agentHarness,
		ProvisionedAppID:   &appID,
		ProvisionedLineID:  &lineID,
	}
}
