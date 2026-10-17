package mcpserver

import (
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestTaskRunPageLimit(t *testing.T) {
	limit, err := taskRunPageLimit(nil)
	require.NoError(t, err)
	assert.Equal(t, models.DefaultTaskRunHistoryLimit, limit)

	limit, err = taskRunPageLimit(float64(500))
	require.NoError(t, err)
	assert.Equal(t, models.MaxTaskRunHistoryLimit, limit)

	_, err = taskRunPageLimit(float64(0))
	require.Error(t, err)
	limit, err = taskRunPageLimit("12")
	require.NoError(t, err)
	assert.Equal(t, 12, limit)
}

func TestListTasksMatchesSourceURL(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Origin", "", "ORG")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "MCP Origin Other", "", "ORO")
	require.NoError(t, err)
	otherUser := support.CreateUser(t, r, r.Organization.ID)
	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, factoryModel.ID)

	githubURL := "https://github.com/Acme/payments/issues/12"
	draftMine := mustOriginTask(t, factoryModel, "Draft mine", "Keep the draft.", githubURL, &r.User)
	openOther := mustOriginTask(t, factoryModel, "Open other", "", "https://github.com/acme/payments/issues/12?utm=1", &otherUser.ID)
	_, err = openOther.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen, Actor: &otherUser.ID})
	require.NoError(t, err)
	require.NoError(t, db.Model(openOther).UpdateColumn("updated_at", time.Now().Add(-time.Hour)).Error)
	require.NoError(t, db.Model(draftMine).UpdateColumn("updated_at", time.Now().Add(-2*time.Hour)).Error)

	mustOriginTask(t, factoryModel, "Issue 123", "", "https://github.com/acme/payments/issues/123", &r.User)
	mustOriginTask(t, factoryModel, "Pull 12", "", "https://github.com/acme/payments/pull/12", &r.User)
	linear := mustOriginTask(t, factoryModel, "Linear slug", "", "https://linear.app/acme/issue/ENG-12/fix-payments", &r.User)
	mustOriginTask(t, factoryModel, "Linear longer", "", "https://linear.app/acme/issue/ENG-123/other", &r.User)
	jira := mustOriginTask(t, factoryModel, "Jira", "", "https://acme.atlassian.net/browse/ENG-12", &r.User)
	mustOriginTask(t, factoryModel, "Jira longer", "", "https://acme.atlassian.net/browse/ENG-123", &r.User)
	custom := mustOriginTask(t, factoryModel, "Custom", "", "https://Example.com/incidents/P123ABC/", &r.User)
	mustOriginTask(t, factoryModel, "Custom longer", "", "https://example.com/incidents/P123ABCD", &r.User)
	sentry := mustOriginTask(t, factoryModel, "Sentry", "", "https://acme.sentry.io/organizations/acme/issues/12/", &r.User)
	mustOriginTask(t, factoryModel, "Sentry longer", "", "https://acme.sentry.io/issues/123", &r.User)
	archived := mustOriginTask(t, factoryModel, "Archived", "", "https://github.com/acme/payments/issues/99", &r.User)
	_, err = archived.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultArchived,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	otherFactoryTask := mustOriginTask(t, otherFactory, "Other factory", "", githubURL, &r.User)

	githubIDs := listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://github.com/acme/payments/issues/12#issuecomment-1",
	})
	assert.Equal(t, []string{openOther.ID.String(), draftMine.ID.String()}, githubIDs)

	paged := listTasks(t, runtime, claims, map[string]any{
		"source_url": githubURL,
		"before_id":  openOther.ID.String(),
	})
	assert.Equal(t, []string{draftMine.ID.String()}, taskIDs(t, paged))
	assert.Equal(t, false, paged["has_next_page"])

	mine := listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": githubURL,
		"mine":       true,
		"states":     []any{"draft"},
	})
	assert.Equal(t, []string{draftMine.ID.String()}, mine)

	openOnly := listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": githubURL,
		"states":     []any{"open"},
	})
	assert.Equal(t, []string{openOther.ID.String()}, openOnly)

	assert.Equal(t, []string{linear.ID.String()}, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://linear.app/acme/issue/ENG-12",
	}))
	assert.NotContains(t, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://linear.app/acme/issue/ENG-123",
	}), linear.ID.String())
	assert.Equal(t, []string{jira.ID.String()}, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://acme.atlassian.net/browse/ENG-12?focusedCommentId=1",
	}))
	assert.NotContains(t, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://acme.atlassian.net/browse/ENG-123",
	}), jira.ID.String())
	assert.Equal(t, []string{custom.ID.String()}, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://example.com/incidents/P123ABC#note",
	}))
	assert.Equal(t, []string{sentry.ID.String()}, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://acme.sentry.io/issues/12",
	}))
	assert.NotContains(t, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://acme.sentry.io/issues/123",
	}), sentry.ID.String())
	assert.Empty(t, listTaskIDs(t, runtime, claims, map[string]any{
		"source_url": "https://github.com/acme/payments/issues/99",
	}))

	rows := listTasks(t, runtime, claims, map[string]any{"source_url": githubURL})
	first, ok := rows["tasks"].([]any)[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://github.com/acme/payments/issues/12?utm=1", first["source_url"])
	assert.Contains(t, first, "assignees")

	_, err = runtime.CallTool(ctx, claims, "list_tasks", map[string]any{"source_url": "12"})
	require.Error(t, err)
	assert.Equal(t, "source_url must be an absolute http URL", err.Error())
	_, err = runtime.CallTool(ctx, claims, "list_tasks", map[string]any{"source_url": "ENG-12"})
	require.Error(t, err)

	otherClaims := toolClaims(r, otherFactory.ID)
	assert.Equal(t, []string{otherFactoryTask.ID.String()}, listTaskIDs(t, runtime, otherClaims, map[string]any{
		"source_url": githubURL,
	}))
	assert.NotContains(t, listTaskIDs(t, runtime, claims, map[string]any{"source_url": githubURL}), otherFactoryTask.ID.String())
}

