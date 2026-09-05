package organizations

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/usage/pricebook"
)

const (
	spendingReportDefaultDays = 30
)

func DescribeOrganizationSpendingReport(
	ctx context.Context,
	orgID string,
	req *pb.DescribeOrganizationSpendingReportRequest,
) (*pb.DescribeOrganizationSpendingReportResponse, error) {
	organizationID, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	since, until, err := resolveSpendingReportWindow(req)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}

	kpiFilter := models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
		Until:          until,
	}

	explorerFilter := models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
		Until:          until,
	}
	if err := applySpendingReportFilters(&explorerFilter, req); err != nil {
		return nil, grpcerrors.InvalidArgument(err, err.Error())
	}
	explorerFilter.UsageKind = normalizeSpendingUsageKind(req.GetUsageKind())

	db := database.DB(ctx)

	kpi, err := models.SummarizeSpendingKPITotals(db, kpiFilter)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization spending report")
	}
	groupBy := strings.TrimSpace(req.GetGroupBy())
	grain := strings.TrimSpace(req.GetTimeGrain())

	explorer, err := models.SummarizeSpendingExplorer(db, explorerFilter, groupBy, grain)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization spending report")
	}

	catalogFilter := models.UsageReportFilter{
		OrganizationID: organizationID,
		Since:          since,
		Until:          until,
	}
	catalogs, err := models.ListSpendingFilterCatalogs(db, catalogFilter)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization spending report")
	}

	credit, err := models.DescribeOrganizationLLMCredit(db, organizationID)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to describe organization spending report")
	}

	billingEnabled, hasCustomer := billingState(ctx, organizationID)
	normalizedGroupBy := models.SpendingGroupByWorkspace
	switch groupBy {
	case models.SpendingGroupByUser:
		normalizedGroupBy = models.SpendingGroupByUser
	case models.SpendingGroupByModel:
		normalizedGroupBy = models.SpendingGroupByModel
	case models.SpendingGroupByMachine:
		normalizedGroupBy = models.SpendingGroupByMachine
	}

	return &pb.DescribeOrganizationSpendingReportResponse{
		KpiTotals:      serializeSpendingReportTotals(kpi),
		ExplorerTotals: serializeUsageTotals(explorer.Totals),
		Series:         serializeSpendingSeries(explorer.Series),
		SeriesKeys:     serializeSpendingSeriesKeys(explorer.SeriesKeys, catalogs, normalizedGroupBy),
		Breakdown:      serializeSpendingBreakdown(explorer.Breakdown, catalogs, normalizedGroupBy),
		Credit:         serializeSpendingCreditSnapshot(credit, billingEnabled, hasCustomer),
		Catalogs:       serializeSpendingCatalogs(catalogs),
	}, nil
}

func resolveSpendingReportWindow(req *pb.DescribeOrganizationSpendingReportRequest) (time.Time, time.Time, error) {
	now := time.Now()
	until := now
	if req.GetEndTime() != nil {
		until = req.GetEndTime().AsTime()
	}

	since := until.AddDate(0, 0, -spendingReportDefaultDays)
	if req.GetStartTime() != nil {
		since = req.GetStartTime().AsTime()
	}

	if err := models.ValidateSpendingReportWindow(since, until); err != nil {
		return time.Time{}, time.Time{}, err
	}
	return since, until, nil
}

func applySpendingReportFilters(filter *models.UsageReportFilter, req *pb.DescribeOrganizationSpendingReportRequest) error {
	if factoryID := strings.TrimSpace(req.GetFactoryId()); factoryID != "" {
		parsed, err := uuid.Parse(factoryID)
		if err != nil {
			return err
		}
		filter.FactoryID = &parsed
	}

	if model := strings.TrimSpace(req.GetModel()); model != "" {
		provider, modelName, ok := strings.Cut(model, "/")
		if !ok || provider == "" || modelName == "" {
			return fmt.Errorf("model must use provider/model format")
		}
		filter.Provider = provider
		filter.Model = modelName
	}

	if machineType := strings.TrimSpace(req.GetMachineType()); machineType != "" {
		filter.MachineType = machineType
	}

	if ownerID := strings.TrimSpace(req.GetTaskOwnerId()); ownerID != "" {
		parsed, err := uuid.Parse(ownerID)
		if err != nil {
			return err
		}
		filter.TaskOwnerID = &parsed
	}

	return nil
}

func normalizeSpendingUsageKind(value string) string {
	if strings.TrimSpace(value) == models.UsageKindCompute {
		return models.UsageKindCompute
	}
	return models.UsageKindModel
}

