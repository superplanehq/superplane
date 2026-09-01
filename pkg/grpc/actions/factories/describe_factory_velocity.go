package factories

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

const (
	velocityPeriodDaysDefault = 14
	velocityPeriodDaysMax     = 30
)

// DescribeFactoryVelocity reports what a workspace shipped, how long the work
// took, and what it cost.
//
// Every number comes from the database. Repository merges by people are stored
// by the factory velocity sync worker, so this handler makes no external calls
// and cannot lose the People series to a third-party failure.
func DescribeFactoryVelocity(
	ctx context.Context,
	organizationID string,
	req *pb.DescribeFactoryVelocityRequest,
) (*pb.DescribeFactoryVelocityResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	period := clampPeriodDays(int(req.GetPeriodDays()))

	db := database.DB(ctx)
	if _, err := models.FindFactory(db, orgID, factoryID); err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	now := time.Now().In(time.Local)
	buckets := buildDayBuckets(now, period)
	current := velocityWindow{start: buckets[0].start, end: buckets[len(buckets)-1].end}
	previous := velocityWindow{start: current.start.AddDate(0, 0, -period), end: current.start}
	previousBuckets := buildWindowBuckets(previous, period)

	repoOwner, repoName, hasRepo := parseOwnerRepo(req.GetRepository())

	// One query covers both windows, so the comparison costs no extra round trip.
	rows, err := models.ListFactoryVelocityPullRequests(db, factoryID, previous.start, current.end)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}
	rows = filterVelocityRowsByRepository(rows, repoOwner, repoName, hasRepo)

	currentOrders := collectVelocityOrders(rows, current)
	previousOrders := collectVelocityOrders(rows, previous)

	usage, err := models.SumUsageForWorkOrders(db, append(velocityOrderIDs(currentOrders), velocityOrderIDs(previousOrders)...))
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}
	applyVelocityOrderUsage(currentOrders, usage)
	applyVelocityOrderUsage(previousOrders, usage)

	cohort, err := loadMergeCohort(db, factoryID, hasRepo, previous.start, current.end)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	fillBuckets(buckets, rows, currentOrders, cohort.merges, current)
	fillBuckets(previousBuckets, rows, previousOrders, cohort.merges, previous)

	yesterdayIdx := len(buckets) - 2
	if yesterdayIdx < 0 {
		yesterdayIdx = 0
	}
	yesterday := buckets[yesterdayIdx]

	totals := aggregateTotals(buckets, cohort.hasPeople)
	previousTotals := aggregateTotals(previousBuckets, cohort.hasPeople)

	people, err := buildVelocityPeople(db, orgID, currentOrders, cohort.merges, current)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to describe factory velocity")
	}

	points := make([]*pb.DescribeFactoryVelocityDay, 0, len(buckets))
	for i := range buckets {
		b := &buckets[i]
		points = append(points, &pb.DescribeFactoryVelocityDay{
			Day:              dayLabel(b.start),
			Date:             timestamppb.New(b.start),
			SuperplaneMerged: int32(b.superplaneMerged),
			PeopleMerged:     int32(b.peopleMerged),
			Waste:            int32(b.waste),
			Intake:           serializeVelocityIntakeCounts(b.intake),
			CostCents:        b.costCents,
			Tokens:           b.tokens,
			WasteCostCents:   b.wasteCostCents,
		})
	}

	return &pb.DescribeFactoryVelocityResponse{
		Yesterday: &pb.DescribeFactoryVelocityYesterday{
			Date:             timestamppb.New(calendarDayUTCNoon(yesterday.start)),
			SuperplaneMerged: int32(yesterday.superplaneMerged),
			Waste:            int32(yesterday.waste),
		},
		Totals:            totals,
		Points:            points,
		Repository:        joinOwnerRepo(repoOwner, repoName),
		HasPeopleCohort:   cohort.hasPeople,
		PeopleSyncedAt:    cohort.syncedAt(),
		PeopleSyncPending: cohort.pending,
		PreviousTotals:    previousTotals,
		HasPreviousWindow: hasVelocityOutput(previousTotals),
		IntakeSources:     serializeVelocityIntakeSources(currentOrders, cohort.agentMergedIn(current)),
		People:            people,
	}, nil
}

