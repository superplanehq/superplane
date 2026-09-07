package factories

import (
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

// Intake keys that are not intake sources. A work order reaches the factory
// without an intake when somebody imported a ticket, opened it by hand, or an
// automation opened it outside an intake canvas.
const (
	velocityIntakeKeyImported   = "imported"
	velocityIntakeKeyManual     = "manual"
	velocityIntakeKeyAutomation = "automation"
)

var velocityIntakeLabels = map[string]string{
	models.FactoryIntakeSourceGitHubIssues:       "GitHub issue",
	models.FactoryIntakeSourceSentryExceptions:   "Sentry exception",
	models.FactoryIntakeSourcePagerDutyIncidents: "PagerDuty incident",
	velocityIntakeKeyImported:                    "Imported ticket",
	velocityIntakeKeyManual:                      "Manually created",
	velocityIntakeKeyAutomation:                  "Automation",
}

// velocityIntakeSeriesOrder fixes the order of the intake bands. A band that
// keeps its position keeps its color when its count changes between requests.
var velocityIntakeSeriesOrder = []string{
	models.FactoryIntakeSourceGitHubIssues,
	models.FactoryIntakeSourceSentryExceptions,
	models.FactoryIntakeSourcePagerDutyIncidents,
	velocityIntakeKeyImported,
	velocityIntakeKeyManual,
	velocityIntakeKeyAutomation,
}

// velocityWindow is one slice of the timeline. Both the reported window and the
// window before it are measured the same way, so their totals compare.
type velocityWindow struct {
	start time.Time
	end   time.Time
}

func (w velocityWindow) contains(t time.Time) bool {
	local := t.In(w.start.Location())
	return !local.Before(w.start) && local.Before(w.end)
}

// velocityOrder is the work order behind the pull requests of one window.
// Costs and cycle time belong to the order, not to each pull request, so an
// order that opened several pull requests is counted once.
type velocityOrder struct {
	id          uuid.UUID
	createdByID *uuid.UUID
	intakeKey   string
	// Local midnight of the day the order is reported on.
	day        time.Time
	merged     bool
	cycleHours *float64
	costCents  int64
	// Bands of costCents: what the order spent on model tokens and on runner
	// compute.
	modelCostCents   int64
	computeCostCents int64
	tokens           int64
}

// collectVelocityOrders attributes every work order with pull request activity
// in the window to one day and one outcome.
//
// An order lands on the day of its earliest merge. An order whose pull requests
// all closed without merging lands on the day of its last close and counts as
// waste.
func collectVelocityOrders(rows []models.FactoryVelocityPullRequest, window velocityWindow) map[uuid.UUID]*velocityOrder {
	orders := make(map[uuid.UUID]*velocityOrder)

	for i := range rows {
		row := &rows[i]

		merged := row.MergedAt != nil && window.contains(*row.MergedAt)
		waste := row.MergedAt == nil && row.ClosedAt != nil && window.contains(*row.ClosedAt)
		if !merged && !waste {
			continue
		}

		at := row.ClosedAt
		if merged {
			at = row.MergedAt
		}
		day := localMidnight(at.In(window.start.Location()))

		order, seen := orders[row.WorkOrderID]
		if !seen {
			orders[row.WorkOrderID] = &velocityOrder{
				id:          row.WorkOrderID,
				createdByID: row.CreatedByID,
				intakeKey:   classifyVelocityIntake(row),
				day:         day,
				merged:      merged,
				cycleHours:  row.CycleHours,
			}
			continue
		}

		switch {
		case merged && !order.merged:
			// The first merge outranks any close: the order produced output.
			order.merged = true
			order.day = day
		case merged && order.merged && day.Before(order.day):
			order.day = day
		case waste && !order.merged && day.After(order.day):
			order.day = day
		}
	}

	return orders
}

func classifyVelocityIntake(row *models.FactoryVelocityPullRequest) string {
	if _, ok := velocityIntakeLabels[row.IntakeSource]; ok {
		return row.IntakeSource
	}
	if origin := strings.TrimSpace(row.OriginURL); origin != "" {
		if strings.Contains(strings.ToLower(origin), "github.com") {
			return models.FactoryIntakeSourceGitHubIssues
		}
		return velocityIntakeKeyImported
	}
	if row.CreatedByID != nil {
		return velocityIntakeKeyManual
	}
	return velocityIntakeKeyAutomation
}

func velocityIntakeLabel(key string) string {
	if label, ok := velocityIntakeLabels[key]; ok {
		return label
	}
	return key
}

// applyVelocityOrderUsage fills tracked spend on each order, in both bands.
//
// The total is the sum of the two rounded bands rather than the rounded sum,
// so a stacked chart of the bands always adds up to the total it is drawn
// against. The two can differ by a cent.
func applyVelocityOrderUsage(orders map[uuid.UUID]*velocityOrder, usage map[uuid.UUID]models.UsageSplit) {
	for id, order := range orders {
		split, ok := usage[id]
		if !ok {
			continue
		}
		order.modelCostCents = split.Model.CostCents()
		order.computeCostCents = split.Compute.CostCents()
		order.costCents = order.modelCostCents + order.computeCostCents
		order.tokens = split.Total().TotalTokens
	}
}

func velocityOrderIDs(orders map[uuid.UUID]*velocityOrder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(orders))
	for id := range orders {
		ids = append(ids, id)
	}
	return ids
}

