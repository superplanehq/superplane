package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func testWindow(t *testing.T) velocityWindow {
	t.Helper()
	start := localMidnight(time.Now().AddDate(0, 0, -6))
	return velocityWindow{start: start, end: start.AddDate(0, 0, 7)}
}

func TestClassifyVelocityIntake(t *testing.T) {
	member := uuid.New()

	cases := []struct {
		name     string
		row      models.FactoryVelocityPullRequest
		expected string
	}{
		{
			name:     "intake source wins",
			row:      models.FactoryVelocityPullRequest{IntakeSource: models.FactoryIntakeSourceSentryExceptions},
			expected: models.FactoryIntakeSourceSentryExceptions,
		},
		{
			name:     "imported GitHub ticket counts as a GitHub issue",
			row:      models.FactoryVelocityPullRequest{OriginURL: "https://github.com/acme/pay/issues/12"},
			expected: models.FactoryIntakeSourceGitHubIssues,
		},
		{
			name:     "other imported ticket",
			row:      models.FactoryVelocityPullRequest{OriginURL: "https://tickets.example.com/issues/7"},
			expected: velocityIntakeKeyImported,
		},
		{
			name:     "member without an origin created it by hand",
			row:      models.FactoryVelocityPullRequest{CreatedByID: &member},
			expected: velocityIntakeKeyManual,
		},
		{
			name:     "no member and no origin means an automation opened it",
			row:      models.FactoryVelocityPullRequest{},
			expected: velocityIntakeKeyAutomation,
		},
		{
			name:     "unknown intake source falls back to the creator",
			row:      models.FactoryVelocityPullRequest{IntakeSource: "brand-new-source", CreatedByID: &member},
			expected: velocityIntakeKeyManual,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := tc.row
			assert.Equal(t, tc.expected, classifyVelocityIntake(&row))
		})
	}
}

func TestCollectVelocityOrders_MergeOutranksWaste(t *testing.T) {
	window := testWindow(t)
	orderID := uuid.New()
	closedAt := window.start.Add(24 * time.Hour)
	mergedAt := window.start.Add(48 * time.Hour)

	orders := collectVelocityOrders([]models.FactoryVelocityPullRequest{
		{WorkOrderID: orderID, ClosedAt: &closedAt},
		{WorkOrderID: orderID, MergedAt: &mergedAt},
	}, window)

	require.Len(t, orders, 1, "one work order produces one row")
	order := orders[orderID]
	assert.True(t, order.merged, "an order that merged anything is output, not waste")
	assert.Equal(t, localMidnight(mergedAt), order.day)
}

func TestCollectVelocityOrders_ReportsEarliestMerge(t *testing.T) {
	window := testWindow(t)
	orderID := uuid.New()
	first := window.start.Add(24 * time.Hour)
	second := window.start.Add(72 * time.Hour)

	orders := collectVelocityOrders([]models.FactoryVelocityPullRequest{
		{WorkOrderID: orderID, MergedAt: &second},
		{WorkOrderID: orderID, MergedAt: &first},
	}, window)

	require.Len(t, orders, 1)
	assert.Equal(t, localMidnight(first), orders[orderID].day)
}

func TestCollectVelocityOrders_SkipsWorkOutsideWindow(t *testing.T) {
	window := testWindow(t)
	before := window.start.Add(-time.Hour)
	after := window.end.Add(time.Hour)

	orders := collectVelocityOrders([]models.FactoryVelocityPullRequest{
		{WorkOrderID: uuid.New(), MergedAt: &before},
		{WorkOrderID: uuid.New(), MergedAt: &after},
		{WorkOrderID: uuid.New()},
	}, window)

	assert.Empty(t, orders)
}

func TestApplyVelocityOrderUsage(t *testing.T) {
	orderID := uuid.New()
	orders := map[uuid.UUID]*velocityOrder{orderID: {id: orderID}}

	applyVelocityOrderUsage(orders, map[uuid.UUID]models.UsageTotals{
		orderID: {TotalTokens: 4200, CostMicros: 2_500_000},
	})

	assert.Equal(t, int64(4200), orders[orderID].tokens)
	assert.Equal(t, int64(250), orders[orderID].costCents, "2.5 million micros is 250 cents")
}

func TestVelocityIntakeTotals_KeepsSeriesOrderAndDropsWaste(t *testing.T) {
	orders := map[uuid.UUID]*velocityOrder{
		uuid.New(): {intakeKey: velocityIntakeKeyManual, merged: true},
		uuid.New(): {intakeKey: models.FactoryIntakeSourceGitHubIssues, merged: true},
		uuid.New(): {intakeKey: models.FactoryIntakeSourceSentryExceptions, merged: false},
	}

	assert.Equal(t, []string{
		models.FactoryIntakeSourceGitHubIssues,
		velocityIntakeKeyManual,
	}, velocityIntakeTotals(orders), "only merged work names a series, in a fixed order")
}

