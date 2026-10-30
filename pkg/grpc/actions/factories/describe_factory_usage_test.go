package factories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/ee/billing/pricebook"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__DescribeFactoryUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "ship", nil)
	require.NoError(t, err)

	app, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factory.ID, "build", "start")
	require.NoError(t, line.Update(db, nil, []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entry},
	}, nil))

	var execution *models.FactoryWorkOrderExecution
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, result, dispatchErr := line.Dispatch(tx, order)
		if dispatchErr != nil {
			return dispatchErr
		}
		execution = result.Execution
		return nil
	}))

	require.NotNil(t, execution.RunID)
	require.NoError(t, models.RecordUsage(db, models.WorkspaceUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderAnthropic,
		Model:           "claude-sonnet-4-6",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))
	const (
		computeMachineType = "e1-large-amd64"
		computeSeconds     = int64(300)
	)
	computeCostCents := pricebook.MicrosToCents(
		pricebook.EstimateComputeMicros(computeMachineType, computeMachineType, computeSeconds),
	)
	require.Greater(t, computeCostCents, int64(0))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     computeMachineType,
		FleetID:         computeMachineType,
		DurationSeconds: computeSeconds,
		IdempotencyKey:  "runner:compute:factory-usage:" + uuid.New().String(),
	}))

	resp, err := DescribeFactoryUsage(context.Background(), r.Organization.ID.String(), &pb.DescribeFactoryUsageRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, int32(30), resp.PeriodDays)
	assert.Equal(t, int64(1_000_000), resp.TotalTokens)
	require.Len(t, resp.ByMachineType, 1)
	assert.Equal(t, int64(300)+computeCostCents, resp.TotalCostCents)
	require.Len(t, resp.ByModel, 1)
	assert.Equal(t, "anthropic", resp.ByModel[0].Provider)
	assert.Equal(t, "claude-sonnet-4-6", resp.ByModel[0].Model)
	assert.Equal(t, computeSeconds, resp.TotalDurationSeconds)
	assert.Equal(t, computeMachineType, resp.ByMachineType[0].MachineType)
	assert.Equal(t, computeSeconds, resp.ByMachineType[0].DurationSeconds)
	assert.Equal(t, computeCostCents, resp.ByMachineType[0].CostCents)
}

func Test__DescribeFactoryUsageCreditGates(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	describe := func() *pb.DescribeFactoryUsageResponse {
		resp, err := DescribeFactoryUsage(context.Background(), r.Organization.ID.String(), &pb.DescribeFactoryUsageRequest{
			FactoryId: factory.ID.String(),
		})
		require.NoError(t, err)
		return resp
	}

	// An uncapped workspace follows the organization balance.
	resp := describe()
	assert.True(t, resp.HostedCreditAvailable)
	assert.True(t, resp.FactoryHostedBudgetAvailable)
	assert.Nil(t, resp.HostedSpendBudgetCents)

	// A sub-cent organization remainder rounds down to zero cents, but the
	// exact gate stays open, same as AssertHostedCreditAvailable.
	before, err := billingmodels.DescribeOrganizationLLMCredit(db, r.Organization.ID)
	require.NoError(t, err)
	_, err = billingmodels.AdjustOrganizationLLMCreditBalance(db, billingmodels.CreditBalanceAdjustment{
		OrganizationID: r.Organization.ID,
		Bucket:         billingmodels.CreditBalanceBucketTrial,
		TargetMicros:   5000,
		ExpectedMicros: before.WelcomeRemainingMicros,
		Note:           "support",
		ActorAccountID: &r.Account.ID,
	})
	require.NoError(t, err)

	resp = describe()
	assert.Zero(t, resp.RemainingCreditCents)
	assert.True(t, resp.HostedCreditAvailable)
	require.NoError(t, billingmodels.AssertHostedCreditAvailable(db, r.Organization.ID))

	// A capped workspace reports its own gate. A zero cap is exhausted even
	// though the organization still has credit.
	budgetCents := int64(0)
	require.NoError(t, factory.UpdateHostedSpendBudget(db, &budgetCents))

	resp = describe()
	require.NotNil(t, resp.HostedSpendBudgetCents)
	assert.True(t, resp.HostedCreditAvailable)
	assert.False(t, resp.FactoryHostedBudgetAvailable)
	assert.ErrorIs(t, billingmodels.AssertFactoryHostedBudgetAvailable(db, factory), billingmodels.ErrFactoryHostedBudgetEmpty)

	// Removing the cap opens the workspace gate again.
	require.NoError(t, factory.UpdateHostedSpendBudget(db, nil))
	resp = describe()
	assert.True(t, resp.FactoryHostedBudgetAvailable)
}
