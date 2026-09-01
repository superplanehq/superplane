package models_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__ListDeletedFactories(t *testing.T) {
	r := support.Setup(t)

	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, factory.SoftDelete(database.DB(t.Context())))

	deleted, err := models.ListDeletedFactories(database.DB(t.Context()))
	require.NoError(t, err)

	var found bool
	for _, item := range deleted {
		if item.ID == factory.ID {
			found = true
			assert.True(t, item.DeletedAt.Valid)
		}
	}
	assert.True(t, found)
}

func Test__ListDeletedFactories__IncludesFactoriesInDeletedOrg(t *testing.T) {
	r := support.Setup(t)

	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))
	deletedAt := time.Now().AddDate(0, 0, -31)
	require.NoError(t, database.DB(t.Context()).Unscoped().
		Model(&models.Organization{}).
		Where("id = ?", r.Organization.ID).
		Update("deleted_at", deletedAt).
		Error)

	deleted, err := models.ListDeletedFactories(database.DB(t.Context()))
	require.NoError(t, err)

	var found *models.Factory
	for i := range deleted {
		if deleted[i].ID == factory.ID {
			found = &deleted[i]
			break
		}
	}
	require.NotNil(t, found)
	require.True(t, found.DeletedAt.Valid)
	assert.WithinDuration(t, deletedAt.UTC(), found.DeletedAt.Time.UTC(), time.Second)
}

func Test__ListDeletedFactories__UsesEarliestDeletedAt(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, factory.SoftDelete(db))

	factoryDeletedAt := time.Now().AddDate(0, 0, -40)
	require.NoError(t, db.Unscoped().Model(&models.Factory{}).
		Where("id = ?", factory.ID).
		Update("deleted_at", factoryDeletedAt).
		Error)

	require.NoError(t, models.SoftDeleteOrganization(r.Organization.ID.String()))
	orgDeletedAt := time.Now().AddDate(0, 0, -10)
	require.NoError(t, db.Unscoped().Model(&models.Organization{}).
		Where("id = ?", r.Organization.ID).
		Update("deleted_at", orgDeletedAt).
		Error)

	deleted, err := models.ListDeletedFactories(db)
	require.NoError(t, err)

	var found *models.Factory
	for i := range deleted {
		if deleted[i].ID == factory.ID {
			found = &deleted[i]
			break
		}
	}
	require.NotNil(t, found)
	require.True(t, found.DeletedAt.Valid)
	assert.WithinDuration(t, factoryDeletedAt.UTC(), found.DeletedAt.Time.UTC(), time.Second)
}

func Test__LockDeletedFactory__SkipLocked(t *testing.T) {
	r := support.Setup(t)

	factory, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, factory.SoftDelete(database.DB(t.Context())))

	tx1 := database.DB(t.Context()).Begin()
	t.Cleanup(func() { _ = tx1.Rollback() })

	locked, err := models.LockDeletedFactory(tx1, factory.ID)
	require.NoError(t, err)
	require.Equal(t, factory.ID, locked.ID)

	tx2 := database.DB(t.Context()).Begin()
	t.Cleanup(func() { _ = tx2.Rollback() })

	_, err = models.LockDeletedFactory(tx2, factory.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__FactoryResourceCleaner__HardDeletesFactoryDomain(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, []uuid.UUID{r.User}, nil)
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "step",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&execution).Error)

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/41",
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, run.ID, "Address review from alice"))

	// Checks reference the order and factory with RESTRICT FKs, so the
	// cleaner must remove them before the order and factory rows.
	_, err = order.ReportCheck(db, models.FactoryWorkOrderCheckParams{
		Key:      "risk-review",
		Name:     "Risk review",
		Score:    42,
		MaxScore: 100,
	})
	require.NoError(t, err)

	require.NoError(t, factory.SoftDelete(db))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := run.DeleteChain(tx)
		return err
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		deleted, complete, cleanErr := models.NewFactoryResourceCleaner(tx, factory).WithLimit(500).Run()
		require.NoError(t, cleanErr)
		assert.True(t, complete)
		assert.Greater(t, deleted, int64(0))
		return nil
	}))

	var factoryCount int64
	require.NoError(t, db.Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
	assert.Equal(t, int64(0), factoryCount)

	var orderCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factory.ID).Count(&orderCount).Error)
	assert.Equal(t, int64(0), orderCount)

	var checkCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrderCheck{}).Where("factory_id = ?", factory.ID).Count(&checkCount).Error)
	assert.Equal(t, int64(0), checkCount)

	var pullRequestCount int64
	require.NoError(t, db.Model(&models.FactoryPullRequest{}).Where("factory_id = ?", factory.ID).Count(&pullRequestCount).Error)
	assert.Equal(t, int64(0), pullRequestCount)
}

