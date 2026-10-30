package contexts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/blob/filesystem"
	factorycomp "github.com/superplanehq/superplane/pkg/components/factory"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	ghdependabot "github.com/superplanehq/superplane/pkg/integrations/github/dependabot"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/integrations/productive"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"github.com/superplanehq/superplane/pkg/storedfiles"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestFactoryContext_CreateWorkOrder(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)

	t.Run("creates work order on factory-owned app", func(t *testing.T) {
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		workOrder, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{
			Title:       "From GitHub issue",
			Description: "Automated intake",
		})
		require.True(t, created)
		require.NoError(t, err)
		assert.Equal(t, "From GitHub issue", workOrder.Title)
		assert.Equal(t, "Automated intake", workOrder.Description)
		assert.NotEmpty(t, workOrder.ID)

		persisted, err := factory.FindWorkOrder(database.Conn(), uuid.MustParse(workOrder.ID))
		require.NoError(t, err)
		require.NotNil(t, persisted.SourceRunID)
		assert.Equal(t, run.ID, *persisted.SourceRunID)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, persisted.State,
			"work orders now start as draft and are promoted to open on first dispatch")
		assert.Empty(t, persisted.Assignees)

		// Creation emits a single `order.status.updated` ("" → draft).
		events, err := persisted.ListEvents(database.Conn(), 0, nil)
		require.NoError(t, err)
		require.Len(t, events, 1)
		assert.Equal(t, factoryevents.EventTypeOrderStatusUpdated, events[0].Type)

		var initialStatus factoryevents.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(events[0].Data, &initialStatus))
		assert.Equal(t, "", initialStatus.FromState)
		assert.Equal(t, models.FactoryWorkOrderStateDraft, initialStatus.ToState)
		// The "" → draft creation event carries the same source-run
		// snapshot as the draft → open promotion so the very first
		// timeline entry already links back to the originating run.
		require.NotNil(t, initialStatus.Run)
		assert.Equal(t, run.ID, initialStatus.Run.ID)
		require.NotNil(t, initialStatus.App)
		assert.Equal(t, canvas.ID, initialStatus.App.ID)

		// On the first `draft → open` promotion the source-run snapshot from
		// `o.SourceRunID` is enriched into the status.updated event so the
		// timeline can link back to the originating canvas run.
		_, err = persisted.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
		})
		require.NoError(t, err)

		statusEvents := listWorkOrderEvents(t, persisted, factoryevents.EventTypeOrderStatusUpdated)
		require.Len(t, statusEvents, 2)
		var opened factoryevents.WorkOrderStatusUpdated
		require.NoError(t, json.Unmarshal(statusEvents[0].Data, &opened))
		assert.Equal(t, models.FactoryWorkOrderStateDraft, opened.FromState)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, opened.ToState)
		require.NotNil(t, opened.Run)
		assert.Equal(t, run.ID, opened.Run.ID)
		require.NotNil(t, opened.App)
		assert.Equal(t, canvas.ID, opened.App.ID)
		assert.Nil(t, opened.User)
	})

	t.Run("sets origin from an intake source event", func(t *testing.T) {
		factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factory.ID, map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"issue": map[string]any{
					"html_url": "https://github.com/acme/payments/issues/12",
					"title":    "Handle duplicate refunds",
				},
			},
		})
		_, err = factory.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
		created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Handle duplicate refunds", DefaultOwnerID: r.User.String()})
		require.NoError(t, err)
		require.True(t, inserted)

		persisted, err := factory.FindWorkOrder(database.Conn(), uuid.MustParse(created.ID))
		require.NoError(t, err)
		require.NotNil(t, persisted.Origin())
		assert.Equal(t, "https://github.com/acme/payments/issues/12", persisted.Origin().URL)
		assert.Equal(t, "acme/payments#12", persisted.Origin().Label)
		require.Len(t, persisted.Assignees, 1)
		assert.Equal(t, r.User, persisted.Assignees[0].UserID)
	})

	t.Run("rejects blank title", func(t *testing.T) {
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		_, _, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "   "})
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrFactoryWorkOrderTitleRequired)
	})

	t.Run("rejects non-factory app", func(t *testing.T) {
		regularCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		ctx := NewFactoryContext(database.Conn(), regularCanvas, nodeExecution)

		_, _, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Should fail"})
		require.Error(t, err)
		assert.EqualError(t, err, "app is not owned by a factory")
	})

	t.Run("rejects run that is executing a work order", func(t *testing.T) {
		line, err := factory.CreateLine(database.Conn(), "ship", nil)
		require.NoError(t, err)

		existingOrder, err := factory.CreateWorkOrder(database.Conn(), "Existing", "", &r.User, nil, nil)
		require.NoError(t, err)

		dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, existingOrder.ID, line.ID, line.Name, nil)

		now := time.Now()
		workOrderExecution := models.FactoryWorkOrderExecution{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			FactoryID:      factory.ID,
			WorkOrderID:    existingOrder.ID,
			LineID:         line.ID,
			LineDispatchID: dispatch.ID,
			StepIndex:      0,
			StepName:       "step-one",
			RunID:          &run.ID,
			Status:         models.FactoryWorkOrderExecutionStatusRunning,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		require.NoError(t, database.Conn().Create(&workOrderExecution).Error)

		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
		_, _, err = ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Nested"})
		require.Error(t, err)
		assert.EqualError(t, err, "cannot create work order while executing another work order")
	})
}

