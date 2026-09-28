package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
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

func TestFactory_ListWorkOrders_PagesByConfidence(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "confidence-sort")
	db := database.Conn()

	low := createScoredDraft(t, factoryModel, callerID, "Low", 1)
	high := createScoredDraft(t, factoryModel, callerID, "High", 5)
	missing := createDraft(t, factoryModel, callerID, "Missing")
	otherCheck := createDraft(t, factoryModel, callerID, "Other check")
	_, err := otherCheck.ReportCheck(db, FactoryWorkOrderCheckParams{
		Key:      "clarity",
		Name:     "Clarity score",
		Score:    5,
		MaxScore: 5,
		Level:    FactoryWorkOrderCheckLevelPositive,
	})
	require.NoError(t, err)

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	setWorkOrderUpdatedAt(t, db, missing.ID, base.Add(2*time.Hour))
	setWorkOrderUpdatedAt(t, db, low.ID, base.Add(time.Hour))
	setWorkOrderUpdatedAt(t, db, high.ID, base)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 2,
		Sort:  FactoryWorkOrderListSortConfidence,
	})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{high.ID, low.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		Sort:     FactoryWorkOrderListSortConfidence,
		BeforeID: &low.ID,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{missing.ID, otherCheck.ID}, workOrderIDs(next))

	ascending, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         2,
		Sort:          FactoryWorkOrderListSortConfidence,
		SortDirection: FactoryWorkOrderListSortDirectionAsc,
	})
	require.NoError(t, err)
	require.Len(t, ascending, 2)
	assert.Equal(t, []uuid.UUID{low.ID, high.ID}, []uuid.UUID{ascending[0].ID, ascending[1].ID})

	ascendingNext, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         2,
		Sort:          FactoryWorkOrderListSortConfidence,
		SortDirection: FactoryWorkOrderListSortDirectionAsc,
		BeforeID:      &high.ID,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{missing.ID, otherCheck.ID}, workOrderIDs(ascendingNext))
}

func TestFactory_ListWorkOrders_PagesByCreatedAt(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "created-sort")
	db := database.Conn()

	oldest := createDraft(t, factoryModel, callerID, "Oldest issue")
	middle := createDraft(t, factoryModel, callerID, "Middle issue")
	newest := createDraft(t, factoryModel, callerID, "Newest issue")

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	setWorkOrderCreatedAt(t, db, oldest.ID, base)
	setWorkOrderCreatedAt(t, db, middle.ID, base.Add(time.Hour))
	setWorkOrderCreatedAt(t, db, newest.ID, base.Add(2*time.Hour))
	setWorkOrderUpdatedAt(t, db, oldest.ID, base.Add(2*time.Hour))
	setWorkOrderUpdatedAt(t, db, newest.ID, base)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 2,
		Sort:  FactoryWorkOrderListSortCreated,
	})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{newest.ID, middle.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		Sort:     FactoryWorkOrderListSortCreated,
		BeforeID: &middle.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, oldest.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_PagesBySource(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "source-sort")
	db := database.Conn()

	manual := createDraft(t, factoryModel, callerID, "Manual")
	github := createDraft(t, factoryModel, callerID, "GitHub")
	sentry := createDraft(t, factoryModel, callerID, "Sentry")
	setWorkOrderOrigin(t, db, github.ID, "https://github.com/acme/pay/issues/1")
	setWorkOrderOrigin(t, db, sentry.ID, "https://acme.sentry.io/issues/12")

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 2,
		Sort:  FactoryWorkOrderListSortSource,
	})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{manual.ID, github.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		Sort:     FactoryWorkOrderListSortSource,
		BeforeID: &github.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, sentry.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_FiltersSourcesAcrossPages(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, callerID, factoryModel := setupFactoryWithUser(t, "source-filter")
	db := database.Conn()

	firstGitHub := createDraft(t, factoryModel, callerID, "GitHub one")
	secondGitHub := createDraft(t, factoryModel, callerID, "GitHub two")
	sentry := createDraft(t, factoryModel, callerID, "Sentry")
	manual := createDraft(t, factoryModel, callerID, "Manual")
	jiraAutomation := createDraft(t, factoryModel, callerID, "Jira automation")
	setWorkOrderOrigin(t, db, firstGitHub.ID, "https://github.com/acme/pay/issues/1")
	setWorkOrderOrigin(t, db, secondGitHub.ID, "https://github.com/acme/pay/security/dependabot/7")
	setWorkOrderOrigin(t, db, sentry.ID, "https://acme.sentry.io/issues/9")
	runID := createCreatorRun(t, db, org.ID, callerID, "Jira intake")
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", jiraAutomation.ID).Update("source_run_id", runID).Error)

	filters := ListFactoryWorkOrdersFilters{
		Limit:   1,
		Sources: []string{FactoryWorkOrderSourceGitHubIssues},
		Sort:    FactoryWorkOrderListSortUpdated,
	}
	page, err := factoryModel.ListWorkOrders(db, filters)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Contains(t, []uuid.UUID{firstGitHub.ID, secondGitHub.ID}, page[0].ID)

	filters.BeforeID = &page[0].ID
	next, err := factoryModel.ListWorkOrders(db, filters)
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.NotEqual(t, page[0].ID, next[0].ID)
	assert.Contains(t, []uuid.UUID{firstGitHub.ID, secondGitHub.ID}, next[0].ID)
	assert.NotContains(t, workOrderIDs(append(page, next...)), sentry.ID)
	assert.NotContains(t, workOrderIDs(append(page, next...)), manual.ID)

	jiraOnly, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:   10,
		Sources: []string{FactoryWorkOrderSourceJiraIssues},
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{jiraAutomation.ID}, workOrderIDs(jiraOnly))

	manualOnly, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:   10,
		Sources: []string{FactoryWorkOrderSourceManual},
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{manual.ID}, workOrderIDs(manualOnly))
}