func Test__FactoryResourceCleaner__DeletesPullRequestRunLinksBeforePullRequests(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/app/pull/77",
	})
	require.NoError(t, err)
	require.NoError(t, pullRequest.LinkRun(db, run.ID, "Please add tests."))
	require.NoError(t, factory.SoftDelete(db))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, complete, cleanErr := models.NewFactoryResourceCleaner(tx, factory).WithLimit(500).Run()
		require.NoError(t, cleanErr)
		assert.True(t, complete)
		return nil
	}))

	var linkCount int64
	require.NoError(t, db.Model(&models.FactoryPullRequestRun{}).Where("run_id = ?", run.ID).Count(&linkCount).Error)
	assert.Equal(t, int64(0), linkCount)
}

func Test__FactoryResourceCleaner__RespectsLimit(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err := factory.CreateWorkOrder(db, support.RandomName("order"), "", &r.User, nil, nil)
		require.NoError(t, err)
	}
	require.NoError(t, factory.SoftDelete(db))

	var deleted int64
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var complete bool
		var cleanErr error
		deleted, complete, cleanErr = models.NewFactoryResourceCleaner(tx, factory).WithLimit(1).Run()
		require.NoError(t, cleanErr)
		assert.False(t, complete)
		assert.LessOrEqual(t, deleted, int64(1))
		return nil
	}))

	var orderCount int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factory.ID).Count(&orderCount).Error)
	// Orders still have events, so a 1-row budget clears an event first and leaves all orders.
	assert.Equal(t, int64(3), orderCount)
}

func Test__FactoryResourceCleaner__LargeFactoryStaysWithinBudget(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	const orderCount = 2000
	const extraEventsPerOrder = 4
	now := time.Now()

	orders := make([]models.FactoryWorkOrder, orderCount)
	for i := range orders {
		orders[i] = models.FactoryWorkOrder{
			ID:             uuid.New(),
			OrganizationID: r.Organization.ID,
			FactoryID:      factory.ID,
			Number:         int64(i + 1),
			Title:          support.RandomName("order"),
			State:          models.FactoryWorkOrderStateOpen,
			CreatedByID:    &r.User,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}
	require.NoError(t, db.CreateInBatches(orders, 500).Error)

	events := make([]models.FactoryWorkOrderEvent, 0, orderCount*(1+extraEventsPerOrder))
	for _, order := range orders {
		for j := 0; j < 1+extraEventsPerOrder; j++ {
			events = append(events, models.FactoryWorkOrderEvent{
				ID:          uuid.New(),
				WorkOrderID: order.ID,
				Type:        "order.opened",
				Data:        []byte(`{}`),
				CreatedAt:   now,
			})
		}
	}
	require.NoError(t, db.CreateInBatches(events, 500).Error)
	require.NoError(t, factory.SoftDelete(db))

	const limit = 500
	start := time.Now()
	var deleted int64
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var complete bool
		var cleanErr error
		deleted, complete, cleanErr = models.NewFactoryResourceCleaner(tx, factory).WithLimit(limit).Run()
		require.NoError(t, cleanErr)
		assert.False(t, complete)
		assert.LessOrEqual(t, deleted, int64(limit))
		assert.Greater(t, deleted, int64(0))
		return nil
	}))
	assert.Less(t, time.Since(start), 2*time.Second, "single budgeted tick should stay fast on large factory")

	start = time.Now()
	for {
		var complete bool
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			_, complete, err = models.NewFactoryResourceCleaner(tx, factory).WithLimit(limit).Run()
			return err
		}))
		if complete {
			break
		}
		require.Less(t, time.Since(start), 30*time.Second, "large factory cleanup should finish across ticks quickly")
	}
	assert.Less(t, time.Since(start), 30*time.Second)

	var remainingOrders int64
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("factory_id = ?", factory.ID).Count(&remainingOrders).Error)
	assert.Equal(t, int64(0), remainingOrders)

	var factoryCount int64
	require.NoError(t, db.Unscoped().Model(&models.Factory{}).Where("id = ?", factory.ID).Count(&factoryCount).Error)
	assert.Equal(t, int64(0), factoryCount)
}