func TestFactoryContext_CreateWorkOrder_SkipsDuplicateSentryIssue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	dependabotPayload := func(number int, manifest string) map[string]any {
		return map[string]any{
			"type": ghdependabot.AlertPayloadType,
			"data": map[string]any{
				"action": "created",
				"alert": map[string]any{
					"number":   number,
					"html_url": "https://github.com/acme/payments/security/dependabot/" + strconv.Itoa(number),
					"dependency": map[string]any{
						"package":       map[string]any{"name": "lodash", "ecosystem": "npm"},
						"manifest_path": manifest,
					},
					"security_advisory": map[string]any{
						"summary":  "Prototype pollution in lodash",
						"severity": "high",
					},
				},
			},
		}
	}

	sentryPayload := func(issueID string) map[string]any {
		return map[string]any{
			"type": "sentry.issue",
			"data": map[string]any{
				"resource": "issue",
				"action":   "created",
				"data": map[string]any{
					"issue": map[string]any{
						"id":        issueID,
						"title":     "boom",
						"permalink": "https://acme.sentry.io/issues/" + issueID + "/",
					},
				},
			},
		}
	}

	countOrders := func(factoryModel *models.Factory) int {
		t.Helper()
		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 100})
		require.NoError(t, err)
		return len(orders)
	}

	t.Run("skips a Sentry issue that already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Existing sentry task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://acme.sentry.io/issues/123/", Label: "boom"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("123"))
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "boom"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("does not skip issue 12 when issue 123 already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Issue 123",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://acme.sentry.io/issues/123/", Label: "123"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("12"))
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "boom"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("skips a self-hosted Sentry issue that already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Existing sentry task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://sentry.internal.example/issues/123/", Label: "boom"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("123"))
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "boom"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("creates a task when the Sentry issue has none", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("456"))
		_, err = factoryModel.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceSentryExceptions)
		require.NoError(t, err)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "boom"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("merges a Dependabot alert into the open task for its package", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		packageRef := ghdependabot.PackageRef{Repository: "acme/payments", Ecosystem: "npm", Name: "lodash"}
		existing, err := factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Fix Dependabot alerts for lodash (npm)",
			"## Alerts\n\n### #7 Prototype pollution\nhttps://github.com/acme/payments/security/dependabot/7",
			nil,
			nil,
			nil,
			packageRef.Origin(),
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, dependabotPayload(8, "package-lock.json"))
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Fix Dependabot alerts for lodash (npm)"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))

		reloaded, err := factoryModel.FindWorkOrder(database.Conn(), existing.ID)
		require.NoError(t, err)
		assert.Contains(t, reloaded.Description, "### #8 Prototype pollution in lodash\nSeverity: high\nManifest: package-lock.json")
		assert.Contains(t, reloaded.Description, "https://github.com/acme/payments/security/dependabot/7\n\n### #8")
	})

	t.Run("opens a new Dependabot task when the package task is closed", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		packageRef := ghdependabot.PackageRef{Repository: "acme/payments", Ecosystem: "npm", Name: "lodash"}
		closed, err := factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Fix Dependabot alerts for lodash (npm)",
			"",
			nil,
			nil,
			nil,
			packageRef.Origin(),
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(closed).Update("state", models.FactoryWorkOrderStateClosed).Error)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, dependabotPayload(9, "package.json"))
		_, err = factoryModel.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceDependabotAlerts)
		require.NoError(t, err)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Fix Dependabot alerts for lodash (npm)"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))

		reloaded, err := factoryModel.FindWorkOrder(database.Conn(), uuid.MustParse(order.ID))
		require.NoError(t, err)
		require.NotNil(t, reloaded.OriginURL)
		assert.Equal(t, packageRef.OriginURL(), *reloaded.OriginURL)
		require.NotNil(t, reloaded.OriginLabel)
		assert.Equal(t, "Dependabot: lodash", *reloaded.OriginLabel)
	})

	t.Run("serializes concurrent creates for the same Sentry issue", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvasOne, executionOne, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("789"))
		canvasTwo, executionTwo, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, sentryPayload("789"))
		_, err = factoryModel.CreateIntake(database.Conn(), canvasOne.ID, models.FactoryIntakeSourceSentryExceptions)
		require.NoError(t, err)
		_, err = factoryModel.CreateIntake(database.Conn(), canvasTwo.ID, models.FactoryIntakeSourceSentryExceptions)
		require.NoError(t, err)

		start := make(chan struct{})
		var wg sync.WaitGroup
		created := make([]bool, 2)
		errs := make([]error, 2)
		canvases := []*models.Canvas{canvasOne, canvasTwo}
		executions := []*models.CanvasNodeExecution{executionOne, executionTwo}

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				errs[idx] = database.Conn().Transaction(func(tx *gorm.DB) error {
					ctx := NewFactoryContext(tx, canvases[idx], executions[idx])
					_, didCreate, createErr := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "boom"})
					created[idx] = didCreate
					return createErr
				})
			}(i)
		}

		close(start)
		wg.Wait()

		for _, createErr := range errs {
			require.NoError(t, createErr)
		}

		createdCount := 0
		for _, didCreate := range created {
			if didCreate {
				createdCount++
			}
		}
		assert.Equal(t, 1, createdCount)
		assert.Equal(t, 1, countOrders(factoryModel))
	})
}

func TestFactoryContext_CreateWorkOrder_SkipsDuplicateJiraIssue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	jiraPayloadOnSite := func(action, siteURL, issueKey string) map[string]any {
		return map[string]any{
			"type": "jira.issue",
			"data": map[string]any{
				"action": action,
				"url":    siteURL + "/browse/" + issueKey,
				"issue":  map[string]any{"key": issueKey},
			},
		}
	}
	jiraPayload := func(action, issueKey string) map[string]any {
		return jiraPayloadOnSite(action, "https://acme.atlassian.net", issueKey)
	}

	countOrders := func(factoryModel *models.Factory) int {
		t.Helper()
		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 100})
		require.NoError(t, err)
		return len(orders)
	}

	t.Run("create then update for the same issue inserts one work order", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		createdCanvas, createdExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayload("created", "ENG-9"))
		_, err = factoryModel.CreateIntake(database.Conn(), createdCanvas.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)
		createdCtx := NewFactoryContext(database.Conn(), createdCanvas, createdExecution)

		first, created, err := createdCtx.CreateWorkOrder(core.WorkOrderParams{Title: "New issue"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, first)
		require.NotNil(t, first.Origin)
		assert.Equal(t, "https://acme.atlassian.net/browse/ENG-9", first.Origin.URL)
		assert.Equal(t, "ENG-9", first.Origin.Label)

		persisted, err := factoryModel.FindWorkOrder(database.Conn(), uuid.MustParse(first.ID))
		require.NoError(t, err)
		beforeEvents, err := persisted.ListEvents(database.Conn(), 0, nil)
		require.NoError(t, err)
		require.NotEmpty(t, beforeEvents)

		updatedCanvas, updatedExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayload("updated", "ENG-9"))
		updatedCtx := NewFactoryContext(database.Conn(), updatedCanvas, updatedExecution)

		second, created, err := updatedCtx.CreateWorkOrder(core.WorkOrderParams{Title: "New issue"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, second)
		assert.Equal(t, 1, countOrders(factoryModel))

		afterEvents, err := persisted.ListEvents(database.Conn(), 0, nil)
		require.NoError(t, err)
		assert.Len(t, afterEvents, len(beforeEvents))
	})

	t.Run("creates a task when the same key exists on another Jira site", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Other site issue",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://other.atlassian.net/browse/ENG-5", Label: "ENG-5"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayloadOnSite("created", "https://acme.atlassian.net", "ENG-5"))
		_, err = factoryModel.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Acme ENG-5"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		require.NotNil(t, order.Origin)
		assert.Equal(t, "https://acme.atlassian.net/browse/ENG-5", order.Origin.URL)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("does not skip ENG-5 when ENG-50 already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Issue ENG-50",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://acme.atlassian.net/browse/ENG-50", Label: "ENG-50"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayload("created", "ENG-5"))
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Issue ENG-5"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("serializes concurrent creates for the same Jira issue", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvasOne, executionOne, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayload("created", "ENG-8"))
		canvasTwo, executionTwo, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, jiraPayload("updated", "ENG-8"))
		_, err = factoryModel.CreateIntake(database.Conn(), canvasOne.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)
		_, err = factoryModel.CreateIntake(database.Conn(), canvasTwo.ID, models.FactoryIntakeSourceJiraIssues)
		require.NoError(t, err)

		start := make(chan struct{})
		var wg sync.WaitGroup
		created := make([]bool, 2)
		errs := make([]error, 2)
		canvases := []*models.Canvas{canvasOne, canvasTwo}
		executions := []*models.CanvasNodeExecution{executionOne, executionTwo}

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				errs[idx] = database.Conn().Transaction(func(tx *gorm.DB) error {
					ctx := NewFactoryContext(tx, canvases[idx], executions[idx])
					_, didCreate, createErr := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "New issue"})
					created[idx] = didCreate
					return createErr
				})
			}(i)
		}

		close(start)
		wg.Wait()

		for _, createErr := range errs {
			require.NoError(t, createErr)
		}

		createdCount := 0
		for _, didCreate := range created {
			if didCreate {
				createdCount++
			}
		}
		assert.Equal(t, 1, createdCount)
		assert.Equal(t, 1, countOrders(factoryModel))
	})
}

