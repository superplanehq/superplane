package models_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__AdjustCreditBalanceSetsTrialRemainingAfterSpend(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	before := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, before.BilledMicros, int64(0))

	lowered, err := models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTrial,
		TargetMicros:   models.CentsToMicros(1000),
		ExpectedMicros: before.WelcomeRemainingMicros,
		Note:           "support",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)
	require.NotNil(t, lowered)
	assert.Equal(t, models.LLMCreditGrantKindTrialAdjustment, lowered.Kind)
	assert.Equal(t, models.CentsToMicros(1000)-before.WelcomeRemainingMicros, lowered.AmountMicros)
	require.NotNil(t, lowered.ExpiresAt)
	assert.Equal(t, models.CentsToMicros(1000), describeCredit(t, db, r.Organization.ID).WelcomeRemainingMicros)

	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTrial,
		TargetMicros:   models.CentsToMicros(6000),
		ExpectedMicros: models.CentsToMicros(1000),
	})
	require.NoError(t, err)

	after := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, models.CentsToMicros(6000), after.WelcomeRemainingMicros)
	assert.Equal(t, models.CentsToMicros(6000), after.RemainingMicros)
}

func Test__AdjustCreditBalanceLoweredTrialSpillsToGrant(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, err := models.AddAdminLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), "", nil)
	require.NoError(t, err)

	before := describeCredit(t, db, r.Organization.ID)
	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTrial,
		TargetMicros:   0,
		ExpectedMicros: before.WelcomeRemainingMicros,
	})
	require.NoError(t, err)

	execution := dispatchWorkOrderExecution(t, r)
	recordHostedPromptUsage(t, db, r.Organization.ID, requireExecutionRunID(t, execution))

	after := describeCredit(t, db, r.Organization.ID)
	require.Greater(t, after.BilledMicros, int64(0))
	assert.Equal(t, int64(0), after.WelcomeRemainingMicros)
	assert.Equal(t, models.CentsToMicros(10000)-after.BilledMicros, after.AdminRemainingMicros)
}

func Test__AdjustCreditBalanceTopupAndGrant(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTopup,
		TargetMicros:   models.CentsToMicros(2500),
		ExpectedMicros: models.CentsToMicros(10000),
	})
	require.NoError(t, err)
	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketGrant,
		TargetMicros:   models.CentsToMicros(700),
		ExpectedMicros: 0,
	})
	require.NoError(t, err)

	summary := describeCredit(t, db, r.Organization.ID)
	assert.Equal(t, models.CentsToMicros(2500), summary.PurchasedRemainingMicros)
	assert.Equal(t, models.CentsToMicros(2500), summary.PurchasedCreditMicros)
	assert.Equal(t, models.CentsToMicros(700), summary.AdminRemainingMicros)

	grants, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	kinds := make([]string, 0, len(grants))
	for _, grant := range grants {
		kinds = append(kinds, grant.Kind)
	}
	assert.Contains(t, kinds, models.LLMCreditGrantKindTopupAdjustment)
	assert.Contains(t, kinds, models.LLMCreditGrantKindAdminAdjustment)
}

func Test__AdjustCreditBalanceRejectsStaleExpectedBalance(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketGrant,
		TargetMicros:   models.CentsToMicros(500),
		ExpectedMicros: models.CentsToMicros(100),
	})
	require.ErrorIs(t, err, models.ErrCreditBalanceChanged)
	assert.Equal(t, int64(0), describeCredit(t, db, r.Organization.ID).AdminRemainingMicros)
}

func Test__AdjustCreditBalanceRejectsInvalidInput(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketGrant,
		TargetMicros:   -1,
	})
	require.ErrorIs(t, err, models.ErrCreditBalanceNegative)

	_, err = models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         "included",
		TargetMicros:   models.CentsToMicros(100),
	})
	require.ErrorIs(t, err, models.ErrCreditBalanceBucket)
}

func Test__AdjustCreditBalanceRejectsTrialWithoutLiveWelcome(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	expireWelcomeGrant(t, db, r.Organization.ID)

	_, err := models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTrial,
		TargetMicros:   models.CentsToMicros(1000),
		ExpectedMicros: 0,
	})
	require.ErrorIs(t, err, models.ErrTrialCreditNotActive)
}

func Test__AdjustCreditBalanceSkipsUnchangedBalance(t *testing.T) {
	restoreInstallationLLMSettings(t)
	r := support.Setup(t)
	db := database.DB(t.Context())
	before := describeCredit(t, db, r.Organization.ID)

	grant, err := models.AdjustOrganizationLLMCreditBalance(db, models.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         models.CreditBalanceBucketTrial,
		TargetMicros:   before.WelcomeRemainingMicros,
		ExpectedMicros: before.WelcomeRemainingMicros,
	})
	require.NoError(t, err)
	assert.Nil(t, grant)

	grants, err := models.ListOrganizationLLMCreditGrants(db, r.Organization.ID)
	require.NoError(t, err)
	assert.Len(t, grants, 1)
}

func describeCredit(t *testing.T, db *gorm.DB, orgID uuid.UUID) models.OrganizationLLMCreditSummary {
	t.Helper()
	summary, err := models.DescribeOrganizationLLMCredit(db, orgID)
	require.NoError(t, err)
	return summary
}
