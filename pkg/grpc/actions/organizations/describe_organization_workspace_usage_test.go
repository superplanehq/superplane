package organizations

import (
	"context"
	"encoding/json"
	"net/http"
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

func Test__DescribeOrganizationWorkspaceUsage(t *testing.T) {
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
		Provider:        models.UsageProviderOpenAI,
		Model:           "gpt-4o",
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-tiny-amd64",
		FleetID:         "e1-tiny-amd64",
		DurationSeconds: 45,
		IdempotencyKey:  "runner:compute:org-usage",
	}))

	resp, err := DescribeOrganizationWorkspaceUsage(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationWorkspaceUsageRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, int32(30), resp.PeriodDays)
	assert.Equal(t, int64(1_000_000), resp.TotalTokens)
	require.Len(t, resp.ByMachineType, 1)
	assert.Equal(t, int64(250)+resp.ByMachineType[0].CostCents, resp.TotalCostCents)
	require.Len(t, resp.ByModel, 1)
	assert.Equal(t, "openai", resp.ByModel[0].Provider)
	assert.Equal(t, int64(45), resp.TotalDurationSeconds)
	assert.Equal(t, "e1-tiny-amd64", resp.ByMachineType[0].MachineType)
	assert.Equal(t, int64(45), resp.ByMachineType[0].DurationSeconds)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.RemainingCreditCents)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.GrantTotalCents)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.SuperplaneGrantCents)
	assert.Equal(t, int64(0), resp.PurchasedCreditCents)
	assert.Equal(t, int64(0), resp.HostedBilledCents)
	assert.False(t, resp.RemainingCreditWarning)
	assert.Empty(t, resp.Invoices)
}

func Test__DescribeOrganizationWorkspaceUsageListsPolarInvoices(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	require.NoError(t, models.SetOrganizationPolarCustomerID(db, r.Organization.ID, "cust_1"))
	_, err := models.AddPolarLLMCreditGrant(db, r.Organization.ID, models.CentsToMicros(10000), uuid.NewString())
	require.NoError(t, err)

	server := polarAPIServer(t, func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/orders/", req.URL.Path)
		assert.Equal(t, r.Organization.ID.String(), req.URL.Query().Get("external_customer_id"))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{
					"id":           "ord_100",
					"created_at":   "2026-08-27T12:00:00Z",
					"status":       "paid",
					"total_amount": 10000,
					"product":      map[string]any{"name": "$100 pack"},
				},
			},
			"pagination": map[string]any{"max_page": 1},
		}))
	})
	usePolarTestServer(t, server)

	resp, err := DescribeOrganizationWorkspaceUsage(
		context.Background(),
		r.Organization.ID.String(),
		&pb.DescribeOrganizationWorkspaceUsageRequest{},
	)
	require.NoError(t, err)
	assert.Equal(t, models.DefaultWelcomeGrantCents, resp.SuperplaneGrantCents)
	assert.Equal(t, int64(10000), resp.PurchasedCreditCents)
	require.Len(t, resp.Invoices, 1)
	assert.Equal(t, "ord_100", resp.Invoices[0].Id)
	assert.Equal(t, int64(10000), resp.Invoices[0].AmountCents)
	assert.Equal(t, "paid", resp.Invoices[0].Status)
	assert.Equal(t, "$100 pack", resp.Invoices[0].ProductName)
}