func TestFactoryContext_CreateWorkOrder_SkipsDuplicateGitHubIssue(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	issuePayload := func(action, label string, number int) map[string]any {
		return map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"action": action,
				"label":  map[string]any{"name": label},
				"issue": map[string]any{
					"html_url": "https://github.com/acme/payments/issues/" + strconv.Itoa(number),
					"title":    "Handle duplicate refunds",
				},
			},
		}
	}

	countOrders := func(factoryModel *models.Factory) int {
		t.Helper()
		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 100})
		require.NoError(t, err)
		return len(orders)
	}

	createFromPayload := func(t *testing.T, factoryModel *models.Factory, payload map[string]any) (*core.WorkOrder, bool, *models.CanvasNodeExecution) {
		t.Helper()
		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, payload)
		_, err := factoryModel.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		order, created, err := NewFactoryContext(database.Conn(), canvas, nodeExecution).
			CreateWorkOrder(core.WorkOrderParams{Title: "Handle duplicate refunds", DefaultOwnerID: r.User.String()})
		require.NoError(t, err)
		return order, created, nodeExecution
	}

	t.Run("a second label on the same issue does not add a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		first, created, _ := createFromPayload(t, factoryModel, issuePayload("labeled", "superplane", 12))
		require.True(t, created)
		require.NotNil(t, first)
		persisted, err := factoryModel.FindWorkOrder(database.Conn(), uuid.MustParse(first.ID))
		require.NoError(t, err)
		require.Len(t, persisted.Assignees, 1)
		require.NoError(t, persisted.UpdateAssignees(database.Conn(), nil, r.User))

		second, created, duplicateExecution := createFromPayload(t, factoryModel, issuePayload("labeled", "benchmark", 12))
		assert.False(t, created)
		assert.Nil(t, second)
		require.NoError(t, database.Conn().First(duplicateExecution, "id = ?", duplicateExecution.ID).Error)
		assert.Equal(t, first.ID, duplicateExecution.Metadata.Data()["existing_task_id"])
		assert.Equal(t, 1, countOrders(factoryModel))
		persisted, err = factoryModel.FindWorkOrder(database.Conn(), persisted.ID)
		require.NoError(t, err)
		assert.Empty(t, persisted.Assignees, "a duplicate must not reapply the default after an explicit clear")
	})

	t.Run("a closed task does not stop a new task for the issue", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		closed, err := factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Earlier task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://github.com/acme/payments/issues/12", Label: "acme/payments#12"},
		)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(closed).Update("state", models.FactoryWorkOrderStateClosed).Error)

		order, created, _ := createFromPayload(t, factoryModel, issuePayload("reopened", "", 12))
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("issue 5 is not skipped when issue 50 already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Issue 50",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://github.com/acme/payments/issues/50", Label: "acme/payments#50"},
		)
		require.NoError(t, err)

		order, created, _ := createFromPayload(t, factoryModel, issuePayload("opened", "", 5))
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("serializes concurrent label events for the same issue", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		canvasOne, executionOne, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, issuePayload("labeled", "superplane", 8))
		canvasTwo, executionTwo, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, issuePayload("labeled", "benchmark", 8))
		_, err = factoryModel.CreateIntake(database.Conn(), canvasOne.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		_, err = factoryModel.CreateIntake(database.Conn(), canvasTwo.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, 2)
		canvases := []*models.Canvas{canvasOne, canvasTwo}
		executions := []*models.CanvasNodeExecution{executionOne, executionTwo}

		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				<-start
				errs[idx] = database.Conn().Transaction(func(tx *gorm.DB) error {
					ctx := NewFactoryContext(tx, canvases[idx], executions[idx])
					_, _, createErr := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "New issue"})
					return createErr
				})
			}(i)
		}

		close(start)
		wg.Wait()

		for _, createErr := range errs {
			require.NoError(t, createErr)
		}
		assert.Equal(t, 1, countOrders(factoryModel))
	})
}

func TestFactoryContext_CreateWorkOrder_SkipsDuplicateProductiveTask(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	productivePayload := func(taskID string, pageURL string) map[string]any {
		envelope := map[string]any{
			"type": productive.TaskPayloadType,
			"data": map[string]any{
				"meta": map[string]any{"event": productive.TaskUpdatedEvent},
				"data": map[string]any{"id": taskID, "type": "tasks"},
			},
		}
		if pageURL != "" {
			data, ok := envelope["data"].(map[string]any)
			require.True(t, ok)
			data["url"] = pageURL
		}
		return envelope
	}

	countOrders := func(factoryModel *models.Factory) int {
		t.Helper()
		orders, err := factoryModel.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 100})
		require.NoError(t, err)
		return len(orders)
	}

	t.Run("skips a Productive task that already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Existing productive task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://app.productive.io/12345/tasks/91", Label: "91"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/12345/tasks/91"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Existing productive task"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("skips when the existing task has no origin URL", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, _, sourceRun := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", ""),
		)
		_, err = factoryModel.CreateWorkOrder(database.Conn(), "Existing productive task", "", nil, nil, &sourceRun.ID)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/12345/tasks/91"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Moved onto Bugs"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("does not skip task 9 when task 91 already has a task", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Task 91",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://app.productive.io/12345/tasks/91", Label: "91"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("9", "https://app.productive.io/12345/tasks/9"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Task 9"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("creates a task when the same id belongs to another organization", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Org A task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://app.productive.io/111/tasks/91", Label: "91"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/222/tasks/91"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Org B task"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("skips a slug URL for the same organization", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		_, err = factoryModel.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Existing productive task",
			"",
			nil,
			nil,
			nil,
			models.WorkOrderOrigin{URL: "https://app.productive.io/12345-acme/tasks/91", Label: "91"},
		)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/12345/tasks/91"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Same organization"})
		require.NoError(t, err)
		assert.False(t, created)
		assert.Nil(t, order)
		assert.Equal(t, 1, countOrders(factoryModel))
	})

	t.Run("creates a task when an origin-less task belongs to another organization", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, _, sourceRun := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/111/tasks/91"),
		)
		_, err = factoryModel.CreateWorkOrder(database.Conn(), "Org A task", "", nil, nil, &sourceRun.ID)
		require.NoError(t, err)

		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("91", "https://app.productive.io/222/tasks/91"),
		)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Org B task"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		assert.Equal(t, 2, countOrders(factoryModel))
	})

	t.Run("creates a task when the Productive task has none", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		pageURL := "https://app.productive.io/12345/tasks/77"
		canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(
			t,
			r,
			factoryModel.ID,
			productivePayload("77", pageURL),
		)
		_, err = factoryModel.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceProductiveTasks)
		require.NoError(t, err)
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

		order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "New productive task"})
		require.NoError(t, err)
		require.True(t, created)
		require.NotNil(t, order)
		require.NotNil(t, order.Origin)
		assert.Equal(t, pageURL, order.Origin.URL)
		assert.Equal(t, 1, countOrders(factoryModel))
	})
}

func TestFactoryContext_UpdateWorkOrderStatus(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)

	// `order.status.updated` is now the sole authoritative event. The
	// draft → open transition carries the automation ref (node/line/step).
	statusEvent := findWorkOrderEvent(t, order, "order.status.updated")
	statusAutomation := extractAutomationPayload(t, statusEvent)
	assert.Equal(t, nodeExecution.NodeID, statusAutomation.NodeID)
	assert.Equal(t, line.Name, statusAutomation.LineName)
	assert.Equal(t, "component-under-test", statusAutomation.StepName)
}

// A re-run of the component with the same target state must be a
// silent no-op: no duplicate `order.status.updated` event, no
// websocket fan-out, and the caller sees `changed=false`.
func TestFactoryContext_UpdateWorkOrderStatus_NoopSkipsEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	var notifications int
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, _ string) { notifications++ })

	_, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, 1, notifications)

	_, changed, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.False(t, changed, "re-hitting the same state must report changed=false")
	assert.Equal(t, 1, notifications, "no-op transition must not fan out again")

	statusEvents := listWorkOrderEvents(t, order, factoryevents.EventTypeOrderStatusUpdated)
	// Creation ("" → draft) + one real draft → open transition = 2.
	// A second event would indicate the no-op leaked into the timeline.
	assert.Len(t, statusEvents, 2, "no-op must not record a second status.updated event")
}