func TestGetTaskReportReturnsUsageAndRunHistory(t *testing.T) {
	r := support.Setup(t)
	enableFactories(t, r.Organization.ID)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "MCP Report", "", "RPT")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "MCP Report Other", "", "RPO")
	require.NoError(t, err)
	implement, entry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "implement", "start")
	checks, checksEntry := support.CreateFactoryAppWithOnRunTrigger(t, r, factoryModel.ID, "checks", "start-checks")
	line, err := factoryModel.CreateLine(db, "ship", []models.FactoryLineStep{})
	require.NoError(t, err)

	origin := "https://github.com/acme/payments/issues/12"
	order, err := factoryModel.CreateWorkOrderWithOrigin(db, "Ship the report", "Record the usage.", &r.User, nil, nil, models.WorkOrderOrigin{URL: origin})
	require.NoError(t, err)
	empty, err := factoryModel.CreateWorkOrder(db, "No usage", "Nothing recorded.", &r.User, nil, nil)
	require.NoError(t, err)
	archived := mustOriginTask(t, factoryModel, "Archived report", "Still readable.", "https://github.com/acme/payments/issues/77", &r.User)
	_, err = archived.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultArchived,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	source, err := models.CreateCanvasRunInTransaction(db, implement.ID, entry, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	require.NoError(t, err)
	_, err = factoryModel.CreateWorkOrder(db, "Sibling", "", &r.User, nil, &source.ID)
	require.NoError(t, err)
	require.NoError(t, db.Model(order).Update("source_run_id", source.ID).Error)
	order.SourceRunID = &source.ID

	unlinkedChild, err := models.CreateCanvasRunInTransaction(db, implement.ID, entry, models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, db.Model(unlinkedChild).Update("parent_run_id", source.ID).Error)

	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	stamp := func(offset int) time.Time { return base.Add(time.Duration(offset) * time.Second) }

	planningRun := mustStampRun(t, implement.ID, entry, models.CanvasRunStateFinished, models.CanvasRunResultPassed, stamp(1), nil)
	require.NoError(t, db.Create(&models.FactoryPlanningSession{
		ID:               uuid.New(),
		OrganizationID:   factoryModel.OrganizationID,
		FactoryID:        factoryModel.ID,
		Repository:       "acme/payments",
		Kind:             models.PlanningSessionKindWorkOrderAnalysis,
		State:            models.PlanningSessionStateEnded,
		CanvasID:         &implement.ID,
		CanvasRunID:      &planningRun.ID,
		DraftWorkOrderID: &order.ID,
		HeartbeatAt:      stamp(1),
		CreatedAt:        stamp(1),
		UpdatedAt:        stamp(1),
	}).Error)

	attempt1 := mustStampRun(t, implement.ID, entry, models.CanvasRunStateFinished, models.CanvasRunResultPassed, stamp(2), nil)
	attempt2 := mustStampRun(t, implement.ID, entry, models.CanvasRunStateFinished, models.CanvasRunResultFailed, stamp(3), nil)
	stepChild := mustStampRun(t, implement.ID, entry, models.CanvasRunStateFinished, models.CanvasRunResultPassed, stamp(4), &attempt2.ID)
	linkedChild := mustStampRun(t, implement.ID, entry, models.CanvasRunStateStarted, "", stamp(5), &source.ID)
	require.NoError(t, db.Model(linkedChild).Update("input", models.NewJSONValue(map[string]any{
		"work_order": map[string]any{"id": order.ID.String()},
	})).Error)
	prRun := mustStampRun(t, checks.ID, checksEntry, models.CanvasRunStateFinished, models.CanvasRunResultFailed, stamp(6), nil)

	dispatch := &models.FactoryWorkOrderLineDispatch{
		ID:             uuid.New(),
		OrganizationID: factoryModel.OrganizationID,
		FactoryID:      factoryModel.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineName:       line.Name,
		Steps:          datatypes.JSONSlice[models.FactoryLineStep]{},
		State:          models.FactoryWorkOrderLineDispatchStateFinished,
		Result:         models.CanvasRunResultPassed,
		Model:          "claude-sonnet-4-6",
		CreatedAt:      stamp(2),
		UpdatedAt:      stamp(3),
	}
	require.NoError(t, db.Create(dispatch).Error)
	execution1 := mustExecution(t, factoryModel, order, line, dispatch, attempt1, 0, models.CanvasRunResultPassed, 999)
	mustExecution(t, factoryModel, order, line, dispatch, attempt2, 0, models.CanvasRunResultFailed, 0)

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/payments",
		Number:     8,
		URL:        "https://github.com/acme/payments/pull/8",
		Title:      "Record task usage",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.FactoryPullRequestRun{
		PullRequestID: pullRequest.ID,
		RunID:         prRun.ID,
		Title:         "Fixing failed checks",
		Description:   "The pull request activity recorded the failed checks.",
		State:         "finished",
		Access:        "concurrent",
		CreatedAt:     stamp(6),
		UpdatedAt:     stamp(6),
	}).Error)

	require.NoError(t, db.Create(&models.WorkspaceUsageEvent{
		ID:                   uuid.New(),
		OrganizationID:       factoryModel.OrganizationID,
		FactoryID:            &factoryModel.ID,
		WorkOrderID:          &order.ID,
		WorkOrderExecutionID: &execution1.ID,
		CanvasRunID:          attempt1.ID,
		NodeExecutionID:      uuid.New(),
		NodeID:               "prompt",
		Provider:             models.UsageProviderAnthropic,
		Model:                "claude-sonnet-4-6",
		UsageKind:            models.UsageKindModel,
		FundingSource:        models.UsageFundingSourceBYOK,
		TotalTokens:          120,
		CostMicros:           450_000,
		Currency:             "usd",
		PriceBookVersion:     "test",
		IdempotencyKey:       uuid.NewString(),
		OccurredAt:           stamp(2),
		CreatedAt:            stamp(2),
	}).Error)
	require.NoError(t, db.Create(&models.WorkspaceUsageEvent{
		ID:               uuid.New(),
		OrganizationID:   factoryModel.OrganizationID,
		FactoryID:        &factoryModel.ID,
		WorkOrderID:      &order.ID,
		CanvasRunID:      attempt1.ID,
		NodeExecutionID:  uuid.New(),
		NodeID:           "runner",
		Provider:         models.UsageProviderRunner,
		Model:            "runner",
		UsageKind:        models.UsageKindCompute,
		FundingSource:    models.UsageFundingSourceBYOK,
		DurationSeconds:  40,
		MachineType:      "e1-large-amd64",
		CostMicros:       280_000,
		Currency:         "usd",
		PriceBookVersion: "test",
		IdempotencyKey:   uuid.NewString(),
		OccurredAt:       stamp(2),
		CreatedAt:        stamp(2),
	}).Error)

	runtime := &Runtime{Auth: r.AuthService}
	claims := toolClaims(r, factoryModel.ID)
	described, err := factories.DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
	})
	require.NoError(t, err)

	report := callReport(t, runtime, claims, map[string]any{"task": factoryModel.WorkOrderKey(order.Number), "limit": float64(1)})
	assert.Equal(t, factoryModel.ID.String(), report["factory"].(map[string]any)["id"])
	task := report["task"].(map[string]any)
	assert.Equal(t, order.ID.String(), task["id"])
	assert.Equal(t, "Ship the report", task["title"])
	assert.Equal(t, "Record the usage.", task["description"])
	assert.Equal(t, origin, task["source_url"])
	assert.Equal(t, "draft", task["state"])

	usage := report["usage"].(map[string]any)
	assert.Equal(t, true, usage["recorded"])
	assert.Equal(t, float64(described.GetOrder().GetTotalTokens()), usage["total_tokens"])
	assert.Equal(t, float64(described.GetOrder().GetTotalCostCents()), usage["total_cost_cents"])
	assert.Equal(t, float64(described.GetOrder().GetTotalDurationSeconds()), usage["vm_duration_seconds"])
	assert.Equal(t, float64(120), usage["total_tokens"])
	assert.Equal(t, float64(73), usage["total_cost_cents"])
	assert.Equal(t, float64(40), usage["vm_duration_seconds"])
	assert.NotEqual(t, float64(999), usage["total_tokens"])

	modelsUsage := report["usage_by_model"].([]any)
	require.Len(t, modelsUsage, len(described.GetOrder().GetUsageByModel()))
	assert.Equal(t, described.GetOrder().GetUsageByModel()[0].GetModel(), modelsUsage[0].(map[string]any)["model"])
	assert.Equal(t, float64(described.GetOrder().GetUsageByModel()[0].GetCostCents()), modelsUsage[0].(map[string]any)["cost_cents"])
	machines := report["usage_by_machine_type"].([]any)
	require.Len(t, machines, 1)
	assert.Equal(t, "e1-large-amd64", machines[0].(map[string]any)["machine_type"])
	assert.Equal(t, float64(described.GetOrder().GetUsageByMachineType()[0].GetDurationSeconds()), machines[0].(map[string]any)["vm_duration_seconds"])

	dispatches := report["line_dispatches"].([]any)
	require.Len(t, dispatches, 1)
	steps := dispatches[0].(map[string]any)["steps"].([]any)
	require.Len(t, steps, 2)
	var cachedTokens float64
	for _, step := range steps {
		if tokens, ok := step.(map[string]any)["total_tokens"].(float64); ok {
			cachedTokens = tokens
		}
	}
	assert.Equal(t, float64(999), cachedTokens)

	pulls := report["pull_requests"].([]any)
	require.Len(t, pulls, 1)
	pull := pulls[0].(map[string]any)
	assert.Equal(t, "https://github.com/acme/payments/pull/8", pull["url"])
	assert.Equal(t, "acme/payments", pull["repository"])
	assert.Equal(t, "Record task usage", pull["title"])
	assert.Equal(t, "open", pull["state"])
	activities := pull["activities"].([]any)
	require.Len(t, activities, 1)
	assert.Equal(t, "Fixing failed checks", activities[0].(map[string]any)["title"])
	assert.Equal(t, "The pull request activity recorded the failed checks.", activities[0].(map[string]any)["description"])
	_, hasCI := report["ci_failures"]
	assert.False(t, hasCI)

	expected := map[string]bool{
		planningRun.ID.String(): true,
		attempt1.ID.String():    true,
		attempt2.ID.String():    true,
		stepChild.ID.String():   true,
		linkedChild.ID.String(): true,
		prRun.ID.String():       true,
	}
	seen := map[string]bool{}
	page := report
	counts := page["run_counts"]
	var pages int
	for {
		pages++
		require.LessOrEqual(t, pages, len(expected)+1)
		runs := page["runs"].([]any)
		require.Len(t, runs, 1)
		run := runs[0].(map[string]any)
		id, _ := run["id"].(string)
		require.NotContains(t, seen, id)
		seen[id] = true
		assert.Equal(t, counts, page["run_counts"])
		if id == prRun.ID.String() {
			assert.Equal(t, "The pull request activity recorded the failed checks.", run["activity_description"])
			assert.Equal(t, checks.Name, run["automation_name"])
			assert.Equal(t, "failed", run["result"])
		}
		if id == attempt1.ID.String() {
			assert.Equal(t, true, run["usage_recorded"])
			assert.Equal(t, float64(120), run["total_tokens"])
		}
		if id == stepChild.ID.String() {
			assert.Equal(t, attempt2.ID.String(), run["parent_run_id"])
			assert.Equal(t, false, run["usage_recorded"])
			_, hasTokens := run["total_tokens"]
			assert.False(t, hasTokens)
		}
		if page["has_next_run_page"] != true {
			break
		}
		page = callReport(t, runtime, claims, map[string]any{
			"task":   order.ID.String(),
			"limit":  1,
			"before": page["next_run_before"],
		})
	}
	assert.Equal(t, expected, seen)
	assert.NotContains(t, seen, source.ID.String())
	assert.NotContains(t, seen, unlinkedChild.ID.String())

	var total int64
	for _, row := range counts.([]any) {
		total += int64(row.(map[string]any)["count"].(float64))
	}
	assert.Equal(t, int64(len(expected)), total)

	emptyReport := callReport(t, runtime, claims, map[string]any{"task": factoryModel.WorkOrderKey(empty.Number)})
	emptyUsage := emptyReport["usage"].(map[string]any)
	assert.Equal(t, false, emptyUsage["recorded"])
	_, hasTokens := emptyUsage["total_tokens"]
	assert.False(t, hasTokens)
	assert.Empty(t, emptyReport["usage_by_model"].([]any))

	archivedReport := callReport(t, runtime, claims, map[string]any{"task": strconv.FormatInt(archived.Number, 10)})
	assert.Equal(t, "closed", archivedReport["task"].(map[string]any)["state"])
	assert.Equal(t, "archived", archivedReport["task"].(map[string]any)["result"])

	_, err = runtime.CallTool(ctx, claims, "get_task", map[string]any{"task": archived.ID.String()})
	require.Error(t, err)
	assert.Equal(t, "Not found", err.Error())

	plain := callTask(t, runtime, claims, order.ID.String())
	assert.Equal(t, order.ID.String(), plain["id"])
	assert.NotContains(t, plain, "usage")
	assert.NotContains(t, plain, "runs")
	assert.Contains(t, plain, "description")

	_, err = runtime.CallTool(ctx, toolClaims(r, otherFactory.ID), "get_task_report", map[string]any{"task": order.ID.String()})
	require.Error(t, err)
	assert.Equal(t, "Not found", err.Error())
}