func TestFactory_ListWorkOrders_FiltersConfidenceAndAge(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "confidence-age")
	db := database.Conn()

	low := createScoredDraft(t, factoryModel, callerID, "Low", 2)
	high := createScoredDraft(t, factoryModel, callerID, "High", 4)
	missing := createDraft(t, factoryModel, callerID, "Missing")
	old := createScoredDraft(t, factoryModel, callerID, "Old", 5)
	setWorkOrderCreatedAt(t, db, old.ID, time.Now().UTC().Add(-100*24*time.Hour))

	min := 3.0
	atLeastThree, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         10,
		MinConfidence: &min,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{high.ID, old.ID}, workOrderIDs(atLeastThree))

	onlyMissing, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:             10,
		ConfidenceMissing: true,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{missing.ID}, workOrderIDs(onlyMissing))

	recent, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 10,
		Age:   FactoryWorkOrderListAgeLast7Days,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{low.ID, high.ID, missing.ID}, workOrderIDs(recent))

	older, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit: 10,
		Age:   FactoryWorkOrderListAgeOlderThan90Days,
	})
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{old.ID}, workOrderIDs(older))

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         1,
		MinConfidence: &min,
		Sort:          FactoryWorkOrderListSortConfidence,
	})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, old.ID, page[0].ID)

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:         1,
		MinConfidence: &min,
		Sort:          FactoryWorkOrderListSortConfidence,
		BeforeID:      &old.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, high.ID, next[0].ID)
	assert.NotContains(t, []uuid.UUID{next[0].ID}, low.ID)
	assert.NotContains(t, []uuid.UUID{next[0].ID}, missing.ID)
}

func TestFactory_ListWorkOrders_ClassifiesSourceGroups(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "source-groups")
	db := database.Conn()

	cases := []struct {
		title  string
		origin string
		group  string
	}{
		{title: "Jira", origin: "https://acme.atlassian.net/browse/PAY-1", group: FactoryWorkOrderSourceJiraIssues},
		{title: "Slack", origin: "https://acme.slack.com/archives/C1", group: FactoryWorkOrderSourceSlack},
		{title: "Productive", origin: "https://app.productive.io/1/tasks/2", group: FactoryWorkOrderSourceProductiveTasks},
		{title: "PagerDuty", origin: "https://acme.pagerduty.com/incidents/1", group: FactoryWorkOrderSourcePagerDutyIncidents},
	}

	for _, tc := range cases {
		order := createDraft(t, factoryModel, callerID, tc.title)
		setWorkOrderOrigin(t, db, order.ID, tc.origin)
		matched, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
			Limit:   10,
			Sources: []string{tc.group},
		})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{order.ID}, workOrderIDs(matched), tc.title)
	}
}

func createDraft(t *testing.T, factoryModel *Factory, callerID uuid.UUID, title string) *FactoryWorkOrder {
	t.Helper()
	order, err := factoryModel.CreateWorkOrder(database.Conn(), title, "", &callerID, nil, nil)
	require.NoError(t, err)
	return order
}

func createScoredDraft(t *testing.T, factoryModel *Factory, callerID uuid.UUID, title string, score float64) *FactoryWorkOrder {
	t.Helper()
	order := createDraft(t, factoryModel, callerID, title)
	_, err := order.ReportCheck(database.Conn(), FactoryWorkOrderCheckParams{
		Key:      PlanningConfidenceCheckKey,
		Name:     PlanningConfidenceCheckName,
		Score:    score,
		MaxScore: 5,
		Level:    FactoryWorkOrderCheckLevelNeutral,
	})
	require.NoError(t, err)
	return order
}

func setWorkOrderUpdatedAt(t *testing.T, db *gorm.DB, id uuid.UUID, at time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", id).UpdateColumn("updated_at", at).Error)
}

func setWorkOrderCreatedAt(t *testing.T, db *gorm.DB, id uuid.UUID, at time.Time) {
	t.Helper()
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", id).UpdateColumn("created_at", at).Error)
}

func setWorkOrderOrigin(t *testing.T, db *gorm.DB, id uuid.UUID, originURL string) {
	t.Helper()
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", id).Update("origin_url", originURL).Error)
}

func createCreatorRun(t *testing.T, db *gorm.DB, organizationID, userID uuid.UUID, canvasName string) uuid.UUID {
	t.Helper()
	now := time.Now()
	versionID := uuid.New()
	canvas := Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &versionID,
		Name:           canvasName,
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	version := CanvasVersion{
		ID:         versionID,
		WorkflowID: canvas.ID,
		OwnerID:    &userID,
		Nodes:      datatypes.NewJSONSlice([]Node{}),
		Edges:      datatypes.NewJSONSlice([]Edge{}),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	run := CanvasRun{
		ID:         uuid.New(),
		WorkflowID: canvas.ID,
		NodeID:     "create-order",
		VersionID:  versionID,
		State:      CanvasRunStateFinished,
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	require.NoError(t, db.Create(&canvas).Error)
	require.NoError(t, db.Create(&version).Error)
	require.NoError(t, db.Create(&run).Error)
	return run.ID
}