func TestFactoryContext_UpdateWorkOrderStatus_CloseReportsRunCancellations(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	agent, err := models.CreateCanvasRunInTransaction(database.Conn(), canvas.ID, "create-work-order", models.CanvasRunStateStarted, "")
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(agent).Update("input", models.NewJSONValue(map[string]any{
		"work_order": map[string]any{"id": order.ID.String()},
	})).Error)

	var reported []uuid.UUID
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).WithRunsCancelled(func(results []*models.RunCancellationResult) {
		for _, result := range results {
			if result == nil || result.Run == nil || result.Drain == nil {
				continue
			}
			reported = append(reported, result.Run.ID)
		}
	})
	_, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultRejected,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, []uuid.UUID{agent.ID}, reported)
	assert.NotContains(t, reported, run.ID)

	agent, err = models.FindUnscopedCanvasRun(database.Conn(), agent.ID)
	require.NoError(t, err)
	assert.Equal(t, models.CanvasRunStateCancelling, agent.State)
}

func TestFactoryContext_UpdateWorkOrderStatus_CloseAttributesAutomation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Status target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	_, err = order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultCompleted,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateClosed, updated.State)

	// The most recent status.updated (open → closed) is the close event.
	closed := findWorkOrderEvent(t, order, "order.status.updated")
	closedAutomation := extractAutomationPayload(t, closed)
	assert.Equal(t, line.Name, closedAutomation.LineName)
	assert.Equal(t, "component-under-test", closedAutomation.StepName)
	assert.Equal(t, nodeExecution.NodeID, closedAutomation.NodeID)
}

func TestFactoryContext_UpdateWorkOrderStatus_RejectedStaysVisible(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factoryModel.ID)
	order, err := factoryModel.CreateWorkOrder(database.Conn(), "Automation rejection", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultRejected,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, updated.Result)

	reloaded, err := factoryModel.FindWorkOrder(database.Conn(), order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderResultManualRejected, reloaded.Result)
	assert.False(t, reloaded.IsArchived())
}

// orderId is required and explicit at the FactoryContext level too — there
// is no implicit fallback to "the work order driving the current run".
// That behavior now lives entirely in the `{{ order().id }}` default on the
// component field, resolved before Execute ever sees the configuration.
func TestFactoryContext_UpdateWorkOrderStatus_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	_, _, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		State: models.FactoryWorkOrderStateOpen,
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

func TestFactoryContext_AddWorkOrderComment_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	err = ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		Body: "Ready for review",
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

func TestFactoryContext_AddWorkOrderArtifact_EmptyOrderIDIsRejected(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)
	_, err = ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		Type: "link",
		Data: map[string]any{"url": "https://github.com/example/repo/pull/1"},
	})
	require.Error(t, err)
	assert.EqualError(t, err, "orderId is required")
}

// Regression coverage for the github.onPullRequest -> close-work-order bug:
// a run with no `factory_work_order_executions` row (e.g. one started by a
// plain webhook trigger, not a factory line dispatch) must still be able to
// target a work order explicitly via `orderId`.
func TestFactoryContext_UpdateWorkOrderStatus_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	updated, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: order.ID.String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, updated.State)

	// Attribution still includes node/app but omits line/step (no
	// dispatch happened) and must not panic.
	statusEvent := findWorkOrderEvent(t, order, "order.status.updated")
	statusAutomation := extractAutomationPayload(t, statusEvent)
	assert.Equal(t, nodeExecution.NodeID, statusAutomation.NodeID)
	assert.NotEmpty(t, statusAutomation.AppName)
	assert.Empty(t, statusAutomation.LineName)
	assert.Empty(t, statusAutomation.StepName)
}

func TestFactoryContext_UpdateWorkOrderStatus_ExplicitOrderIDNotFound(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	_, _, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
		OrderID: uuid.New().String(),
		State:   models.FactoryWorkOrderStateOpen,
	})
	require.Error(t, err)
}

func TestFactoryContext_AddWorkOrderComment_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached comment target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: order.ID.String(),
		Body:    "Merged via PR",
	}))

	commentEvent := findWorkOrderEvent(t, order, "order.comment.added")
	assert.NotNil(t, commentEvent)
}

func TestFactoryContext_AddWorkOrderArtifact_ExplicitOrderIDOnUnattachedRun(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Unattached artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url": "https://github.com/example/repo/pull/7",
		},
		Key: "https://github.com/example/repo/pull/7",
	})
	require.NoError(t, err)
	require.NotNil(t, artifact)
	assert.Equal(t, order.ID.String(), artifact.WorkOrderID)
}

func TestFactoryContext_ReportWorkOrderCheck(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Check target", "", &r.User, nil, nil)
	require.NoError(t, err)

	notifications := 0
	lastReason := ""
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, reason string) {
			notifications++
			lastReason = reason
		})

	check, err := ctx.ReportWorkOrderCheck(core.ReportWorkOrderCheckParams{
		OrderID:  order.ID.String(),
		CheckKey: "risk-review",
		Name:     "Risk review",
		Score:    7,
		MaxScore: 10,
		Level:    models.FactoryWorkOrderCheckLevelCaution,
	})
	require.NoError(t, err)
	assert.Equal(t, order.ID.String(), check.WorkOrderID)
	assert.Nil(t, check.PreviousScore)
	assert.Equal(t, 1, notifications)
	assert.Equal(t, factoryevents.EventTypeOrderCheckReported, lastReason)

	// A re-report of the same key updates in place and keeps the prior score.
	updated, err := ctx.ReportWorkOrderCheck(core.ReportWorkOrderCheckParams{
		OrderID:  order.ID.String(),
		CheckKey: "risk-review",
		Name:     "Risk review",
		Score:    3,
		MaxScore: 10,
		Level:    models.FactoryWorkOrderCheckLevelPositive,
	})
	require.NoError(t, err)
	assert.Equal(t, check.ID, updated.ID)
	require.NotNil(t, updated.PreviousScore)
	assert.Equal(t, float64(7), *updated.PreviousScore)
	assert.Equal(t, 2, notifications)

	// The timeline event carries automation + run attribution.
	event := findWorkOrderEvent(t, order, factoryevents.EventTypeOrderCheckReported)
	var payload factoryevents.WorkOrderCheckReported
	require.NoError(t, json.Unmarshal(event.Data, &payload))
	require.NotNil(t, payload.Automation)
	assert.Equal(t, canvas.ID, payload.Automation.AppID)
	require.NotNil(t, payload.Run)
	assert.Equal(t, nodeExecution.RunID, payload.Run.ID)
}

func TestFactoryContext_FindWorkOrder_ByID(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Find by id target", "", &r.User, nil, nil)
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	t.Run("finds the order", func(t *testing.T) {
		found, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: order.ID.String()})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), found.ID)
	})

	t.Run("returns ErrWorkOrderNotFound for an unknown id", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: uuid.New().String()})
		assert.ErrorIs(t, err, core.ErrWorkOrderNotFound)
	})

	t.Run("returns a plain error for an invalid uuid", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: "not-a-uuid"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, core.ErrWorkOrderNotFound))
	})

	t.Run("rejects a non-factory app", func(t *testing.T) {
		regularCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, nil, nil)
		regularCtx := NewFactoryContext(database.Conn(), regularCanvas, nodeExecution)

		_, err := regularCtx.FindWorkOrder(core.FindWorkOrderParams{By: "id", OrderID: order.ID.String()})
		require.Error(t, err)
		assert.EqualError(t, err, "app is not owned by a factory")
	})
}

func TestFactoryContext_FindWorkOrder_ByArtifactKey(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Find by artifact key target", "", &r.User, nil, nil)
	require.NoError(t, err)

	_, err = order.CreateArtifact(database.Conn(), models.FactoryWorkOrderArtifactParams{
		Type: "link",
		Data: map[string]any{"url": "https://github.com/example/repo/pull/99"},
		Key:  "https://github.com/example/repo/pull/99",
	})
	require.NoError(t, err)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	t.Run("finds the order", func(t *testing.T) {
		found, err := ctx.FindWorkOrder(core.FindWorkOrderParams{
			By:          "artifactKey",
			ArtifactKey: "https://github.com/example/repo/pull/99",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), found.ID)
	})

	t.Run("returns ErrWorkOrderNotFound for an unknown key", func(t *testing.T) {
		_, err := ctx.FindWorkOrder(core.FindWorkOrderParams{By: "artifactKey", ArtifactKey: "no-such-key"})
		assert.ErrorIs(t, err, core.ErrWorkOrderNotFound)
	})
}