func TestVelocityPeopleBuilder_JoinsGitHubAuthorWithMember(t *testing.T) {
	userID := uuid.New()
	builder := newVelocityPeopleBuilder([]models.FactoryVelocityMember{
		{UserID: userID, Name: "Ada Lovelace", Email: "ada@example.com", GitHubLogin: "AdaLovelace"},
	})

	builder.addAuthoredMerge(&models.FactoryVelocityRepositoryMerge{
		AuthorLogin:     "adalovelace",
		AuthorAvatarURL: "https://avatars/ada",
	})
	builder.addFactoryOrder(&velocityOrder{createdByID: &userID, merged: true, costCents: 120})

	rows := builder.rowsSorted(velocitySortTotal, velocitySortDesc)
	require.Len(t, rows, 1, "one person is one row, whichever identity the work came from")
	assert.Equal(t, userID.String(), rows[0].id)
	assert.Equal(t, "Ada Lovelace", rows[0].name)
	assert.Equal(t, 1, rows[0].authoredMerged)
	assert.Equal(t, 1, rows[0].factoryMerged)
	assert.Equal(t, int64(120), rows[0].costCents)
	assert.Equal(t, "https://avatars/ada", rows[0].avatarURL, "a member without a photo borrows the GitHub one")
}

func TestVelocityPeopleBuilder_KeepsAuthorsOutsideTheOrganization(t *testing.T) {
	builder := newVelocityPeopleBuilder(nil)

	builder.addAuthoredMerge(&models.FactoryVelocityRepositoryMerge{AuthorLogin: "outside-contributor"})

	rows := builder.rowsSorted(velocitySortTotal, velocitySortDesc)
	require.Len(t, rows, 1)
	assert.Equal(t, "github:outside-contributor", rows[0].id)
	assert.Equal(t, "outside-contributor", rows[0].name, "the login stands in for a missing name")
}

func TestVelocityPeopleBuilder_SkipsAutomationOrders(t *testing.T) {
	builder := newVelocityPeopleBuilder(nil)

	builder.addFactoryOrder(&velocityOrder{merged: true})
	builder.addFactoryOrder(&velocityOrder{createdByID: func() *uuid.UUID { id := uuid.New(); return &id }(), merged: true})

	assert.Empty(t, builder.rowsSorted(velocitySortTotal, velocitySortDesc), "the table lists people, not automations or former members")
}

func TestVelocityPeopleBuilder_ReportsWasteOnlyContributors(t *testing.T) {
	userID := uuid.New()
	builder := newVelocityPeopleBuilder([]models.FactoryVelocityMember{{UserID: userID, Name: "Grace"}})

	builder.addFactoryOrder(&velocityOrder{createdByID: &userID, merged: false, costCents: 90})

	rows := builder.rowsSorted(velocitySortTotal, velocitySortDesc)
	require.Len(t, rows, 1, "spend without a merge still belongs to somebody")
	assert.Equal(t, 1, rows[0].factoryWaste)
}

func TestVelocityPeopleBuilder_OrdersByMergedThenName(t *testing.T) {
	first, second, third := uuid.New(), uuid.New(), uuid.New()
	builder := newVelocityPeopleBuilder([]models.FactoryVelocityMember{
		{UserID: first, Name: "Zoe"},
		{UserID: second, Name: "Alan"},
		{UserID: third, Name: "Barbara"},
	})

	builder.addFactoryOrder(&velocityOrder{createdByID: &first, merged: true})
	builder.addFactoryOrder(&velocityOrder{createdByID: &second, merged: true})
	builder.addFactoryOrder(&velocityOrder{createdByID: &third, merged: true})
	builder.addFactoryOrder(&velocityOrder{createdByID: &third, merged: true})

	rows := builder.rowsSorted(velocitySortTotal, velocitySortDesc)
	require.Len(t, rows, 3)
	assert.Equal(t, "Barbara", rows[0].name, "most merges first")
	assert.Equal(t, "Alan", rows[1].name, "ties break on name")
	assert.Equal(t, "Zoe", rows[2].name)
}