// mergeCohort is the stored repository history both merge series are built
// from, together with how fresh it is.
type mergeCohort struct {
	merges []models.FactoryVelocityRepositoryMerge
	// hasPeople reports whether repository merges are stored, so the People
	// series and the SuperPlane share are meaningful.
	hasPeople bool
	// pending reports a repository whose first sync has not stored merges yet.
	// The UI explains the gap instead of claiming people merged nothing.
	pending  bool
	syncedOn *time.Time
}

func (c mergeCohort) syncedAt() *timestamppb.Timestamp {
	if c.syncedOn == nil {
		return nil
	}
	return timestamppb.New(*c.syncedOn)
}

// agentMergedIn counts the agent merges of a window. They carry no work order,
// so the intake breakdown has to learn about them from here.
func (c mergeCohort) agentMergedIn(window velocityWindow) int {
	merged := 0
	for i := range c.merges {
		if c.merges[i].IsAgent() && window.contains(c.merges[i].MergedAt) {
			merged++
		}
	}
	return merged
}

// loadMergeCohort reads the repository merges the sync worker stored. A
// workspace with no repository, or one whose first sync has not finished,
// reports no cohort and the SuperPlane counts still return.
func loadMergeCohort(
	tx *gorm.DB,
	factoryID uuid.UUID,
	hasRepo bool,
	from, to time.Time,
) (mergeCohort, error) {
	if !hasRepo {
		return mergeCohort{}, nil
	}

	sync, err := models.FindFactoryVelocitySync(tx, factoryID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return mergeCohort{}, err
	}
	if sync == nil || sync.SyncedAt == nil {
		return mergeCohort{pending: true}, nil
	}

	merges, err := models.ListFactoryVelocityRepositoryMerges(tx, factoryID, from, to)
	if err != nil {
		return mergeCohort{}, err
	}

	return mergeCohort{
		merges:    merges,
		hasPeople: true,
		syncedOn:  sync.SyncedAt,
	}, nil
}

// hasVelocityOutput reports whether a window holds enough output to compare
// against. A workspace younger than two windows has an empty baseline, and a
// delta against nothing reads as infinite growth.
func hasVelocityOutput(totals *pb.DescribeFactoryVelocityTotals) bool {
	if totals == nil {
		return false
	}
	return totals.GetSuperplaneMerged()+totals.GetPeopleMerged()+totals.GetWaste() > 0
}

func serializeVelocityIntakeCounts(counts map[string]int) []*pb.DescribeFactoryVelocityIntakeCount {
	if len(counts) == 0 {
		return nil
	}

	out := make([]*pb.DescribeFactoryVelocityIntakeCount, 0, len(counts))
	for _, key := range velocityIntakeSeriesOrder {
		merged, ok := counts[key]
		if !ok || merged == 0 {
			continue
		}
		out = append(out, &pb.DescribeFactoryVelocityIntakeCount{Key: key, Merged: int32(merged)})
	}
	return out
}

func serializeVelocityIntakeSources(
	orders map[uuid.UUID]*velocityOrder,
	agentMerged int,
) []*pb.DescribeFactoryVelocityIntakeSource {
	totals := velocityIntakeMergedCounts(orders, agentMerged)

	keys := velocityIntakeKeysPresent(totals)
	out := make([]*pb.DescribeFactoryVelocityIntakeSource, 0, len(keys))
	for _, key := range keys {
		out = append(out, &pb.DescribeFactoryVelocityIntakeSource{
			Key:    key,
			Label:  velocityIntakeLabel(key),
			Merged: int32(totals[key]),
		})
	}
	return out
}