func TestFactoryContext_AddPullRequest_NotifiesGitHubRecorded(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(database.Conn(), "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	var recordedOrg, recordedFactory, recordedPR uuid.UUID
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithGitHubPullRequestRecorded(func(organizationID, factoryID, pullRequestID uuid.UUID) {
			recordedOrg = organizationID
			recordedFactory = factoryID
			recordedPR = pullRequestID
		})

	created, err := ctx.AddPullRequest(core.AddPullRequestParams{
		OrderID:    order.ID.String(),
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: "acme/app",
		Number:     44,
		URL:        "https://github.com/acme/app/pull/44",
		Title:      "Ready",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	assert.Equal(t, factory.OrganizationID, recordedOrg)
	assert.Equal(t, factory.ID, recordedFactory)
	assert.Equal(t, created.ID, recordedPR.String())

	recordedPR = uuid.Nil
	_, err = ctx.AddPullRequest(core.AddPullRequestParams{
		OrderID:    order.ID.String(),
		Provider:   models.FactoryPullRequestProviderBitbucket,
		Repository: "acme/app",
		Number:     45,
		URL:        "https://bitbucket.org/acme/app/pull-requests/45",
		State:      models.FactoryPullRequestStateOpen,
	})
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, recordedPR)
}

func TestFactoryContext_FindPullRequest_IncludesWorkOrderOrigin(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factory.ID)
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	t.Run("includes origin when the task has one", func(t *testing.T) {
		originURL := "https://github.com/acme/payments/issues/12"
		originLabel := "acme/payments#12"
		order, err := factory.CreateWorkOrderWithOrigin(
			database.Conn(),
			"Close origin after merge",
			"",
			&r.User,
			nil,
			nil,
			models.WorkOrderOrigin{URL: originURL, Label: originLabel},
		)
		require.NoError(t, err)

		pullRequest, err := order.CreatePullRequest(database.Conn(), models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/payments",
			Number:     42,
			URL:        "https://github.com/acme/payments/pull/42",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		match, err := ctx.FindPullRequest(core.FindPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/payments",
			Number:     42,
		})
		require.NoError(t, err)
		require.NotNil(t, match.PullRequest)
		assert.Equal(t, pullRequest.ID.String(), match.PullRequest.ID)
		require.NotNil(t, match.WorkOrder)
		require.NotNil(t, match.WorkOrder.Origin)
		assert.Equal(t, originURL, match.WorkOrder.Origin.URL)
		assert.Equal(t, originLabel, match.WorkOrder.Origin.Label)
		assert.Equal(t, uiBaseURL()+order.URLPath(factory.RouteSegment()), match.WorkOrder.URL)
	})

	t.Run("omits origin when the task has none", func(t *testing.T) {
		order, err := factory.CreateWorkOrder(database.Conn(), "No origin", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = order.CreatePullRequest(database.Conn(), models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/payments",
			Number:     43,
			URL:        "https://github.com/acme/payments/pull/43",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)

		match, err := ctx.FindPullRequest(core.FindPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/payments",
			Number:     43,
		})
		require.NoError(t, err)
		require.NotNil(t, match.WorkOrder)
		assert.Nil(t, match.WorkOrder.Origin)
	})
}

func TestFactoryContext_AddWorkOrderComment(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Comment target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: order.ID.String(),
		Body:    "Ready for review",
	}))

	commentEvent := findWorkOrderEvent(t, order, "order.comment.added")

	var payload struct {
		Author struct {
			Kind       string `json:"kind"`
			Automation struct {
				NodeID   string `json:"nodeId"`
				AppName  string `json:"appName"`
				LineName string `json:"lineName"`
				StepName string `json:"stepName"`
			} `json:"automation"`
		} `json:"author"`
	}
	require.NoError(t, json.Unmarshal(commentEvent.Data, &payload))
	assert.Equal(t, "automation", payload.Author.Kind)
	assert.Equal(t, nodeExecution.NodeID, payload.Author.Automation.NodeID)
	assert.NotEmpty(t, payload.Author.Automation.AppName)
	assert.Equal(t, line.Name, payload.Author.Automation.LineName)
	assert.Equal(t, "component-under-test", payload.Author.Automation.StepName)
}

func TestFactoryContext_AddWorkOrderComment_EmitsNotification(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Comment target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	var notifications []messages.FactoryWorkOrderNotificationMessage
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderNotification(func(notification messages.FactoryWorkOrderNotificationMessage) {
			notifications = append(notifications, notification)
		})

	require.NoError(t, ctx.AddWorkOrderComment(core.AddWorkOrderCommentParams{
		OrderID: order.ID.String(),
		Body:    "Ready for review",
	}))

	assert.Empty(t, notifications)
}

func TestFactoryContext_SetWorkOrderStatusNote_EmitsNotification(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Note target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	_, err = order.UpdateStatus(database.Conn(), models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	var notifications []messages.FactoryWorkOrderNotificationMessage
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderNotification(func(notification messages.FactoryWorkOrderNotificationMessage) {
			notifications = append(notifications, notification)
		})

	_, err = ctx.SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams{
		OrderID:  order.ID.String(),
		NoteKey:  "pr-closure",
		Headline: "Review the pull request",
		Body:     "Merging the PR completes this work order automatically.",
		CtaLabel: "Review PR #42",
		CtaURL:   "https://github.com/example/repo/pull/42",
	})
	require.NoError(t, err)

	require.Len(t, notifications, 1)
	assert.Equal(t, factory.ID.String(), notifications[0].FactoryID)
	assert.Equal(t, order.ID.String(), notifications[0].OrderID)
	assert.Equal(t, factoryevents.EventTypeOrderStatusNoteUpdated, notifications[0].EventType)
	assert.Equal(t, "Review the pull request", notifications[0].StatusNoteHeadline)
	assert.Equal(t, "Merging the PR completes this work order automatically.", notifications[0].StatusNoteBody)
	assert.Equal(t, "Review PR #42", notifications[0].StatusNoteCtaLabel)
	assert.Equal(t, "https://github.com/example/repo/pull/42", notifications[0].StatusNoteCtaURL)
	assert.NotEmpty(t, notifications[0].ActorName)

	// Re-setting the same note key sends a fresh notification: a status
	// note update is new information the reviewer should see, same as a
	// second comment firing its own email.
	_, err = ctx.SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams{
		OrderID:  order.ID.String(),
		NoteKey:  "pr-closure",
		Headline: "Still waiting on review",
	})
	require.NoError(t, err)
	require.Len(t, notifications, 2)
	assert.Equal(t, "Still waiting on review", notifications[1].StatusNoteHeadline)
}

func TestFactoryContext_AddWorkOrderArtifact(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)
	line := linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution)

	artifact, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url":   "https://github.com/example/repo/pull/1",
			"title": "Draft",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, artifact)
	assert.Equal(t, "link", artifact.Type)
	assert.Equal(t, "https://github.com/example/repo/pull/1", artifact.Data["url"])

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	artifactEvent := findWorkOrderEvent(t, order, "order.artifact.added")
	artifactAutomation := extractAutomationPayload(t, artifactEvent)
	assert.Equal(t, nodeExecution.NodeID, artifactAutomation.NodeID)
	assert.Equal(t, line.Name, artifactAutomation.LineName)
	assert.Equal(t, "component-under-test", artifactAutomation.StepName)
}