func Test__CanvasRun__DeleteChain__NullsFactoryWorkOrderExecutionRunID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "step",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusFinished,
		Result:         models.CanvasRunResultPassed,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&execution).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := run.DeleteChain(tx)
		return err
	}))

	var persisted models.FactoryWorkOrderExecution
	require.NoError(t, db.Where("id = ?", execution.ID).First(&persisted).Error)
	assert.Nil(t, persisted.RunID)
	assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, persisted.Status)
	assert.Equal(t, models.CanvasRunResultPassed, persisted.Result)

	reloaded, err := models.FindWorkOrderLineDispatch(db, dispatch.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderLineDispatchStateActive, reloaded.State)

	_, err = models.FindWorkOrderExecutionByRunID(db, run.ID)
	assert.ErrorIs(t, err, models.ErrFactoryWorkOrderExecutionNotFound)

	var runCount int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", run.ID).Count(&runCount).Error)
	assert.Equal(t, int64(0), runCount)
}

func Test__CanvasRun__DeleteChain__CancelsInFlightFactoryWorkOrderExecution(t *testing.T) {
	for _, status := range []string{
		models.FactoryWorkOrderExecutionStatusPending,
		models.FactoryWorkOrderExecutionStatusRunning,
	} {
		t.Run(status, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())

			factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
			require.NoError(t, err)
			order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
			require.NoError(t, err)
			line, err := factory.CreateLine(db, "line", nil)
			require.NoError(t, err)

			canvas, _ := support.CreateCanvas(
				t,
				r.Organization.ID,
				r.User,
				[]models.CanvasNode{
					{NodeID: "trigger", Type: models.NodeTypeTrigger},
					{NodeID: "node-1", Type: models.NodeTypeComponent},
				},
				nil,
			)
			rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
			run := createRunForRootEvent(t, rootEvent)
			dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

			now := time.Now()
			execution := models.FactoryWorkOrderExecution{
				ID:             uuid.New(),
				OrganizationID: r.Organization.ID,
				FactoryID:      factory.ID,
				WorkOrderID:    order.ID,
				LineID:         line.ID,
				LineDispatchID: dispatch.ID,
				StepIndex:      0,
				StepName:       "step",
				RunID:          &run.ID,
				Status:         status,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			require.NoError(t, db.Create(&execution).Error)

			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				_, err := run.DeleteChain(tx)
				return err
			}))

			var persisted models.FactoryWorkOrderExecution
			require.NoError(t, db.Where("id = ?", execution.ID).First(&persisted).Error)
			assert.Nil(t, persisted.RunID)
			assert.Equal(t, models.FactoryWorkOrderExecutionStatusFinished, persisted.Status)
			assert.Equal(t, models.CanvasRunResultCancelled, persisted.Result)
			require.NotNil(t, persisted.FinishedAt)

			var active int64
			require.NoError(t, db.Model(&models.FactoryWorkOrderExecution{}).
				Where("line_id = ?", line.ID).
				Where("status IN ?", []string{
					models.FactoryWorkOrderExecutionStatusPending,
					models.FactoryWorkOrderExecutionStatusRunning,
				}).
				Count(&active).Error)
			assert.Equal(t, int64(0), active)

			reloaded, err := models.FindWorkOrderLineDispatch(db, dispatch.ID)
			require.NoError(t, err)
			assert.Equal(t, models.FactoryWorkOrderLineDispatchStateFinished, reloaded.State)
			assert.Equal(t, models.CanvasRunResultCancelled, reloaded.Result)
		})
	}
}

