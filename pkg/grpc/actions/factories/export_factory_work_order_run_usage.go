package factories

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
)

// utf8BOM marks the CSV as UTF-8 so Excel does not mangle non-ASCII names.
const utf8BOM = "\uFEFF"

var workOrderRunUsageCSVHeader = []string{
	"Date", "User", "Task", "Model", "Tokens", "Token price", "VM type", "Time", "VM price",
}

// ExportFactoryWorkOrderRunUsage returns the full task-run spend ledger for a
// workspace as CSV. Unlike ListFactoryWorkOrderRunUsage, it applies no date
// window and no page size, so the export always matches the ledger.
func ExportFactoryWorkOrderRunUsage(
	ctx context.Context,
	organizationID string,
	req *pb.ExportFactoryWorkOrderRunUsageRequest,
) (*pb.ExportFactoryWorkOrderRunUsageResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to export factory work order run usage")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to export factory work order run usage")
	}
	factoryID := factory.ID

	rows, err := models.ListAllWorkOrderRunUsage(db, models.UsageReportFilter{
		OrganizationID: orgID,
		FactoryID:      &factoryID,
	})
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to export factory work order run usage")
	}

	body, err := workOrderRunUsageCSV(factory, rows)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to export factory work order run usage")
	}

	return &pb.ExportFactoryWorkOrderRunUsageResponse{
		Csv:      body,
		Filename: workOrderRunUsageCSVFilename(factory),
	}, nil
}

func workOrderRunUsageCSVFilename(factory *models.Factory) string {
	key := strings.TrimSpace(factory.Key)
	if key == "" {
		key = "factory"
	}
	return fmt.Sprintf("%s-usage.csv", key)
}

// workOrderRunUsageCSV writes one row per task run, at the same grain as
// serializeWorkOrderRunUsageRow, so the export matches the table exactly.
func workOrderRunUsageCSV(factory *models.Factory, rows []models.WorkOrderRunUsage) (string, error) {
	var buf bytes.Buffer
	buf.WriteString(utf8BOM)

	writer := csv.NewWriter(&buf)
	if err := writer.Write(workOrderRunUsageCSVHeader); err != nil {
		return "", err
	}
	for _, row := range rows {
		item := serializeWorkOrderRunUsageRow(factory, row)
		if err := writer.Write(workOrderRunUsageCSVRecord(item)); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func workOrderRunUsageCSVRecord(item *pb.WorkOrderRunUsageRow) []string {
	user := item.GetUserName()
	if user == "" {
		user = item.GetUserEmail()
	}

	tokenPriceCents := item.GetHostedCostCents() + item.GetByokCostCents()
	vmPriceCents := item.GetCostCents() - tokenPriceCents
	if vmPriceCents < 0 {
		vmPriceCents = 0
	}

	return []string{
		formatUsageCSVDate(item.GetLastOccurredAt().AsTime()),
		sanitizeUsageCSVCell(user),
		sanitizeUsageCSVCell(item.GetWorkOrderKey()),
		sanitizeUsageCSVCell(formatUsageCSVModels(item.GetModels(), item.GetByokModels())),
		formatUsageCSVInt(item.GetTotalTokens()),
		formatUsageCSVCents(tokenPriceCents),
		sanitizeUsageCSVCell(formatUsageCSVMachineTypes(item.GetMachineTypes())),
		formatUsageCSVInt(item.GetDurationSeconds()),
		formatUsageCSVCents(vmPriceCents),
	}
}

// sanitizeUsageCSVCell neutralizes spreadsheet formula injection. User-supplied
// text (names, models, machine types, task keys) can start with a formula
// trigger, which Excel and other spreadsheets would otherwise execute on open,
// potentially issuing external requests or rendering deceptive links. Prefixing
// such a value with a single quote forces the cell to be treated as literal
// text. Numeric and date cells are produced by this package and never need it.
func sanitizeUsageCSVCell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}

func formatUsageCSVDate(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

// formatUsageCSVModels mirrors the table's hosted / "(your keys)" join, but
// leaves the cell empty (not an em dash) when there is nothing to show.
func formatUsageCSVModels(models, byokModels []string) string {
	hosted := strings.Join(models, " · ")
	if len(byokModels) == 0 {
		return hosted
	}

	byok := strings.Join(byokModels, " · ") + " (your keys)"
	if hosted == "" {
		return byok
	}
	return hosted + " · " + byok
}

func formatUsageCSVMachineTypes(machineTypes []string) string {
	return strings.Join(machineTypes, " · ")
}

// formatUsageCSVInt renders a spreadsheet-friendly integer. Zero means the
// row has nothing in that band (for example, a compute-only row has no
// tokens), so the cell stays empty instead of showing "0".
func formatUsageCSVInt(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

// formatUsageCSVCents renders cents as a plain decimal dollar amount, with no
// "$" prefix, so Excel treats it as a number. Zero stays empty for the same
// reason as formatUsageCSVInt.
func formatUsageCSVCents(cents int64) string {
	if cents <= 0 {
		return ""
	}
	return strconv.FormatFloat(float64(cents)/100, 'f', 2, 64)
}