// velocityIntakeTotals counts merged pull requests per intake source, keeping
// the fixed series order and dropping sources with no merges.
func velocityIntakeTotals(orders map[uuid.UUID]*velocityOrder) []string {
	return velocityIntakeKeysPresent(velocityIntakeMergedCounts(orders, 0))
}

// velocityIntakeMergedCounts counts merged pull requests per intake source.
//
// agentMerged is agent work this instance did not open, which reaches the report
// through the repository sync rather than through a work order. It has no intake
// of its own, so it counts as automation.
func velocityIntakeMergedCounts(orders map[uuid.UUID]*velocityOrder, agentMerged int) map[string]int {
	counts := make(map[string]int, len(orders))
	for _, order := range orders {
		if order.merged {
			counts[order.intakeKey]++
		}
	}
	if agentMerged > 0 {
		counts[velocityIntakeKeyAutomation] += agentMerged
	}
	return counts
}

// velocityIntakeKeysPresent names the series that hold output, in a fixed order.
// A band that keeps its position keeps its color between requests.
func velocityIntakeKeysPresent(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for _, key := range velocityIntakeSeriesOrder {
		if counts[key] > 0 {
			keys = append(keys, key)
		}
	}
	return keys
}

// velocityPersonRow accumulates one row of the People table.
type velocityPersonRow struct {
	id             string
	name           string
	email          string
	avatarURL      string
	authoredMerged int
	factoryMerged  int
	factoryWaste   int
	costCents      int64
	cycleHours     []float64
}

func (r *velocityPersonRow) totalMerged() int {
	return r.authoredMerged + r.factoryMerged
}

// velocityPeopleBuilder joins two identities of the same person: the GitHub
// author of a merged pull request, and the SuperPlane member who opened a work
// order. A member with a connected GitHub account is one row.
type velocityPeopleBuilder struct {
	rows        map[string]*velocityPersonRow
	byUserID    map[uuid.UUID]*models.FactoryVelocityMember
	byGitHubKey map[string]*models.FactoryVelocityMember
}

func newVelocityPeopleBuilder(members []models.FactoryVelocityMember) *velocityPeopleBuilder {
	builder := &velocityPeopleBuilder{
		rows:        make(map[string]*velocityPersonRow),
		byUserID:    make(map[uuid.UUID]*models.FactoryVelocityMember, len(members)),
		byGitHubKey: make(map[string]*models.FactoryVelocityMember, len(members)),
	}

	for i := range members {
		member := &members[i]
		builder.byUserID[member.UserID] = member
		if login := normalizeGitHubLogin(member.GitHubLogin); login != "" {
			builder.byGitHubKey[login] = member
		}
	}
	return builder
}

func normalizeGitHubLogin(login string) string {
	return strings.ToLower(strings.TrimSpace(login))
}

func (b *velocityPeopleBuilder) memberRow(member *models.FactoryVelocityMember) *velocityPersonRow {
	id := member.UserID.String()
	row, ok := b.rows[id]
	if ok {
		return row
	}

	row = &velocityPersonRow{
		id:        id,
		name:      member.Name,
		email:     member.Email,
		avatarURL: member.AvatarURL,
	}
	b.rows[id] = row
	return row
}