func mustOriginTask(t *testing.T, factory *models.Factory, title, description, rawURL string, user *uuid.UUID) *models.FactoryWorkOrder {
	t.Helper()
	order, err := factory.CreateWorkOrderWithOrigin(database.DB(t.Context()), title, description, user, nil, nil, models.WorkOrderOrigin{URL: rawURL})
	require.NoError(t, err)
	return order
}

func mustStampRun(t *testing.T, workflowID uuid.UUID, entry, state, result string, stamp time.Time, parent *uuid.UUID) *models.CanvasRun {
	t.Helper()
	tx := database.DB(t.Context())
	run, err := models.CreateCanvasRunInTransaction(tx, workflowID, entry, state, result)
	require.NoError(t, err)
	updates := map[string]any{"created_at": stamp, "updated_at": stamp}
	if parent != nil {
		updates["parent_run_id"] = *parent
	}
	if state == models.CanvasRunStateFinished {
		updates["finished_at"] = stamp
	}
	require.NoError(t, tx.Model(run).UpdateColumns(updates).Error)
	run.CreatedAt = &stamp
	run.UpdatedAt = &stamp
	run.ParentRunID = parent
	return run
}

func mustExecution(
	t *testing.T,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	line *models.FactoryLine,
	dispatch *models.FactoryWorkOrderLineDispatch,
	run *models.CanvasRun,
	index int,
	result string,
	tokens int64,
) *models.FactoryWorkOrderExecution {
	t.Helper()
	finished := time.Now()
	execution := &models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: factory.OrganizationID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      index,
		StepName:       "implement",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         result,
		TotalTokens:    tokens,
		CreatedAt:      finished,
		UpdatedAt:      finished,
		FinishedAt:     &finished,
	}
	require.NoError(t, database.DB(t.Context()).Create(execution).Error)
	return execution
}