func TestFactoryContext_AddWorkOrderArtifact_KeyedRefresh(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Keyed artifact target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	var reasons []string
	var notifications []messages.FactoryWorkOrderNotificationMessage
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, reason string) {
			reasons = append(reasons, reason)
		}).
		WithWorkOrderNotification(func(notification messages.FactoryWorkOrderNotificationMessage) {
			notifications = append(notifications, notification)
		})

	first, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url":   "https://preview.example.com/v1",
			"title": "Preview",
		},
		Key: "storybook-preview",
	})
	require.NoError(t, err)

	second, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url": "https://preview.example.com/v2",
		},
		Key: "storybook-preview",
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "https://preview.example.com/v2", second.Data["url"])
	_, hasTitle := second.Data["title"]
	assert.False(t, hasTitle)

	assert.Equal(t, []string{
		factoryevents.EventTypeOrderArtifactAdded,
		factoryevents.EventTypeOrderArtifactUpdated,
	}, reasons)
	assert.Empty(t, notifications)

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)

	events, err := order.ListEvents(database.Conn(), 50, nil)
	require.NoError(t, err)
	addedCount := 0
	for _, event := range events {
		if event.Type == factoryevents.EventTypeOrderArtifactAdded {
			addedCount++
		}
		assert.NotEqual(t, factoryevents.EventTypeOrderArtifactUpdated, event.Type)
	}
	assert.Equal(t, 1, addedCount)
}

func TestFactoryContext_AddWorkOrderArtifact_UpdateByTitle(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, nodeExecution, run := setupFactoryAppExecution(t, r, factory.ID)
	order, err := factory.CreateWorkOrder(database.Conn(), "Title update target", "", &r.User, nil, nil)
	require.NoError(t, err)
	linkRunToWorkOrder(t, r, factory, order.ID, run.ID)

	other, err := factory.CreateWorkOrder(database.Conn(), "Other title target", "", &r.User, nil, nil)
	require.NoError(t, err)
	otherArtifact, err := other.CreateArtifact(database.Conn(), models.FactoryWorkOrderArtifactParams{
		Type: models.FactoryWorkOrderArtifactTypeLink,
		Data: map[string]any{
			"url":   "https://preview.example.com/other",
			"title": "Preview",
		},
	})
	require.NoError(t, err)

	var reasons []string
	ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).
		WithWorkOrderUpdated(func(_, _, reason string) {
			reasons = append(reasons, reason)
		})

	first, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url":   "https://preview.example.com/v1",
			"title": "Preview",
			"note":  "first",
		},
		Key:           "storybook-preview",
		UpdateByTitle: true,
	})
	require.NoError(t, err)

	second, err := ctx.AddWorkOrderArtifact(core.AddWorkOrderArtifactParams{
		OrderID: order.ID.String(),
		Type:    "link",
		Data: map[string]any{
			"url":   "https://preview.example.com/v2",
			"title": "Preview",
		},
		Key:           "storybook-preview",
		UpdateByTitle: true,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "https://preview.example.com/v2", second.Data["url"])
	_, hasNote := second.Data["note"]
	assert.False(t, hasNote)

	assert.Equal(t, []string{
		factoryevents.EventTypeOrderArtifactAdded,
		factoryevents.EventTypeOrderArtifactUpdated,
	}, reasons)

	artifacts, err := order.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, artifacts, 1)
	assert.Nil(t, artifacts[0].Key)

	otherArtifacts, err := other.ListArtifacts(database.Conn())
	require.NoError(t, err)
	require.Len(t, otherArtifacts, 1)
	assert.Equal(t, otherArtifact.ID, otherArtifacts[0].ID)
	assert.Equal(t, "https://preview.example.com/other", mustContextArtifactData(t, &otherArtifacts[0])["url"])
}

func mustContextArtifactData(t *testing.T, artifact *models.FactoryWorkOrderArtifact) map[string]any {
	t.Helper()
	var data map[string]any
	require.NoError(t, json.Unmarshal(artifact.Data, &data))
	return data
}

type automationPayload struct {
	NodeID   string `json:"nodeId"`
	NodeName string `json:"nodeName"`
	AppName  string `json:"appName"`
	LineName string `json:"lineName"`
	StepName string `json:"stepName"`
}

func extractAutomationPayload(t *testing.T, event *models.FactoryWorkOrderEvent) automationPayload {
	t.Helper()

	var wrapper struct {
		Automation *automationPayload `json:"automation"`
	}
	require.NoError(t, json.Unmarshal(event.Data, &wrapper))
	require.NotNil(t, wrapper.Automation, "event %s missing automation payload", event.Type)
	return *wrapper.Automation
}

func findWorkOrderEvent(t *testing.T, order *models.FactoryWorkOrder, eventType string) *models.FactoryWorkOrderEvent {
	t.Helper()

	events, err := order.ListEvents(database.Conn(), 50, nil)
	require.NoError(t, err)

	// ListEvents sorts DESC by created_at, so the first match is the latest.
	for i := range events {
		if events[i].Type == eventType {
			return &events[i]
		}
	}

	t.Fatalf("expected %s event on work order %s", eventType, order.ID)
	return nil
}

// listWorkOrderEvents returns every event of the given type, newest first.
func listWorkOrderEvents(t *testing.T, order *models.FactoryWorkOrder, eventType string) []models.FactoryWorkOrderEvent {
	t.Helper()

	events, err := order.ListEvents(database.Conn(), 100, nil)
	require.NoError(t, err)

	matches := make([]models.FactoryWorkOrderEvent, 0)
	for _, event := range events {
		if event.Type == eventType {
			matches = append(matches, event)
		}
	}
	return matches
}

// linkRunToWorkOrder creates the `factory_work_order_executions` row
// FactoryContext looks up from `execution.RunID`; the returned line is
// used by tests that assert line-based automation attribution.
func linkRunToWorkOrder(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
	workOrderID uuid.UUID,
	runID uuid.UUID,
) *models.FactoryLine {
	t.Helper()

	line, err := factory.CreateLine(database.Conn(), support.RandomName("line"), nil)
	require.NoError(t, err)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, workOrderID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    workOrderID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "component-under-test",
		RunID:          &runID,
		Status:         models.FactoryWorkOrderExecutionStatusRunning,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, database.Conn().Create(&execution).Error)
	return line
}

func TestFactoryContext_CreateWorkOrderIngestsGitHubImagesBeforeEmit(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	onWorkOrderCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "on-work-order",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName},
			}),
		}},
		nil,
	)
	require.NoError(t, db.Model(onWorkOrderCanvas).Update("factory_id", factoryModel.ID).Error)

	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factoryModel.ID)
	imageURL := "https://user-images.githubusercontent.com/1/ok.png"
	description := "See ![bug](" + imageURL + ")"

	ctx := NewFactoryContext(db, canvas, nodeExecution).WithRemoteImageFetch(
		func(context.Context, *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(bytes.NewReader([]byte("png-bytes"))),
			}, nil
		},
	)
	created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{
		Title:       "From GitHub issue",
		Description: description,
	})
	require.True(t, inserted)
	require.NoError(t, err)

	persisted, err := factoryModel.FindWorkOrder(db, uuid.MustParse(created.ID))
	require.NoError(t, err)
	assert.Contains(t, persisted.Description, blob.FileRefScheme+"://")
	assert.NotContains(t, persisted.Description, imageURL)

	files, err := models.ListReadyTaskFiles(db, persisted.ID)
	require.NoError(t, err)
	require.Len(t, files, 1)
	reader, err := store.Get(t.Context(), files[0].StorageKey)
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close() })
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, []byte("png-bytes"), body)

	events, err := models.ListCanvasEvents(db, onWorkOrderCanvas.ID, "on-work-order", 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload := onWorkOrderEventWorkOrder(t, events[0])
	assert.Equal(t, persisted.Description, payload["description"])
	assert.Contains(t, payload["description"], blob.FileRef(files[0].ID))
	listed, ok := payload["files"].([]any)
	require.True(t, ok)
	require.Len(t, listed, 1)
	item, ok := listed[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, files[0].ID.String(), item["id"])
	url, ok := item["url"].(string)
	require.True(t, ok)
	assert.Contains(t, url, "/api/v1/public/files/"+files[0].ID.String())
}

