package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestFactory_ListWorkOrders_UserID(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, callerID, factoryModel := setupFactoryWithUser(t, "user-filter")
	otherUser := createOrgUser(t, org.ID, "user-filter-other")
	unassignedTrue := true
	db := database.Conn()

	assigneeOnly, err := factoryModel.CreateWorkOrder(db, "Assignee only", "", &otherUser.ID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	creatorOnly, err := factoryModel.CreateWorkOrder(db, "Creator only", "", &callerID, nil, nil)
	require.NoError(t, err)

	both, err := factoryModel.CreateWorkOrder(db, "Creator and assignee", "", &callerID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	_, err = factoryModel.CreateWorkOrder(db, "Other user only", "", &otherUser.ID, []uuid.UUID{otherUser.ID}, nil)
	require.NoError(t, err)

	otherUnassigned, err := factoryModel.CreateWorkOrder(db, "Other unassigned", "", &otherUser.ID, nil, nil)
	require.NoError(t, err)

	orders, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{UserID: &callerID, Limit: 10})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{assigneeOnly.ID, creatorOnly.ID, both.ID}, workOrderIDs(orders))

	nobody, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Unassigned: &unassignedTrue,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{creatorOnly.ID, otherUnassigned.ID}, workOrderIDs(nobody))

	userOrNobody, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		UserID:     &callerID,
		Unassigned: &unassignedTrue,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{assigneeOnly.ID, creatorOnly.ID, both.ID, otherUnassigned.ID}, workOrderIDs(userOrNobody))
}

func workOrderIDs(orders []FactoryWorkOrder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	return ids
}

func TestFactory_ListWorkOrders_PagesByUpdatedAt(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "page-orders")
	db := database.Conn()

	first, err := factoryModel.CreateWorkOrder(db, "Oldest update", "", &callerID, nil, nil)
	require.NoError(t, err)
	second, err := factoryModel.CreateWorkOrder(db, "Middle update", "", &callerID, nil, nil)
	require.NoError(t, err)
	third, err := factoryModel.CreateWorkOrder(db, "Newest update", "", &callerID, nil, nil)
	require.NoError(t, err)

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", first.ID).UpdateColumn("updated_at", base).Error)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", second.ID).UpdateColumn("updated_at", base.Add(time.Hour)).Error)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", third.ID).UpdateColumn("updated_at", base.Add(2*time.Hour)).Error)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{Limit: 2})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{third.ID, second.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		BeforeID: &second.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, first.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_UsesDefaultLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "default-limit")
	db := database.Conn()
	created, err := factoryModel.CreateWorkOrder(db, "Default page", "", &callerID, nil, nil)
	require.NoError(t, err)

	orders, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, created.ID, orders[0].ID)
}

func TestFactory_ListWorkOrders_PagesByCreatedAt(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "created-sort")
	db := database.Conn()
	oldest, middle, newest := createTimedOrders(t, db, factoryModel, callerID)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 2,
		Sort:  WorkOrderListSortCreated,
	})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{newest.ID, middle.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		Sort:     WorkOrderListSortCreated,
		BeforeID: &middle.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, oldest.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_PagesByCreatedAtAscending(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "created-asc")
	db := database.Conn()
	oldest, middle, newest := createTimedOrders(t, db, factoryModel, callerID)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         2,
		Sort:          WorkOrderListSortCreated,
		SortDirection: WorkOrderListDirectionAsc,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{oldest.ID, middle.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         2,
		Sort:          WorkOrderListSortCreated,
		SortDirection: WorkOrderListDirectionAsc,
		BeforeID:      &middle.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, newest.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_ConfidenceSortKeepsMissingLast(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "confidence-sort")
	db := database.Conn()
	high := createScoredOrder(t, db, factoryModel, callerID, "High", 5)
	low := createScoredOrder(t, db, factoryModel, callerID, "Low", 2)
	missing, err := factoryModel.CreateWorkOrder(db, "Missing", "", &callerID, nil, nil)
	require.NoError(t, err)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 2,
		Sort:  WorkOrderListSortConfidence,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{high.ID, low.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		Sort:     WorkOrderListSortConfidence,
		BeforeID: &low.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, missing.ID, next[0].ID)

	ascending, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Sort:          WorkOrderListSortConfidence,
		SortDirection: WorkOrderListDirectionAsc,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{low.ID, high.ID, missing.ID}, workOrderIDs(ascending))
}