func listTasks(t *testing.T, runtime *Runtime, claims *AccessClaims, args map[string]any) map[string]any {
	t.Helper()
	result, err := runtime.CallTool(t.Context(), claims, "list_tasks", args)
	require.NoError(t, err)
	return decodeToolJSON(t, result)
}

func listTaskIDs(t *testing.T, runtime *Runtime, claims *AccessClaims, args map[string]any) []string {
	t.Helper()
	return taskIDs(t, listTasks(t, runtime, claims, args))
}

func taskIDs(t *testing.T, payload map[string]any) []string {
	t.Helper()
	rows, ok := payload["tasks"].([]any)
	require.True(t, ok)
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		item, ok := row.(map[string]any)
		require.True(t, ok)
		id, _ := item["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

func callReport(t *testing.T, runtime *Runtime, claims *AccessClaims, args map[string]any) map[string]any {
	t.Helper()
	result, err := runtime.CallTool(t.Context(), claims, "get_task_report", args)
	require.NoError(t, err)
	return decodeToolJSON(t, result)
}

func callTask(t *testing.T, runtime *Runtime, claims *AccessClaims, task string) map[string]any {
	t.Helper()
	result, err := runtime.CallTool(t.Context(), claims, "get_task", map[string]any{"task": task})
	require.NoError(t, err)
	return decodeToolJSON(t, result)
}