// buildVelocityPeople joins repository authorship with the work orders each
// member opened. It reports no rows when the organization has no members with
// activity, which is what a brand new workspace looks like.
func buildVelocityPeople(
	tx *gorm.DB,
	orgID uuid.UUID,
	orders map[uuid.UUID]*velocityOrder,
	merges []models.FactoryVelocityRepositoryMerge,
	window velocityWindow,
) ([]*pb.DescribeFactoryVelocityPerson, error) {
	members, err := models.ListFactoryVelocityMembers(tx, orgID)
	if err != nil {
		return nil, err
	}

	builder := newVelocityPeopleBuilder(members)
	for i := range merges {
		// An agent merge belongs to SuperPlane, not to the account that opened
		// it, which is the GitHub App rather than a person.
		if merges[i].IsAgent() || !window.contains(merges[i].MergedAt) {
			continue
		}
		builder.addAuthoredMerge(&merges[i])
	}
	for _, order := range orders {
		builder.addFactoryOrder(order)
	}

	rows := builder.rowsByMergedDesc()
	out := make([]*pb.DescribeFactoryVelocityPerson, 0, len(rows))
	for _, row := range rows {
		out = append(out, &pb.DescribeFactoryVelocityPerson{
			Id:               row.id,
			Name:             row.name,
			Email:            row.email,
			AvatarUrl:        row.avatarURL,
			AuthoredMerged:   int32(row.authoredMerged),
			FactoryMerged:    int32(row.factoryMerged),
			FactoryWaste:     int32(row.factoryWaste),
			CostCents:        row.costCents,
			MedianCycleHours: medianFloats(row.cycleHours),
		})
	}
	return out, nil
}

func clampPeriodDays(v int) int {
	if v <= 0 {
		return velocityPeriodDaysDefault
	}
	if v > velocityPeriodDaysMax {
		return velocityPeriodDaysMax
	}
	return v
}

type dayBucket struct {
	start            time.Time
	end              time.Time
	superplaneMerged int
	peopleMerged     int
	waste            int
	// Merged SuperPlane pull requests of the day, keyed by intake source.
	intake         map[string]int
	costCents      int64
	tokens         int64
	wasteCostCents int64
}

// dayLabel names the axis tick of a day, as a weekday and a date.
//
// The weekday carries most of the meaning: it explains the gaps in the chart,
// because a quiet Saturday reads as a weekend rather than as an outage. The date
// tells the reader which day it was, which a "day 9 of 14" number cannot.
//
// The month appears only where the window crosses into a new one, compared
// against the day immediately before it. Naming it on every tick would make
// no tick stand out; naming it nowhere would leave a reader unsure which
// month a mid-window tick belongs to.
//
// Every day gets a full label. How many of them a chart has room to draw is
// the chart's decision (see pickVelocityAxisTicks on the frontend), not this
// producer's: it only names days.
func dayLabel(start time.Time) string {
	if start.Month() != start.AddDate(0, 0, -1).Month() {
		return start.Format("Mon Jan 2")
	}
	return start.Format("Mon 2")
}

func buildDayBuckets(now time.Time, periodDays int) []dayBucket {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	buckets := make([]dayBucket, periodDays)
	for i := 0; i < periodDays; i++ {
		start := today.AddDate(0, 0, -(periodDays - 1 - i))
		buckets[i] = newDayBucket(start)
	}
	return buckets
}

func buildWindowBuckets(window velocityWindow, periodDays int) []dayBucket {
	buckets := make([]dayBucket, periodDays)
	for i := 0; i < periodDays; i++ {
		buckets[i] = newDayBucket(window.start.AddDate(0, 0, i))
	}
	return buckets
}

func newDayBucket(start time.Time) dayBucket {
	return dayBucket{
		start:  start,
		end:    start.AddDate(0, 0, 1),
		intake: map[string]int{},
	}
}

