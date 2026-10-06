package factories

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// AdminVelocityFactory is one workspace an installation admin can select.
// It carries only the id and name the selector needs.
type AdminVelocityFactory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AdminVelocityTotals matches the customer total fields the report mapper reads.
type AdminVelocityTotals struct {
	SuperplaneMerged int32 `json:"superplaneMerged"`
	PeopleMerged     int32 `json:"peopleMerged"`
	Waste            int32 `json:"waste"`
	CostCents        int64 `json:"costCents"`
	Tokens           int64 `json:"tokens"`
	WasteCostCents   int64 `json:"wasteCostCents"`
	TasksClosed      int32 `json:"tasksClosed"`
	TasksWaste       int32 `json:"tasksWaste"`
	ModelCostCents   int64 `json:"modelCostCents"`
	ComputeCostCents int64 `json:"computeCostCents"`
}

// AdminVelocityIntakeCount is one intake series on a day.
type AdminVelocityIntakeCount struct {
	Key    string `json:"key"`
	Merged int32  `json:"merged"`
}

// AdminVelocityPoint is one day of the delivery and cost charts.
type AdminVelocityPoint struct {
	Day                        string                     `json:"day"`
	SuperplaneMerged           int32                      `json:"superplaneMerged"`
	PeopleMerged               int32                      `json:"peopleMerged"`
	Waste                      int32                      `json:"waste"`
	Intake                     []AdminVelocityIntakeCount `json:"intake"`
	CostCents                  int64                      `json:"costCents"`
	Tokens                     int64                      `json:"tokens"`
	WasteCostCents             int64                      `json:"wasteCostCents"`
	ModelCostCents             int64                      `json:"modelCostCents"`
	ComputeCostCents           int64                      `json:"computeCostCents"`
	MedianTaskModelCostCents   int64                      `json:"medianTaskModelCostCents"`
	MedianTaskComputeCostCents int64                      `json:"medianTaskComputeCostCents"`
}

// AdminVelocityIntakeSource is one intake category for the period.
type AdminVelocityIntakeSource struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Merged int32  `json:"merged"`
}

// AdminVelocityAutomation is one canvas that ran in the period.
type AdminVelocityAutomation struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Runs                 int32   `json:"runs"`
	Failed               int32   `json:"failed"`
	CostCents            int64   `json:"costCents"`
	AverageDurationHours float64 `json:"averageDurationHours"`
}

// AdminVelocityReport is the selected workspace report.
// It omits people, the repository, sync state, yesterday, and cycle time.
type AdminVelocityReport struct {
	FactoryID         string                      `json:"factoryId"`
	PeriodDays        int                         `json:"periodDays"`
	HasPeopleCohort   bool                        `json:"hasPeopleCohort"`
	HasPreviousWindow bool                        `json:"hasPreviousWindow"`
	Totals            AdminVelocityTotals         `json:"totals"`
	PreviousTotals    AdminVelocityTotals         `json:"previousTotals"`
	Points            []AdminVelocityPoint        `json:"points"`
	IntakeSources     []AdminVelocityIntakeSource `json:"intakeSources"`
	Automations       []AdminVelocityAutomation   `json:"automations"`
}

// AdminOrganizationVelocity is the installation admin velocity response.
// A nil report means the organization has no workspace to select.
type AdminOrganizationVelocity struct {
	Factories []AdminVelocityFactory `json:"factories"`
	*AdminVelocityReport
}

// DescribeAdminOrganizationVelocity loads one workspace velocity report for
// Installation Admin. It reuses the velocity loader and does not build people.
//
// An omitted factoryID selects the workspace updated most recently. A missing,
// foreign, deleted, or malformed id is not found. An organization with no
// workspaces returns an empty list and no report.
func DescribeAdminOrganizationVelocity(
	ctx context.Context,
	organizationID uuid.UUID,
	factoryID string,
	periodDays int,
) (*AdminOrganizationVelocity, error) {
	db := database.DB(ctx)
	factories, err := models.ListOrganizationFactoriesByRecentUpdate(db, organizationID)
	if err != nil {
		return nil, err
	}

	response := &AdminOrganizationVelocity{
		Factories: adminVelocityFactories(factories),
	}
	selected, err := selectAdminVelocityFactory(factories, factoryID)
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return response, nil
	}

	period := clampPeriodDays(periodDays)
	now := time.Now().In(time.Local)
	loaded, err := loadVelocityWindowData(
		db,
		selected,
		period,
		selected.OnboardingConfigValue().AppRepository,
		now,
	)
	if err != nil {
		return nil, err
	}

	report, err := buildAdminVelocityReport(db, selected.ID, period, loaded)
	if err != nil {
		return nil, err
	}
	response.AdminVelocityReport = report
	return response, nil
}

func adminVelocityFactories(factories []models.Factory) []AdminVelocityFactory {
	out := make([]AdminVelocityFactory, 0, len(factories))
	for i := range factories {
		out = append(out, AdminVelocityFactory{
			ID:   factories[i].ID.String(),
			Name: factories[i].Name,
		})
	}
	return out
}

