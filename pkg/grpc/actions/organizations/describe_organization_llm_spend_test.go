package organizations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DescribeOrganizationLLMSpend(t *testing.T) {
	_, err := models.UpdateInstallationLLMSettings(database.Conn(), models.InstallationLLMSettings{
		WelcomeGrantCents:   models.DefaultWelcomeGrantCents,
		MarkupBPS:           models.DefaultMarkupBPS,
		WarningThresholdBPS: models.DefaultWarningThresholdBPS,
	})
	require.NoError(t, err)
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
	}))

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
	require.NoError(t, models.RecordUsage(db, models.LLMUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "prompt",
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))

	resp, err := DescribeOrganizationLLMSpend(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationLLMSpendRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, int32(30), resp.PeriodDays)
	assert.Equal(t, int64(1_000_000), resp.TotalTokens)
	assert.Equal(t, int64(250), resp.TotalCostCents)
	require.Len(t, resp.ByModel, 1)
	assert.Equal(t, "openai", resp.ByModel[0].Provider)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.RemainingCreditCents)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.GrantTotalCents)
	assert.Equal(t, int64(0), resp.HostedBilledCents)
	assert.False(t, resp.RemainingCreditWarning)
}