// addAuthoredMerge credits a merged pull request to its GitHub author. Authors
// outside the organization still get a row, because they are part of what the
// repository shipped.
func (b *velocityPeopleBuilder) addAuthoredMerge(merge *models.FactoryVelocityRepositoryMerge) {
	login := normalizeGitHubLogin(merge.AuthorLogin)
	if login == "" {
		return
	}

	if member, ok := b.byGitHubKey[login]; ok {
		row := b.memberRow(member)
		if row.avatarURL == "" {
			row.avatarURL = merge.AuthorAvatarURL
		}
		row.authoredMerged++
		return
	}

	id := "github:" + login
	row, ok := b.rows[id]
	if !ok {
		name := strings.TrimSpace(merge.AuthorName)
		if name == "" {
			name = merge.AuthorLogin
		}
		row = &velocityPersonRow{id: id, name: name, avatarURL: merge.AuthorAvatarURL}
		b.rows[id] = row
	}
	row.authoredMerged++
}

// addFactoryOrder credits a work order to the member who opened it. Orders an
// automation opened have no member and stay out of the table.
func (b *velocityPeopleBuilder) addFactoryOrder(order *velocityOrder) {
	if order.createdByID == nil {
		return
	}
	member, ok := b.byUserID[*order.createdByID]
	if !ok {
		return
	}

	row := b.memberRow(member)
	if order.merged {
		row.factoryMerged++
	} else {
		row.factoryWaste++
	}
	row.costCents += order.costCents
	if order.cycleHours != nil {
		row.cycleHours = append(row.cycleHours, *order.cycleHours)
	}
}

// velocityPeopleSortKey names the column the People table is ordered by.
type velocityPeopleSortKey int

const (
	velocitySortTotal velocityPeopleSortKey = iota
	velocitySortFactoryMerged
	velocitySortAuthoredMerged
	velocitySortMedianCycleHours
	velocitySortCostUsd
)

// velocitySortDirection is ascending or descending, applied to the primary
// sort key only; every tie-break stays in its own fixed direction so paging
// is stable regardless of which column or direction is active.
type velocitySortDirection int

const (
	velocitySortDesc velocitySortDirection = iota
	velocitySortAsc
)

// rowsSorted returns people with activity, ordered by key and direction. Ties
// on the primary key break through total merged, then SuperPlane merged, then
// name, then id, in that fixed order, so a page requested with an offset never
// drops or duplicates a row because two requests disagreed on the order of
// equal rows.
func (b *velocityPeopleBuilder) rowsSorted(key velocityPeopleSortKey, direction velocitySortDirection) []*velocityPersonRow {
	rows := make([]*velocityPersonRow, 0, len(b.rows))
	for _, row := range b.rows {
		if row.totalMerged() == 0 && row.factoryWaste == 0 {
			continue
		}
		rows = append(rows, row)
	}

	// Computed once per row, rather than inside the comparator, so an O(n log n)
	// sort does not resort to an O(n log n) sort of median computations.
	medians := make(map[*velocityPersonRow]float64, len(rows))
	for _, row := range rows {
		medians[row] = medianFloats(row.cycleHours)
	}

	primary := func(row *velocityPersonRow) float64 {
		switch key {
		case velocitySortFactoryMerged:
			return float64(row.factoryMerged)
		case velocitySortAuthoredMerged:
			return float64(row.authoredMerged)
		case velocitySortMedianCycleHours:
			return medians[row]
		case velocitySortCostUsd:
			return float64(row.costCents)
		default:
			return float64(row.totalMerged())
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		if lv, rv := primary(left), primary(right); lv != rv {
			if direction == velocitySortAsc {
				return lv < rv
			}
			return lv > rv
		}
		if left.totalMerged() != right.totalMerged() {
			return left.totalMerged() > right.totalMerged()
		}
		if left.factoryMerged != right.factoryMerged {
			return left.factoryMerged > right.factoryMerged
		}
		if left.name != right.name {
			return left.name < right.name
		}
		return left.id < right.id
	})
	return rows
}

func medianFloats(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// medianCents returns the middle value of a cent sample. An even sample
// averages the two middle values and truncates, because a cent is the smallest
// amount the report shows.
func medianCents(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func localMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