// TestVelocityPeopleBuilder_RowsSorted_EveryKeyAndDirection covers the sort
// keys the People table can request, in both directions, including a tie on
// the primary key so the name/id tie-break proves stable paging.
func TestVelocityPeopleBuilder_RowsSorted_EveryKeyAndDirection(t *testing.T) {
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()

	// Alice: 1 factory merge + 1 authored merge (total 2), $1 cost, no cycle time.
	// Bob: 3 factory merges (total 3), $3 cost, median cycle 20h.
	// Carol: 3 factory merges (total 3, ties Bob), $9 cost, median cycle 5h.
	build := func() *velocityPeopleBuilder {
		builder := newVelocityPeopleBuilder([]models.FactoryVelocityMember{
			{UserID: alice, Name: "Alice", GitHubLogin: "alice-gh"},
			{UserID: bob, Name: "Bob"},
			{UserID: carol, Name: "Carol"},
		})
		builder.addFactoryOrder(&velocityOrder{createdByID: &alice, merged: true, costCents: 100})
		builder.addAuthoredMerge(&models.FactoryVelocityRepositoryMerge{AuthorLogin: "alice-gh"})
		for i := 0; i < 3; i++ {
			cycle := 20.0
			builder.addFactoryOrder(&velocityOrder{createdByID: &bob, merged: true, costCents: 100, cycleHours: &cycle})
		}
		for i := 0; i < 3; i++ {
			cycle := 5.0
			builder.addFactoryOrder(&velocityOrder{createdByID: &carol, merged: true, costCents: 300, cycleHours: &cycle})
		}
		return builder
	}

	t.Run("total desc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortTotal, velocitySortDesc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Bob", rows[0].name, "bob and carol tie on total; name breaks the tie")
		assert.Equal(t, "Carol", rows[1].name)
		assert.Equal(t, "Alice", rows[2].name, "alice has the lowest total")
	})

	t.Run("total asc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortTotal, velocitySortAsc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0].name, "alice has the lowest total")
		assert.Equal(t, "Bob", rows[1].name, "the tie-break stays name-ascending even in ASC order")
		assert.Equal(t, "Carol", rows[2].name)
	})

	t.Run("factoryMerged desc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortFactoryMerged, velocitySortDesc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Bob", rows[0].name, "bob and carol tie on factory merges; name breaks the tie")
		assert.Equal(t, "Carol", rows[1].name)
		assert.Equal(t, "Alice", rows[2].name)
	})

	t.Run("authoredMerged desc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortAuthoredMerged, velocitySortDesc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0].name, "only alice authored a merge outside SuperPlane")
	})

	t.Run("medianCycleHours asc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortMedianCycleHours, velocitySortAsc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0].name, "no cycle time sorts as zero")
		assert.Equal(t, "Carol", rows[1].name, "carol's median cycle is shorter than bob's")
		assert.Equal(t, "Bob", rows[2].name)
	})

	t.Run("medianCycleHours desc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortMedianCycleHours, velocitySortDesc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Bob", rows[0].name)
		assert.Equal(t, "Carol", rows[1].name)
		assert.Equal(t, "Alice", rows[2].name)
	})

	t.Run("costUsd desc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortCostUsd, velocitySortDesc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Carol", rows[0].name, "carol spent the most")
		assert.Equal(t, "Bob", rows[1].name)
		assert.Equal(t, "Alice", rows[2].name, "alice spent the least")
	})

	t.Run("costUsd asc", func(t *testing.T) {
		rows := build().rowsSorted(velocitySortCostUsd, velocitySortAsc)
		require.Len(t, rows, 3)
		assert.Equal(t, "Alice", rows[0].name, "alice spent the least")
		assert.Equal(t, "Bob", rows[1].name)
		assert.Equal(t, "Carol", rows[2].name, "carol spent the most, so she is last")
	})
}

func TestVelocityPeopleBuilder_MedianCycleOfMemberOrders(t *testing.T) {
	userID := uuid.New()
	builder := newVelocityPeopleBuilder([]models.FactoryVelocityMember{{UserID: userID, Name: "Grace"}})

	for _, hours := range []float64{4, 10, 30} {
		cycle := hours
		builder.addFactoryOrder(&velocityOrder{createdByID: &userID, merged: true, cycleHours: &cycle})
	}

	rows := builder.rowsSorted(velocitySortTotal, velocitySortDesc)
	require.Len(t, rows, 1)
	assert.Equal(t, float64(10), medianFloats(rows[0].cycleHours))
}

func TestMedianFloats(t *testing.T) {
	assert.Equal(t, float64(0), medianFloats(nil), "no sample has no median")
	assert.Equal(t, float64(7), medianFloats([]float64{7}))
	assert.Equal(t, float64(3), medianFloats([]float64{4, 2}))
	assert.Equal(t, float64(5), medianFloats([]float64{9, 1, 5}))
}
