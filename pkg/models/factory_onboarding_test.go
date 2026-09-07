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
		assert.False(t, factory.IsInitialOnboarding())
		assert.Equal(t, models.FactoryOnboardingConfig{}, factory.OnboardingConfigValue())

		reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.Nil(t, reloaded.OnboardingCompletedAt)
		assert.Equal(t, models.FactoryOnboardingConfig{}, reloaded.OnboardingConfigValue())
	})

	t.Run("stores the initial onboarding attempt outside the description", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		attemptID := uuid.New()
		require.NoError(t, factory.SetInitialOnboardingAttempt(db, attemptID))

		assert.Empty(t, factory.Description)
		assert.True(t, factory.HasInitialOnboardingAttempt(attemptID))
		assert.True(t, factory.IsInitialOnboarding())

		reloaded, err := models.FindFactory(db, r.Organization.ID, factory.ID)
		require.NoError(t, err)
		assert.True(t, reloaded.HasInitialOnboardingAttempt(attemptID))
		assert.True(t, reloaded.IsInitialOnboarding())
		assert.True(t, reloaded.IsPendingInitialOnboarding())
	})

	t.Run("pending initial onboarding is false after complete", func(t *testing.T) {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		require.NoError(t, factory.SetInitialOnboardingAttempt(db, uuid.New()))
		assert.True(t, factory.IsPendingInitialOnboarding())

		require.NoError(t, factory.CompleteOnboarding(db, readyOnboardingPatch()))
		assert.False(t, factory.IsPendingInitialOnboarding())
		assert.True(t, factory.IsInitialOnboarding())
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
		defaultBranch := "main"
		issuesSource := models.FactoryOnboardingIssuesSourceVCS
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
			BacklogRepository: &backlogRepo,
			DefaultBranch:     &defaultBranch,
			IssuesSource:      &issuesSource,
		}))

		config := factory.OnboardingConfigValue()
		assert.Equal(t, vcsID, config.VCSIntegrationID)
		assert.Equal(t, "acme/api", config.AppRepository)
		assert.Equal(t, "acme/backlog", config.BacklogRepository)
		assert.Equal(t, "main", config.DefaultBranch)
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

func Test__OrganizationIDsPendingInitialOnboardingOnly(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	pendingOrg, err := models.CreateOrganization("pending-"+uuid.NewString(), "Pending Org")
	require.NoError(t, err)
	readyOrg, err := models.CreateOrganization("ready-"+uuid.NewString(), "Ready Org")
	require.NoError(t, err)

	pendingFactory, err := models.CreateFactory(db, pendingOrg.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, pendingFactory.SetInitialOnboardingAttempt(db, uuid.New()))

	readyFactory, err := models.CreateFactory(db, readyOrg.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, readyFactory.SetInitialOnboardingAttempt(db, uuid.New()))
	require.NoError(t, readyFactory.CompleteOnboarding(db, readyOnboardingPatch()))

	pending, err := models.OrganizationIDsPendingInitialOnboardingOnly(db, []uuid.UUID{pendingOrg.ID, readyOrg.ID, r.Organization.ID})
	require.NoError(t, err)

	_, pendingListed := pending[pendingOrg.ID]
	_, readyListed := pending[readyOrg.ID]
	_, setupListed := pending[r.Organization.ID]
	assert.True(t, pendingListed)
	assert.False(t, readyListed)
	assert.False(t, setupListed)
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