func TestFactoryContext_CreateWorkOrderStoresProductiveAttachments(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	onWorkOrderCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "on-work-order",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName},
			}),
		}},
		nil,
	)
	require.NoError(t, db.Model(onWorkOrderCanvas).Update("factory_id", factoryModel.ID).Error)

	inline := "https://files.productive.io/attachments/files/1/original/shot.png"
	canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, map[string]any{
		"type": productive.TaskPayloadType,
		"data": map[string]any{
			"meta": map[string]any{"event": productive.TaskCreatedEvent},
			"data": map[string]any{"id": "20305431", "type": "tasks"},
		},
	})

	ctx := NewFactoryContext(db, canvas, nodeExecution).WithProductiveTaskFiles(
		func(context.Context, string, string) ([]productive.TaskFile, error) {
			return []productive.TaskFile{
				{
					Name:        "shot.png",
					ContentType: "image/png",
					Body:        []byte("png-bytes"),
					ReplaceURLs: []string{inline},
				},
				{
					Name:        "notes.pdf",
					ContentType: "application/pdf",
					Body:        []byte("pdf-bytes"),
				},
			}, nil
		},
	)
	created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{
		Title:       "From Productive.io task",
		Description: "See ![shot](" + inline + ")",
	})
	require.True(t, inserted)
	require.NoError(t, err)

	persisted, err := factoryModel.FindWorkOrder(db, uuid.MustParse(created.ID))
	require.NoError(t, err)
	assert.NotContains(t, persisted.Description, inline)
	assert.Contains(t, persisted.Description, "![shot]("+blob.FileRefScheme+"://")
	assert.Contains(t, persisted.Description, "[notes.pdf]("+blob.FileRefScheme+"://")

	files, err := models.ListReadyTaskFiles(db, persisted.ID)
	require.NoError(t, err)
	require.Len(t, files, 2)

	events, err := models.ListCanvasEvents(db, onWorkOrderCanvas.ID, "on-work-order", 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload := onWorkOrderEventWorkOrder(t, events[0])
	assert.Equal(t, persisted.Description, payload["description"])
	listed, ok := payload["files"].([]any)
	require.True(t, ok)
	require.Len(t, listed, 2)
}

func TestFactoryContext_CreateWorkOrderIngestsJiraIssueFiles(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	onWorkOrderCanvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{
			NodeID: "on-work-order",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: factorycomp.OnWorkOrderTriggerName},
			}),
		}},
		nil,
	)
	require.NoError(t, db.Model(onWorkOrderCanvas).Update("factory_id", factoryModel.ID).Error)

	inline := testProxyURL("/rest/api/3/attachment/content/10001")
	canvas, nodeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factoryModel.ID, map[string]any{
		"type": jira.IssueEventPayloadType,
		"data": map[string]any{
			"url":   "https://acme.atlassian.net/browse/ENG-5",
			"issue": map[string]any{"key": "ENG-5"},
		},
	})

	ctx := NewFactoryContext(db, canvas, nodeExecution).WithJiraIssueFiles(
		func(context.Context, string, string) ([]jira.IssueFile, error) {
			return []jira.IssueFile{
				{
					Name:        "shot.png",
					ContentType: "image/png",
					Body:        []byte("png-bytes"),
					ReplaceURLs: []string{inline},
				},
				{
					Name:        "notes.pdf",
					ContentType: "application/pdf",
					Body:        []byte("pdf-bytes"),
				},
			}, nil
		},
	)
	created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{
		Title:       "From Jira issue",
		Description: "See ![shot](" + inline + ")",
	})
	require.True(t, inserted)
	require.NoError(t, err)

	persisted, err := factoryModel.FindWorkOrder(db, uuid.MustParse(created.ID))
	require.NoError(t, err)
	assert.NotContains(t, persisted.Description, inline)
	assert.Contains(t, persisted.Description, "![shot]("+blob.FileRefScheme+"://")
	assert.Contains(t, persisted.Description, "[notes.pdf]("+blob.FileRefScheme+"://")

	files, err := models.ListReadyTaskFiles(db, persisted.ID)
	require.NoError(t, err)
	require.Len(t, files, 2)

	events, err := models.ListCanvasEvents(db, onWorkOrderCanvas.ID, "on-work-order", 10, nil)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload := onWorkOrderEventWorkOrder(t, events[0])
	assert.Equal(t, persisted.Description, payload["description"])
	listed, ok := payload["files"].([]any)
	require.True(t, ok)
	require.Len(t, listed, 2)
}

func testProxyURL(path string) string {
	return jira.APIProxyHost + "/" + "35273b54-3f06-40d2-880f-dd28cf6daafa" + path
}

func TestFactoryContext_CreateWorkOrderDefersFileCleanupUntilCallerApplies(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	t.Setenv("BLOB_STORAGE_SIGNING_KEY", "test-signing-key")
	t.Setenv("BASE_URL", "http://files.test")
	store, err := filesystem.New(t.TempDir())
	require.NoError(t, err)
	blob.SetCurrent(store)
	t.Cleanup(func() { blob.SetCurrent(nil) })

	db := database.Conn()
	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factoryModel.ID)

	file, err := models.CreatePendingFile(db, models.CreateFileParams{
		Scope:          blob.ScopeWorkspace,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryModel.ID,
		Filename:       "bug.png",
		ContentType:    "image/png",
		CreatedByID:    r.User,
	})
	require.NoError(t, err)
	require.NoError(t, storedfiles.CompleteUpload(t.Context(), db, store, file, bytes.NewReader([]byte("png-bytes"))))
	loaded, err := models.FindFile(db, file.ID)
	require.NoError(t, err)
	sourceKey := loaded.StorageKey

	var jobs []FileBindCleanup
	ctx := NewFactoryContext(db, canvas, nodeExecution).WithFileBindCleanup(func(job FileBindCleanup) {
		jobs = append(jobs, job)
	})
	_, _, err = ctx.CreateWorkOrder(core.WorkOrderParams{
		Title:       "From workspace file",
		Description: "See ![bug](" + blob.FileRef(file.ID) + ")",
	})
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	_, err = store.Head(t.Context(), sourceKey)
	require.NoError(t, err)

	ApplyFileBindCleanups(jobs, nil)
	_, err = store.Head(t.Context(), sourceKey)
	assert.ErrorIs(t, err, blob.ErrNotFound)
}

func onWorkOrderEventWorkOrder(t *testing.T, event models.CanvasEvent) map[string]any {
	t.Helper()
	raw, err := json.Marshal(event.Data.Data())
	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(raw, &envelope))
	data, ok := envelope["data"].(map[string]any)
	require.True(t, ok, "event data: %s", raw)
	workOrder, ok := data["workOrder"].(map[string]any)
	require.True(t, ok, "workOrder payload: %s", raw)
	return workOrder
}

