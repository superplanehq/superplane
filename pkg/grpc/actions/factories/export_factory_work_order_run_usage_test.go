package factories

import (
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ExportFactoryWorkOrderRunUsage(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Publish draft", "", &r.User, nil, nil)
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
		FundingSource:   models.UsageFundingSourceBYOK,
		InputTokens:     1_000_000,
		TotalTokens:     1_000_000,
	}))
	require.NoError(t, models.RecordComputeUsage(db, models.ComputeUsageEventInput{
		OrganizationID:  r.Organization.ID,
		CanvasRunID:     *execution.RunID,
		NodeExecutionID: uuid.New(),
		NodeID:          "runner",
		MachineType:     "e1-large-amd64",
		FleetID:         "e1-large-amd64",
		DurationSeconds: 90,
		IdempotencyKey:  "runner:compute:export-usage:" + uuid.New().String(),
	}))

	// Push the ledger entries past the table's 30-day default and 366-day
	// maximum window, so only the unbounded export can see them.
	oldOccurredAt := time.Now().AddDate(-1, 0, 0)
	require.NoError(t, db.Model(&models.WorkspaceUsageEvent{}).
		Where("work_order_execution_id = ?", execution.ID).
		Update("occurred_at", oldOccurredAt).Error)

	windowed, err := ListFactoryWorkOrderRunUsage(context.Background(), r.Organization.ID.String(), &pb.ListFactoryWorkOrderRunUsageRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Zero(t, windowed.TotalCount, "table window should not see a row this old")

	resp, err := ExportFactoryWorkOrderRunUsage(context.Background(), r.Organization.ID.String(), &pb.ExportFactoryWorkOrderRunUsageRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, factory.Key+"-usage.csv", resp.Filename)

	records := parseWorkOrderRunUsageCSV(t, resp.Csv)
	require.Len(t, records, 2, "header plus one row")
	assert.Equal(t, workOrderRunUsageCSVHeader, records[0])

	row := records[1]
	assert.Equal(t, oldOccurredAt.UTC().Format(time.RFC3339), row[0])
	assert.Equal(t, r.UserModel.Name, row[1])
	assert.Equal(t, factory.WorkOrderKey(order.Number), row[2])
	assert.Equal(t, "anthropic/claude-sonnet-4-6 (your keys)", row[3])
	assert.Equal(t, "1000000", row[4])
	assert.Equal(t, "e1-large-amd64", row[6])
	assert.Equal(t, "90", row[7])

	tokenPriceCents, err := strconv.ParseFloat(row[5], 64)
	require.NoError(t, err)
	assert.Greater(t, tokenPriceCents, 0.0)

	vmPriceCents, err := strconv.ParseFloat(row[8], 64)
	require.NoError(t, err)
	assert.Greater(t, vmPriceCents, 0.0)
}

func Test__ExportFactoryWorkOrderRunUsage__NoRows(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	resp, err := ExportFactoryWorkOrderRunUsage(context.Background(), r.Organization.ID.String(), &pb.ExportFactoryWorkOrderRunUsageRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Equal(t, factory.Key+"-usage.csv", resp.Filename)

	records := parseWorkOrderRunUsageCSV(t, resp.Csv)
	require.Len(t, records, 1, "header only")
	assert.Equal(t, workOrderRunUsageCSVHeader, records[0])
}

func Test__WorkOrderRunUsageCSVRecord__UserFallsBackToEmail(t *testing.T) {
	item := &pb.WorkOrderRunUsageRow{UserEmail: "person@example.com"}
	record := workOrderRunUsageCSVRecord(item)
	assert.Equal(t, "person@example.com", record[1])
}

func Test__SanitizeUsageCSVCell(t *testing.T) {
	assert.Equal(t, "", sanitizeUsageCSVCell(""))
	assert.Equal(t, "Ada Lovelace", sanitizeUsageCSVCell("Ada Lovelace"))
	assert.Equal(t, "anthropic/claude-sonnet-4-6", sanitizeUsageCSVCell("anthropic/claude-sonnet-4-6"))

	// Formula triggers must be neutralized so spreadsheets treat them as text.
	for _, prefix := range []string{"=", "+", "-", "@", "\t", "\r"} {
		value := prefix + "cmd|'/C calc'!A1"
		assert.Equal(t, "'"+value, sanitizeUsageCSVCell(value), "prefix %q must be escaped", prefix)
	}
}

func Test__WorkOrderRunUsageCSVRecord__NeutralizesFormulaInjection(t *testing.T) {
	item := &pb.WorkOrderRunUsageRow{
		UserName:     "=HYPERLINK(\"http://evil\",\"click\")",
		WorkOrderKey: "@SUM(A1:A9)",
		Models:       []string{"-2+3"},
		MachineTypes: []string{"+bad"},
	}
	record := workOrderRunUsageCSVRecord(item)
	assert.Equal(t, "'=HYPERLINK(\"http://evil\",\"click\")", record[1])
	assert.Equal(t, "'@SUM(A1:A9)", record[2])
	assert.Equal(t, "'-2+3", record[3])
	assert.Equal(t, "'+bad", record[6])
}

func Test__FormatUsageCSVModels(t *testing.T) {
	assert.Equal(t, "", formatUsageCSVModels(nil, nil))
	assert.Equal(t, "anthropic/claude-sonnet-4-6", formatUsageCSVModels([]string{"anthropic/claude-sonnet-4-6"}, nil))
	assert.Equal(t, "anthropic/claude-sonnet-4-6 (your keys)", formatUsageCSVModels(nil, []string{"anthropic/claude-sonnet-4-6"}))
	assert.Equal(t,
		"anthropic/claude-haiku-4-5 · anthropic/claude-sonnet-4-6 (your keys)",
		formatUsageCSVModels([]string{"anthropic/claude-haiku-4-5"}, []string{"anthropic/claude-sonnet-4-6"}),
	)
}

func Test__FormatUsageCSVInt(t *testing.T) {
	assert.Equal(t, "", formatUsageCSVInt(0))
	assert.Equal(t, "42", formatUsageCSVInt(42))
}

func Test__FormatUsageCSVCents(t *testing.T) {
	assert.Equal(t, "", formatUsageCSVCents(0))
	assert.Equal(t, "1.23", formatUsageCSVCents(123))
	assert.Equal(t, "0.03", formatUsageCSVCents(3))
}

func parseWorkOrderRunUsageCSV(t *testing.T, body string) [][]string {
	t.Helper()
	require.True(t, strings.HasPrefix(body, utf8BOM), "csv must start with a UTF-8 BOM")
	body = strings.TrimPrefix(body, utf8BOM)
	records, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	return records
}
