package factories

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// FactoryVelocityWindowTotals is the share math a public badge and the
// Velocity report both use.
type FactoryVelocityWindowTotals struct {
	SuperplaneMerged int
	PeopleMerged     int
	Waste            int
	CostCents        int64
}

// FactoryVelocityDayPoint is one day of the "who created" series.
type FactoryVelocityDayPoint struct {
	Date             time.Time
	SuperplaneMerged int
	PeopleMerged     int
}

// FactoryVelocityWindows is the current window, the previous window of the
// same length, and the daily merged counts. It does not include people,
// automations, or intake.
type FactoryVelocityWindows struct {
	Current     FactoryVelocityWindowTotals
	Previous    FactoryVelocityWindowTotals
	HasPrevious bool
	Days        []FactoryVelocityDayPoint
}

type velocityWindowData struct {
	buckets         []dayBucket
	previousBuckets []dayBucket
	current         velocityWindow
	currentOrders   map[uuid.UUID]*velocityOrder
	cohort          mergeCohort
	repoOwner       string
	repoName        string
}

// ComputeFactoryVelocityWindows loads the same windows DescribeFactoryVelocity
// uses for totals, the previous window, and the daily merged counts.
// now should already be in the location used for calendar days.
func ComputeFactoryVelocityWindows(
	db *gorm.DB,
	factory *models.Factory,
	periodDays int,
	repository string,
	now time.Time,
) (FactoryVelocityWindows, error) {
	loaded, err := loadVelocityWindowData(db, factory, periodDays, repository, now)
	if err != nil {
		return FactoryVelocityWindows{}, err
	}
	return summarizeVelocityWindows(loaded), nil
}

func loadVelocityWindowData(
	db *gorm.DB,
	factory *models.Factory,
	periodDays int,
	repository string,
	now time.Time,
) (*velocityWindowData, error) {
	period := clampPeriodDays(periodDays)
	buckets := buildDayBuckets(now, period)
	current := velocityWindow{start: buckets[0].start, end: buckets[len(buckets)-1].end}
	previous := velocityWindow{start: current.start.AddDate(0, 0, -period), end: current.start}
	previousBuckets := buildWindowBuckets(previous, period)

	repoOwner, repoName, hasRepo := parseOwnerRepo(repository)
	rows, err := models.ListFactoryVelocityPullRequests(db, factory.ID, previous.start, current.end)
	if err != nil {
		return nil, err
	}
	rows = filterVelocityRowsByRepository(rows, repoOwner, repoName, hasRepo)

	currentOrders := collectVelocityOrders(rows, current)
	previousOrders := collectVelocityOrders(rows, previous)
	usage, err := models.SumUsageForWorkOrdersByKind(db, append(velocityOrderIDs(currentOrders), velocityOrderIDs(previousOrders)...))
	if err != nil {
		return nil, err
	}
	applyVelocityOrderUsage(currentOrders, usage)
	applyVelocityOrderUsage(previousOrders, usage)

	cohort, err := loadMergeCohort(db, factory.ID, hasRepo, previous.start, current.end)
	if err != nil {
		return nil, err
	}
	fillBuckets(buckets, rows, currentOrders, cohort.merges, current)
	fillBuckets(previousBuckets, rows, previousOrders, cohort.merges, previous)

	return &velocityWindowData{
		buckets:         buckets,
		previousBuckets: previousBuckets,
		current:         current,
		currentOrders:   currentOrders,
		cohort:          cohort,
		repoOwner:       repoOwner,
		repoName:        repoName,
	}, nil
}

func summarizeVelocityWindows(data *velocityWindowData) FactoryVelocityWindows {
	current := aggregateTotals(data.buckets, data.cohort.hasPeople)
	previous := aggregateTotals(data.previousBuckets, data.cohort.hasPeople)
	days := make([]FactoryVelocityDayPoint, len(data.buckets))
	for i := range data.buckets {
		people := data.buckets[i].peopleMerged
		if !data.cohort.hasPeople {
			people = 0
		}
		days[i] = FactoryVelocityDayPoint{
			Date:             data.buckets[i].start,
			SuperplaneMerged: data.buckets[i].superplaneMerged,
			PeopleMerged:     people,
		}
	}
	return FactoryVelocityWindows{
		Current:     windowTotalsFromProto(current),
		Previous:    windowTotalsFromProto(previous),
		HasPrevious: hasVelocityOutput(previous),
		Days:        days,
	}
}

func windowTotalsFromProto(totals *pb.DescribeFactoryVelocityTotals) FactoryVelocityWindowTotals {
	if totals == nil {
		return FactoryVelocityWindowTotals{}
	}
	return FactoryVelocityWindowTotals{
		SuperplaneMerged: int(totals.GetSuperplaneMerged()),
		PeopleMerged:     int(totals.GetPeopleMerged()),
		Waste:            int(totals.GetWaste()),
		CostCents:        totals.GetCostCents(),
	}
}