func TestFactory_ListWorkOrders_FiltersConfidenceAndAge(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "confidence-age")
	db := database.Conn()
	high := createScoredOrder(t, db, factoryModel, callerID, "High", 4)
	low := createScoredOrder(t, db, factoryModel, callerID, "Low", 1)
	missing, err := factoryModel.CreateWorkOrder(db, "Missing", "", &callerID, nil, nil)
	require.NoError(t, err)
	old, err := factoryModel.CreateWorkOrder(db, "Old", "", &callerID, nil, nil)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", old.ID).UpdateColumn("created_at", now.Add(-100*24*time.Hour)).Error)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", high.ID).UpdateColumn("created_at", now.Add(-2*24*time.Hour)).Error)

	minScore := 3.0
	atLeastThree, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{MinConfidence: &minScore})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{high.ID}, workOrderIDs(atLeastThree))

	onlyMissing, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{ConfidenceMissing: true})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{missing.ID, old.ID}, workOrderIDs(onlyMissing))
	assert.NotContains(t, workOrderIDs(onlyMissing), low.ID)

	recent, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{Age: WorkOrderListAgeLast7Days})
	require.NoError(t, err)
	assert.NotContains(t, workOrderIDs(recent), old.ID)
	assert.Contains(t, workOrderIDs(recent), high.ID)

	older, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{Age: WorkOrderListAgeOlderThan90Days})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{old.ID}, workOrderIDs(older))
}

func TestFactory_ListWorkOrders_FiltersAndSortsSourceAcrossPages(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, callerID, factoryModel := setupFactoryWithUser(t, "source-sort")
	db := database.Conn()
	manual, err := factoryModel.CreateWorkOrder(db, "Manual", "", &callerID, nil, nil)
	require.NoError(t, err)
	github := createOriginOrder(t, db, factoryModel, callerID, "GitHub", "https://github.com/acme/payments/issues/12")
	sentry := createOriginOrder(t, db, factoryModel, callerID, "Sentry", "https://acme.sentry.io/issues/99/")
	jira := createOriginOrder(t, db, factoryModel, callerID, "Jira", "https://acme.atlassian.net/browse/ENG-42")
	slack := createOriginOrder(t, db, factoryModel, callerID, "Slack", "https://acme.slack.com/archives/C1")
	pagerduty := createOriginOrder(t, db, factoryModel, callerID, "PagerDuty", "https://acme.pagerduty.com/incidents/P1")
	productive := createOriginOrder(t, db, factoryModel, callerID, "Productive", "https://app.productive.io/1/tasks/9")
	dependabot := createOriginOrder(t, db, factoryModel, callerID, "Dependabot", "https://github.com/acme/payments/security/dependabot/1")

	canvas, _ := support.CreateCanvas(t, org.ID, callerID, nil, nil)
	require.NoError(t, db.Model(&Canvas{}).Where("id = ?", canvas.ID).Update("name", "Sentry intake").Error)
	run, err := CreateCanvasRunInTransaction(db, canvas.ID, "ingest", CanvasRunStateFinished, CanvasRunResultPassed)
	require.NoError(t, err)
	fromAutomation, err := factoryModel.CreateWorkOrder(db, "Automation", "", &callerID, nil, &run.ID)
	require.NoError(t, err)
	originWins := createOriginOrder(t, db, factoryModel, callerID, "Origin wins", "https://github.com/acme/payments/issues/3")
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", originWins.ID).Update("source_run_id", run.ID).Error)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         4,
		Sort:          WorkOrderListSortSource,
		SortDirection: WorkOrderListDirectionAsc,
	})
	require.NoError(t, err)
	require.Len(t, page, 4)
	assert.Equal(t, manual.ID, page[0].ID)

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         20,
		Sort:          WorkOrderListSortSource,
		SortDirection: WorkOrderListDirectionAsc,
		BeforeID:      &page[3].ID,
	})
	require.NoError(t, err)
	assert.NotContains(t, workOrderIDs(next), page[0].ID)
	assert.NotContains(t, workOrderIDs(next), page[3].ID)

	githubOnly, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:        1,
		SourceGroups: []string{WorkOrderSourceGroupGitHubIssues},
		Sort:         WorkOrderListSortCreated,
	})
	require.NoError(t, err)
	require.Len(t, githubOnly, 1)
	githubNext, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:        10,
		SourceGroups: []string{WorkOrderSourceGroupGitHubIssues},
		Sort:         WorkOrderListSortCreated,
		BeforeID:     &githubOnly[0].ID,
	})
	require.NoError(t, err)
	githubIDs := append(workOrderIDs(githubOnly), workOrderIDs(githubNext)...)
	assert.ElementsMatch(t, []uuid.UUID{github.ID, dependabot.ID, originWins.ID}, githubIDs)
	assert.NotContains(t, githubIDs, sentry.ID)

	sentryOnly, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		SourceGroups: []string{WorkOrderSourceGroupSentryExceptions},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{sentry.ID, fromAutomation.ID}, workOrderIDs(sentryOnly))

	manualOnly, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		SourceGroups: []string{WorkOrderSourceGroupManual},
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{manual.ID}, workOrderIDs(manualOnly))

	unknown, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		SourceGroups: []string{"dependabot-alerts"},
	})
	require.NoError(t, err)
	assert.Empty(t, unknown)

	grouped, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		SourceGroups: []string{
			WorkOrderSourceGroupJiraIssues,
			WorkOrderSourceGroupSlack,
			WorkOrderSourceGroupPagerDutyIncidents,
			WorkOrderSourceGroupProductiveTasks,
		},
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{jira.ID, slack.ID, pagerduty.ID, productive.ID}, workOrderIDs(grouped))
}