func TestFactoryContext_ProductAnalytics(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	factoryModel, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	canvas, nodeExecution, _ := setupFactoryAppExecution(t, r, factoryModel.ID)

	t.Run("automation creation uses a system identity", func(t *testing.T) {
		events := []productanalytics.Event{}
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).WithProductAnalytics(func(event productanalytics.Event) {
			events = append(events, event)
		})
		_, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{
			Title:       "From automation",
			Description: "Do not send this prompt",
		})
		require.NoError(t, err)
		require.True(t, created)
		require.Len(t, events, 1)
		assert.Equal(t, productanalytics.EventFactoryTaskCreated, events[0].Name)
		assert.Equal(t, "automation", events[0].Properties["creation_source"])
		assert.Equal(t, productanalytics.ActorSystem, events[0].ActorType)
		assert.Equal(t, productanalytics.SourceAutomation, events[0].Source)
		assert.NotEqual(t, r.Account.ID.String(), events[0].DistinctID)
		encoded, err := json.Marshal(events[0])
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "From automation")
		assert.NotContains(t, string(encoded), "Do not send this prompt")
	})

	t.Run("failed creation emits nothing", func(t *testing.T) {
		events := []productanalytics.Event{}
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).WithProductAnalytics(func(event productanalytics.Event) {
			events = append(events, event)
		})
		_, _, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "   "})
		require.Error(t, err)
		assert.Empty(t, events)
	})

	t.Run("issue intake uses import source without the ticket url", func(t *testing.T) {
		intakeFactory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		intakeCanvas, intakeExecution, _ := setupFactoryAppExecutionWithPayload(t, r, intakeFactory.ID, map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"issue": map[string]any{
					"html_url": "https://github.com/acme/payments/issues/12",
					"title":    "Handle duplicate refunds",
				},
			},
		})
		_, err = intakeFactory.CreateIntake(database.Conn(), intakeCanvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)

		events := []productanalytics.Event{}
		ctx := NewFactoryContext(database.Conn(), intakeCanvas, intakeExecution).WithProductAnalytics(func(event productanalytics.Event) {
			events = append(events, event)
		})
		created, inserted, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: "Handle duplicate refunds"})
		require.NoError(t, err)
		require.True(t, inserted)
		require.Len(t, events, 1)
		assert.Equal(t, "issue_import", events[0].Properties["creation_source"])
		assert.Equal(t, models.FactoryIntakeSourceGitHubIssues, events[0].Properties["issue_source"])
		encoded, err := json.Marshal(events[0])
		require.NoError(t, err)
		assert.NotContains(t, string(encoded), "https://github.com/acme/payments/issues/12")
		assert.NotEmpty(t, created.ID)

		duplicateCanvas, duplicateExecution, _ := setupFactoryAppExecutionWithPayload(t, r, intakeFactory.ID, map[string]any{
			"type": "github.issue",
			"data": map[string]any{
				"issue": map[string]any{
					"html_url": "https://github.com/acme/payments/issues/12",
					"title":    "Handle duplicate refunds",
				},
			},
		})
		_, err = intakeFactory.CreateIntake(database.Conn(), duplicateCanvas.ID, models.FactoryIntakeSourceGitHubIssues)
		require.NoError(t, err)
		events = nil
		_, inserted, err = NewFactoryContext(database.Conn(), duplicateCanvas, duplicateExecution).
			WithProductAnalytics(func(event productanalytics.Event) {
				events = append(events, event)
			}).
			CreateWorkOrder(core.WorkOrderParams{Title: "Handle duplicate refunds"})
		require.NoError(t, err)
		assert.False(t, inserted)
		assert.Empty(t, events)
	})

	t.Run("same status emits nothing and archive uses the system identity", func(t *testing.T) {
		order, created, err := NewFactoryContext(database.Conn(), canvas, nodeExecution).CreateWorkOrder(core.WorkOrderParams{Title: "Archive later"})
		require.NoError(t, err)
		require.True(t, created)

		events := []productanalytics.Event{}
		ctx := NewFactoryContext(database.Conn(), canvas, nodeExecution).WithProductAnalytics(func(event productanalytics.Event) {
			events = append(events, event)
		})
		_, changed, err := ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
			OrderID: order.ID,
			State:   models.FactoryWorkOrderStateDraft,
		})
		require.NoError(t, err)
		assert.False(t, changed)
		assert.Empty(t, events)

		_, changed, err = ctx.UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams{
			OrderID: order.ID,
			State:   models.FactoryWorkOrderStateClosed,
			Result:  models.FactoryWorkOrderResultArchived,
		})
		require.NoError(t, err)
		require.True(t, changed)
		require.Len(t, events, 1)
		assert.Equal(t, productanalytics.EventFactoryTaskArchived, events[0].Name)
		assert.Equal(t, "automation", events[0].Properties["archive_source"])
		assert.Equal(t, productanalytics.ActorSystem, events[0].ActorType)
		assert.NotEqual(t, r.Account.ID.String(), events[0].DistinctID)
	})
}

func setupFactoryAppExecution(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
) (*models.Canvas, *models.CanvasNodeExecution, *models.CanvasRun) {
	t.Helper()
	return setupFactoryAppExecutionWithPayload(t, r, factoryID, map[string]any{"key": "value"})
}

func setupFactoryAppExecutionWithPayload(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	payload map[string]any,
) (*models.Canvas, *models.CanvasNodeExecution, *models.CanvasRun) {
	t.Helper()

	const nodeID = "create-work-order"

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: nodeID, Type: models.NodeTypeComponent},
		},
		nil,
	)
	require.NoError(t, database.Conn().Model(canvas).Update("factory_id", factoryID).Error)
	canvas.FactoryID = &factoryID

	triggerEvent := support.EmitCanvasEventForNodeWithData(t, canvas.ID, nodeID, "default", nil, payload)
	run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(database.Conn(), triggerEvent)
	require.NoError(t, err)

	nodeExecution := support.CreateCanvasNodeExecution(t, canvas.ID, nodeID, triggerEvent.ID, triggerEvent.ID)
	nodeExecution.RunID = run.ID
	require.NoError(t, database.Conn().Save(nodeExecution).Error)

	return canvas, nodeExecution, run
}

func TestFactoryContextDefaultOwner(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	f, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	key, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, support.RandomName("key"), nil, r.User, nil, nil)
	require.NoError(t, err)
	account, err := models.CreateAccount(support.RandomName("owner"), support.RandomName("owner")+"@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, database.Conn().Delete(member).Error)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureCodexSubscriptions))
	connection, err := models.CreateFactoryCodexConnection(database.Conn(), r.Organization.ID, f.ID, r.User, time.Now())
	require.NoError(t, err)
	require.NoError(t, database.Conn().Model(connection).Updates(map[string]any{"credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
	for _, tc := range []struct {
		name, owner  string
		assigned     bool
		subscription bool
	}{
		{name: "active member", owner: r.User.String(), assigned: true},
		{name: "missing member", owner: uuid.NewString()},
		{name: "removed member", owner: member.ID.String()},
		{name: "non human", owner: key.ID.String()},
		{name: "old configuration"},
		{name: "connected subscriber", owner: r.User.String(), assigned: true, subscription: true},
		{name: "disconnected subscriber", owner: r.User.String(), subscription: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, f.UpdateOnboarding(database.Conn(), models.FactoryOnboardingPatch{CodexSubscription: &tc.subscription}))
			status := models.CodexConnectionStatusDisconnected
			if tc.assigned {
				status = models.CodexConnectionStatusReady
			}
			require.NoError(t, database.Conn().Model(connection).Update("status", status).Error)
			canvas, execution, _ := setupFactoryAppExecution(t, r, f.ID)
			ctx := NewFactoryContext(database.Conn(), canvas, execution)
			order, created, err := ctx.CreateWorkOrder(core.WorkOrderParams{Title: tc.name, DefaultOwnerID: tc.owner})
			require.NoError(t, err)
			require.True(t, created)
			persisted, err := f.FindWorkOrder(database.Conn(), uuid.MustParse(order.ID))
			require.NoError(t, err)
			assert.Nil(t, persisted.CreatedBy)
			if tc.assigned {
				require.Len(t, persisted.Assignees, 1)
				assert.Equal(t, r.User, persisted.Assignees[0].UserID)
				return
			}
			assert.Empty(t, persisted.Assignees)
		})
	}
}