func Test__CanvasRun__DeleteChain__ClearsWorkOrderSourceRunID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{NodeID: "trigger", Type: models.NodeTypeTrigger},
			{NodeID: "node-1", Type: models.NodeTypeComponent},
		},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, &run.ID)
	require.NoError(t, err)
	require.NotNil(t, order.SourceRunID)
	assert.Equal(t, run.ID, *order.SourceRunID)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := run.DeleteChain(tx)
		return err
	}))

	persisted, err := factory.FindWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Nil(t, persisted.SourceRunID)

	var runCount int64
	require.NoError(t, db.Model(&models.CanvasRun{}).Where("id = ?", run.ID).Count(&runCount).Error)
	assert.Equal(t, int64(0), runCount)
}

// Reverting an open work order back to draft while a step is still running
// would leave the executor and FSM out of sync; UpdateStatus must reject it
// with the same guard DispatchWorkOrder uses.
func Test__FactoryWorkOrder__UpdateStatus__OpenToDraft__RejectsWhenExecutionActive(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Order", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateOpen,
		Actor:   &r.User,
	})
	require.NoError(t, err)

	line, err := factory.CreateLine(db, "line", nil)
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	dispatch := support.CreateFactoryLineDispatch(t, r.Organization.ID, factory.ID, order.ID, line.ID, line.Name, nil)

	now := time.Now()
	execution := models.FactoryWorkOrderExecution{
		ID:             uuid.New(),
		OrganizationID: r.Organization.ID,
		FactoryID:      factory.ID,
		WorkOrderID:    order.ID,
		LineID:         line.ID,
		LineDispatchID: dispatch.ID,
		StepIndex:      0,
		StepName:       "step",
		RunID:          &run.ID,
		Status:         models.FactoryWorkOrderExecutionStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	require.NoError(t, db.Create(&execution).Error)

	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateDraft,
		Actor:   &r.User,
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrFactoryWorkOrderLineDispatchActive)

	reloaded, err := models.FindUnscopedWorkOrder(db, order.ID)
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State,
		"failed back-to-draft must not mutate the row")

	require.NoError(t, execution.MarkFinished(db, models.FactoryWorkOrderResultCompleted))
	require.NoError(t, dispatch.Finish(db, models.FactoryWorkOrderResultCompleted))

	_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateDraft,
		Actor:   &r.User,
	})
	require.NoError(t, err)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, order.State)
}

// A canvas-created work order (SourceRunID set) must carry the source run
// + app refs on its very first `order.status.updated` event, so the
// timeline can link back to the originating run without waiting for the
// draft → open promotion.
func Test__Factory__CreateWorkOrder__WithSourceRun__EnrichesInitialStatusEvent(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
		nil,
	)
	rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
	run := createRunForRootEvent(t, rootEvent)

	order, err := factory.CreateWorkOrder(db, "Canvas-created", "", &r.User, nil, &run.ID)
	require.NoError(t, err)

	events, err := order.ListEvents(db, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, factoryevents.EventTypeOrderStatusUpdated, events[0].Type)

	var initial factoryevents.WorkOrderStatusUpdated
	require.NoError(t, json.Unmarshal(events[0].Data, &initial))
	assert.Equal(t, "", initial.FromState)
	assert.Equal(t, models.FactoryWorkOrderStateDraft, initial.ToState)
	require.NotNil(t, initial.Run, "source-run created orders must attribute run on creation")
	assert.Equal(t, run.ID, initial.Run.ID)
	require.NotNil(t, initial.App, "source-run created orders must attribute app on creation")
	assert.Equal(t, canvas.ID, initial.App.ID)
}

// The plain path (no source run) still records a bare "" → draft event.
func Test__Factory__CreateWorkOrder__WithoutSourceRun__OmitsRunAndApp(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factory.CreateWorkOrder(db, "Manual intake", "", &r.User, nil, nil)
	require.NoError(t, err)

	events, err := order.ListEvents(db, 10, nil)
	require.NoError(t, err)
	require.Len(t, events, 1)

	var initial factoryevents.WorkOrderStatusUpdated
	require.NoError(t, json.Unmarshal(events[0].Data, &initial))
	assert.Nil(t, initial.Run)
	assert.Nil(t, initial.App)
}