func createTimedOrders(t *testing.T, db *gorm.DB, factoryModel *Factory, callerID uuid.UUID) (*FactoryWorkOrder, *FactoryWorkOrder, *FactoryWorkOrder) {
	t.Helper()

	oldest, err := factoryModel.CreateWorkOrder(db, "Oldest created", "", &callerID, nil, nil)
	require.NoError(t, err)
	middle, err := factoryModel.CreateWorkOrder(db, "Middle created", "", &callerID, nil, nil)
	require.NoError(t, err)
	newest, err := factoryModel.CreateWorkOrder(db, "Newest created", "", &callerID, nil, nil)
	require.NoError(t, err)

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	setWorkOrderTimes(t, db, oldest.ID, base, base.Add(2*time.Hour))
	setWorkOrderTimes(t, db, middle.ID, base.Add(time.Hour), base)
	setWorkOrderTimes(t, db, newest.ID, base.Add(2*time.Hour), base.Add(time.Hour))
	return oldest, middle, newest
}

func setWorkOrderTimes(t *testing.T, db *gorm.DB, id uuid.UUID, created, updated time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", id).UpdateColumns(map[string]any{
		"created_at": created,
		"updated_at": updated,
	}).Error)
}

func createScoredOrder(t *testing.T, db *gorm.DB, factoryModel *Factory, callerID uuid.UUID, title string, score float64) *FactoryWorkOrder {
	t.Helper()
	order, err := factoryModel.CreateWorkOrder(db, title, "", &callerID, nil, nil)
	require.NoError(t, err)
	_, err = order.ReportCheck(db, FactoryWorkOrderCheckParams{
		Key:      PlanningConfidenceCheckKey,
		Name:     PlanningConfidenceCheckName,
		Score:    score,
		MaxScore: 5,
		Level:    FactoryWorkOrderCheckLevelNeutral,
		Summary:  "Score",
	})
	require.NoError(t, err)
	return order
}

func createOriginOrder(t *testing.T, db *gorm.DB, factoryModel *Factory, callerID uuid.UUID, title, rawURL string) *FactoryWorkOrder {
	t.Helper()
	order, err := factoryModel.CreateWorkOrderWithOrigin(db, title, "", &callerID, nil, nil, WorkOrderOrigin{
		URL:   rawURL,
		Label: title,
	})
	require.NoError(t, err)
	return order
}