// calendarDayUTCNoon returns 12:00 UTC on the civil date of t in t's location.
// The UI formats this instant in UTC so the label matches the server calendar
// day, independent of the browser timezone.
func calendarDayUTCNoon(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

// filterVelocityRowsByRepository keeps only pull requests of the repository the
// workspace reports on. Without a selected repository every factory pull
// request counts.
func filterVelocityRowsByRepository(
	rows []models.FactoryVelocityPullRequest,
	repoOwner, repoName string,
	hasRepo bool,
) []models.FactoryVelocityPullRequest {
	if !hasRepo {
		return rows
	}

	out := make([]models.FactoryVelocityPullRequest, 0, len(rows))
	for i := range rows {
		owner, repo, ok := parseOwnerRepo(rows[i].Repository)
		if !ok {
			continue
		}
		if owner == repoOwner && repo == repoName {
			out = append(out, rows[i])
		}
	}
	return out
}

func parseOwnerRepo(repository string) (owner, repo string, ok bool) {
	repository = strings.TrimSpace(repository)
	if repository == "" {
		return "", "", false
	}
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return strings.ToLower(parts[0]), strings.ToLower(parts[1]), true
}

func joinOwnerRepo(owner, repo string) string {
	if owner == "" || repo == "" {
		return ""
	}
	return owner + "/" + repo
}

// fillBuckets counts a window into its day buckets. Pull request counts come
// from the pull requests themselves, while cost comes from the work orders
// behind them, so an order that opened several pull requests is charged once.
func fillBuckets(
	buckets []dayBucket,
	rows []models.FactoryVelocityPullRequest,
	orders map[uuid.UUID]*velocityOrder,
	merges []models.FactoryVelocityRepositoryMerge,
	window velocityWindow,
) {
	starts := make([]time.Time, len(buckets))
	for i := range buckets {
		starts[i] = buckets[i].start
	}
	locate := func(t time.Time) int {
		if len(buckets) == 0 {
			return -1
		}
		local := t.In(buckets[0].start.Location())
		idx := sort.Search(len(starts), func(i int) bool { return starts[i].After(local) }) - 1
		if idx < 0 || idx >= len(buckets) {
			return -1
		}
		if local.Before(buckets[idx].start) || !local.Before(buckets[idx].end) {
			return -1
		}
		return idx
	}

	for i := range rows {
		row := &rows[i]
		if row.MergedAt != nil {
			if !window.contains(*row.MergedAt) {
				continue
			}
			if idx := locate(*row.MergedAt); idx >= 0 {
				buckets[idx].superplaneMerged++
				buckets[idx].intake[classifyVelocityIntake(row)]++
			}
			continue
		}
		if row.ClosedAt == nil || !window.contains(*row.ClosedAt) {
			continue
		}
		if idx := locate(*row.ClosedAt); idx >= 0 {
			buckets[idx].waste++
		}
	}

	for _, order := range orders {
		idx := locate(order.day)
		if idx < 0 {
			continue
		}
		buckets[idx].costCents += order.costCents
		buckets[idx].tokens += order.tokens
		if !order.merged {
			buckets[idx].wasteCostCents += order.costCents
		}
	}

	for i := range merges {
		if !window.contains(merges[i].MergedAt) {
			continue
		}
		idx := locate(merges[i].MergedAt)
		if idx < 0 {
			continue
		}
		if !merges[i].IsAgent() {
			buckets[idx].peopleMerged++
			continue
		}
		// Agent work this instance did not open still is SuperPlane output. It
		// has no work order here, so it counts as automation intake.
		buckets[idx].superplaneMerged++
		buckets[idx].intake[velocityIntakeKeyAutomation]++
	}
}

func aggregateTotals(buckets []dayBucket, hasPeople bool) *pb.DescribeFactoryVelocityTotals {
	sp, people, waste := 0, 0, 0
	var costCents, tokens, wasteCostCents int64
	for _, b := range buckets {
		sp += b.superplaneMerged
		if hasPeople {
			people += b.peopleMerged
		}
		waste += b.waste
		costCents += b.costCents
		tokens += b.tokens
		wasteCostCents += b.wasteCostCents
	}

	totals := &pb.DescribeFactoryVelocityTotals{
		SuperplaneMerged: int32(sp),
		PeopleMerged:     int32(people),
		Waste:            int32(waste),
		CostCents:        costCents,
		Tokens:           tokens,
		WasteCostCents:   wasteCostCents,
	}
	totalMerged := sp + people
	if totalMerged > 0 && hasPeople {
		totals.SuperplaneSharePct = int32((sp * 100) / totalMerged)
	}
	spClosures := sp + waste
	if spClosures > 0 {
		totals.WastePct = int32((waste * 100) / spClosures)
	}
	return totals
}