func selectAdminVelocityFactory(factories []models.Factory, factoryID string) (*models.Factory, error) {
	if len(factories) == 0 {
		return nil, nil
	}

	requested := strings.TrimSpace(factoryID)
	if requested == "" {
		selected := factories[0]
		return &selected, nil
	}

	id, err := uuid.Parse(requested)
	if err != nil {
		return nil, models.ErrFactoryNotFound
	}
	for i := range factories {
		if factories[i].ID != id {
			continue
		}
		selected := factories[i]
		return &selected, nil
	}
	return nil, models.ErrFactoryNotFound
}

func buildAdminVelocityReport(
	db *gorm.DB,
	factoryID uuid.UUID,
	period int,
	loaded *velocityWindowData,
) (*AdminVelocityReport, error) {
	totals := aggregateTotals(loaded.buckets, loaded.cohort.hasPeople)
	previousTotals := aggregateTotals(loaded.previousBuckets, loaded.cohort.hasPeople)
	rows, err := models.SummarizeFactoryAutomationRuns(db, factoryID, loaded.current.start, loaded.current.end)
	if err != nil {
		return nil, err
	}

	return &AdminVelocityReport{
		FactoryID:         factoryID.String(),
		PeriodDays:        period,
		HasPeopleCohort:   loaded.cohort.hasPeople,
		HasPreviousWindow: hasVelocityOutput(previousTotals),
		Totals:            adminVelocityTotals(totals),
		PreviousTotals:    adminVelocityTotals(previousTotals),
		Points:            adminVelocityPoints(loaded.buckets),
		IntakeSources: adminVelocityIntakeSources(serializeVelocityIntakeSources(
			loaded.currentOrders,
			loaded.cohort.agentMergedIn(loaded.current),
		)),
		Automations: adminVelocityAutomations(rows),
	}, nil
}

func adminVelocityTotals(totals *pb.DescribeFactoryVelocityTotals) AdminVelocityTotals {
	if totals == nil {
		return AdminVelocityTotals{}
	}
	return AdminVelocityTotals{
		SuperplaneMerged: totals.GetSuperplaneMerged(),
		PeopleMerged:     totals.GetPeopleMerged(),
		Waste:            totals.GetWaste(),
		CostCents:        totals.GetCostCents(),
		Tokens:           totals.GetTokens(),
		WasteCostCents:   totals.GetWasteCostCents(),
		TasksClosed:      totals.GetTasksClosed(),
		TasksWaste:       totals.GetTasksWaste(),
		ModelCostCents:   totals.GetModelCostCents(),
		ComputeCostCents: totals.GetComputeCostCents(),
	}
}

func adminVelocityPoints(buckets []dayBucket) []AdminVelocityPoint {
	points := make([]AdminVelocityPoint, 0, len(buckets))
	for i := range buckets {
		bucket := &buckets[i]
		points = append(points, AdminVelocityPoint{
			Day:                        dayLabel(bucket.start),
			SuperplaneMerged:           int32(bucket.superplaneMerged),
			PeopleMerged:               int32(bucket.peopleMerged),
			Waste:                      int32(bucket.waste),
			Intake:                     adminVelocityIntakeCounts(serializeVelocityIntakeCounts(bucket.intake)),
			CostCents:                  bucket.costCents,
			Tokens:                     bucket.tokens,
			WasteCostCents:             bucket.wasteCostCents,
			ModelCostCents:             bucket.modelCostCents,
			ComputeCostCents:           bucket.computeCostCents,
			MedianTaskModelCostCents:   medianCents(bucket.taskModelCostCents),
			MedianTaskComputeCostCents: medianCents(bucket.taskComputeCostCents),
		})
	}
	return points
}

func adminVelocityIntakeCounts(counts []*pb.DescribeFactoryVelocityIntakeCount) []AdminVelocityIntakeCount {
	out := make([]AdminVelocityIntakeCount, 0, len(counts))
	for _, count := range counts {
		if count == nil {
			continue
		}
		out = append(out, AdminVelocityIntakeCount{
			Key:    count.GetKey(),
			Merged: count.GetMerged(),
		})
	}
	return out
}

func adminVelocityIntakeSources(sources []*pb.DescribeFactoryVelocityIntakeSource) []AdminVelocityIntakeSource {
	out := make([]AdminVelocityIntakeSource, 0, len(sources))
	for _, source := range sources {
		if source == nil {
			continue
		}
		out = append(out, AdminVelocityIntakeSource{
			Key:    source.GetKey(),
			Label:  source.GetLabel(),
			Merged: source.GetMerged(),
		})
	}
	return out
}

func adminVelocityAutomations(rows []models.FactoryAutomationRuns) []AdminVelocityAutomation {
	out := make([]AdminVelocityAutomation, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminVelocityAutomation{
			ID:                   row.CanvasID.String(),
			Name:                 row.Name,
			Runs:                 int32(row.Runs),
			Failed:               int32(row.Failed),
			CostCents:            row.CostCents(),
			AverageDurationHours: row.AverageDurationHours(),
		})
	}
	return out
}