func serializeSpendingReportTotals(totals models.SpendingKPITotals) *pb.SpendingReportTotals {
	return &pb.SpendingReportTotals{
		CostCents:       totals.CostCents(),
		TotalTokens:     totals.TotalTokens,
		DurationSeconds: totals.DurationSeconds,
		HostedCostCents: totals.HostedCostCents(),
		ByokCostCents:   totals.BYOKCostCents(),
	}
}

func serializeUsageTotals(totals models.UsageTotals) *pb.SpendingReportTotals {
	return &pb.SpendingReportTotals{
		CostCents:       totals.CostCents(),
		TotalTokens:     totals.TotalTokens,
		DurationSeconds: totals.DurationSeconds,
	}
}

func serializeSpendingSeries(points []models.SpendingSeriesPoint) []*pb.SpendingReportSeriesPoint {
	out := make([]*pb.SpendingReportSeriesPoint, 0, len(points))
	for _, point := range points {
		seriesIDs := make([]string, 0, len(point.Values))
		for seriesID := range point.Values {
			seriesIDs = append(seriesIDs, seriesID)
		}
		sort.Strings(seriesIDs)

		values := make([]*pb.SpendingReportSeriesValue, 0, len(seriesIDs))
		for _, seriesID := range seriesIDs {
			values = append(values, &pb.SpendingReportSeriesValue{
				SeriesId:  seriesID,
				CostCents: point.Values[seriesID],
			})
		}
		out = append(out, &pb.SpendingReportSeriesPoint{
			Key:        point.Key,
			Label:      point.Label,
			TotalCents: point.TotalCents,
			Values:     values,
		})
	}
	return out
}

func serializeSpendingSeriesKeys(
	keys []models.SpendingSeriesKey,
	catalogs models.SpendingCatalogs,
	groupBy string,
) []*pb.SpendingReportCatalogItem {
	out := make([]*pb.SpendingReportCatalogItem, 0, len(keys))
	for _, key := range keys {
		label := key.Label
		if key.ID != "other" {
			label = models.SpendingBreakdownLabel(key.ID, catalogs, groupBy)
		}
		out = append(out, &pb.SpendingReportCatalogItem{
			Id:    key.ID,
			Label: label,
		})
	}
	return out
}

func serializeSpendingBreakdown(
	rows []models.SpendingBreakdownRow,
	catalogs models.SpendingCatalogs,
	groupBy string,
) []*pb.SpendingReportBreakdownRow {
	totalCost := int64(0)
	for _, row := range rows {
		totalCost += row.CostMicros
	}

	out := make([]*pb.SpendingReportBreakdownRow, 0, len(rows))
	for _, row := range rows {
		share := float64(0)
		if totalCost > 0 {
			share = float64(row.CostMicros) / float64(totalCost)
		}
		out = append(out, &pb.SpendingReportBreakdownRow{
			Id:              row.ID,
			Label:           models.SpendingBreakdownLabel(row.ID, catalogs, groupBy),
			TotalTokens:     row.TotalTokens,
			DurationSeconds: row.DurationSeconds,
			CostCents:       row.CostCents(),
			Share:           share,
		})
	}
	return out
}

func serializeSpendingCreditSnapshot(
	credit models.OrganizationLLMCreditSummary,
	billingEnabled bool,
	hasCustomer bool,
) *pb.SpendingReportCreditSnapshot {
	return &pb.SpendingReportCreditSnapshot{
		RemainingCreditCents:   pricebook.MicrosToCents(credit.RemainingMicros),
		GrantTotalCents:        pricebook.MicrosToCents(credit.GrantMicros),
		SuperplaneGrantCents:   pricebook.MicrosToCents(credit.SuperPlaneGrantMicros),
		PurchasedCreditCents:   pricebook.MicrosToCents(credit.PurchasedCreditMicros),
		HostedBilledCents:      pricebook.MicrosToCents(credit.BilledMicros),
		RemainingCreditWarning: credit.Warning,
		BillingEnabled:         billingEnabled,
		HasBillingCustomer:     hasCustomer,
	}
}

func serializeSpendingCatalogs(catalogs models.SpendingCatalogs) *pb.SpendingReportCatalogs {
	return &pb.SpendingReportCatalogs{
		Workspaces: serializeSpendingCatalogItems(catalogs.Workspaces),
		Users:      serializeSpendingCatalogItems(catalogs.Users),
		Models:     serializeSpendingCatalogItems(catalogs.Models),
		Machines:   serializeSpendingCatalogItems(catalogs.Machines),
	}
}

func serializeSpendingCatalogItems(items []models.SpendingCatalogItem) []*pb.SpendingReportCatalogItem {
	out := make([]*pb.SpendingReportCatalogItem, 0, len(items))
	for _, item := range items {
		out = append(out, &pb.SpendingReportCatalogItem{
			Id:    item.ID,
			Label: item.Label,
		})
	}
	return out
}
