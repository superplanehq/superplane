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
	"github.com/superplanehq/superplane/pkg/models/factory"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__FactoryPullRequest(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	createOrder := func(t *testing.T, factoryModel *models.Factory) *models.FactoryWorkOrder {
		t.Helper()
		order, err := factoryModel.CreateWorkOrder(db, "PR order", "", &r.User, nil, nil)
		require.NoError(t, err)
		return order
	}

	t.Run("creates a pull request and records the added event", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)

		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			Provider:   models.FactoryPullRequestProviderGitHub,
			ExternalID: "4242",
			Repository: "acme/app",
			Number:     42,
			URL:        "https://github.com/acme/app/pull/42",
			Title:      "Fix retry handling",
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		assert.Equal(t, factoryModel.ID, pullRequest.FactoryID)
		assert.Equal(t, order.ID, pullRequest.WorkOrderID)
		assert.Equal(t, int64(42), pullRequest.Number)
		assert.Equal(t, "https://github.com/acme/app/pull/42", pullRequest.URL)
		assert.Nil(t, pullRequest.MergedAt)
		assert.Nil(t, pullRequest.ClosedAt)

		events, err := order.ListEvents(db, 10, nil)
		require.NoError(t, err)
		event := findWorkOrderEventType(t, events, factory.EventTypeOrderPullRequestAdded)
		var payload factory.WorkOrderPullRequestAdded
		require.NoError(t, json.Unmarshal(event.Data, &payload))
		require.NotNil(t, payload.PullRequest)
		assert.Equal(t, pullRequest.ID, payload.PullRequest.ID)
		assert.Equal(t, int64(42), payload.PullRequest.Number)
	})

	t.Run("fills repository and number from a GitHub URL", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)

		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/7#discussion_r1",
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestProviderGitHub, pullRequest.Provider)
		assert.Equal(t, "acme/app", pullRequest.Repository)
		assert.Equal(t, int64(7), pullRequest.Number)
		assert.Equal(t, "https://github.com/acme/app/pull/7", pullRequest.URL)
	})

	t.Run("rejects a duplicate identity instead of moving the work order", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		first := createOrder(t, factoryModel)
		second := createOrder(t, factoryModel)

		_, err = first.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/9",
		})
		require.NoError(t, err)

		_, err = second.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/9",
		})
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestAlreadyExists)
	})

	t.Run("finds by id, external id, identity, and url", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		created, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			ExternalID: "ext-11",
			URL:        "https://github.com/acme/app/pull/11",
		})
		require.NoError(t, err)

		byID, err := factoryModel.FindPullRequest(db, models.FactoryPullRequestLookup{ID: created.ID})
		require.NoError(t, err)
		assert.Equal(t, created.ID, byID.ID)

		byExternal, err := factoryModel.FindPullRequest(db, models.FactoryPullRequestLookup{
			Provider:   models.FactoryPullRequestProviderGitHub,
			ExternalID: "ext-11",
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, byExternal.ID)

		byIdentity, err := factoryModel.FindPullRequest(db, models.FactoryPullRequestLookup{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     11,
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, byIdentity.ID)

		byURL, err := factoryModel.FindPullRequest(db, models.FactoryPullRequestLookup{
			URL: "https://github.com/acme/app/pull/11/",
		})
		require.NoError(t, err)
		assert.Equal(t, created.ID, byURL.ID)
	})

	t.Run("lists pull requests for a work order number", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		other := createOrder(t, factoryModel)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/21"})
		require.NoError(t, err)
		_, err = other.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/22"})
		require.NoError(t, err)

		listed, err := factoryModel.ListPullRequests(db, models.FactoryPullRequestFilter{WorkOrderNumber: &order.Number})
		require.NoError(t, err)
		require.Len(t, listed, 1)
		assert.Equal(t, int64(21), listed[0].Number)
	})

	t.Run("lists pull requests by repository and states", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		draftOrder := createOrder(t, factoryModel)
		otherRepoOrder := createOrder(t, factoryModel)
		closedOrder := createOrder(t, factoryModel)

		open, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/61", State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		draft, err := draftOrder.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/62", State: models.FactoryPullRequestStateDraft,
		})
		require.NoError(t, err)
		_, err = otherRepoOrder.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/other/pull/63", State: models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		_, err = closedOrder.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL: "https://github.com/acme/app/pull/64", State: models.FactoryPullRequestStateClosed,
		})
		require.NoError(t, err)

		listed, err := factoryModel.ListPullRequests(db, models.FactoryPullRequestFilter{
			Repository: "ACME/App",
			States:     []string{models.FactoryPullRequestStateOpen, models.FactoryPullRequestStateDraft},
		})
		require.NoError(t, err)
		ids := []uuid.UUID{listed[0].ID}
		for _, pr := range listed[1:] {
			ids = append(ids, pr.ID)
		}
		assert.ElementsMatch(t, []uuid.UUID{open.ID, draft.ID}, ids)
	})

	t.Run("rejects an invalid state in the states filter", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)

		_, err = factoryModel.ListPullRequests(db, models.FactoryPullRequestFilter{
			States: []string{"bogus"},
		})
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestInvalid)
	})

	t.Run("merged stamps merged_at once and later open does not clear it", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		explicit := time.Now().Add(-2 * time.Hour).UTC().Truncate(time.Second)
		pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:      "https://github.com/acme/app/pull/31",
			State:    models.FactoryPullRequestStateMerged,
			MergedAt: &explicit,
		})
		require.NoError(t, err)
		require.NotNil(t, pullRequest.MergedAt)
		assert.True(t, pullRequest.MergedAt.Equal(explicit))

		open := models.FactoryPullRequestStateOpen
		require.NoError(t, pullRequest.Update(db, models.FactoryPullRequestPatch{State: &open}))
		require.NotNil(t, pullRequest.MergedAt)
		assert.True(t, pullRequest.MergedAt.Equal(explicit))

		events, err := order.ListEvents(db, 10, nil)
		require.NoError(t, err)
		_ = findWorkOrderEventType(t, events, factory.EventTypeOrderPullRequestUpdated)
	})

	t.Run("links a run once and rejects a second pull request", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		first, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/41"})
		require.NoError(t, err)
		second, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/42"})
		require.NoError(t, err)

		canvas, _ := support.CreateCanvas(
			t,
			r.Organization.ID,
			r.User,
			[]models.CanvasNode{{NodeID: "trigger", Type: models.NodeTypeTrigger}},
			nil,
		)
		rootEvent := support.EmitCanvasEventForNode(t, canvas.ID, "trigger", "default", nil)
		run, err := models.FindOrCreateCanvasRunForRootEventInTransaction(db, rootEvent)
		require.NoError(t, err)

		require.NoError(t, first.LinkRun(db, run.ID, "Address review from alice"))
		require.NoError(t, first.LinkRun(db, run.ID, "ignored on retry"))

		err = second.LinkRun(db, run.ID, "")
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestRunAlreadyLinked)

		runs, err := models.ListPullRequestRuns(db, []uuid.UUID{first.ID, second.ID})
		require.NoError(t, err)
		require.Len(t, runs[first.ID], 1)
		assert.Equal(t, run.ID, runs[first.ID][0].Run.ID)
		assert.Equal(t, "Address review from alice", runs[first.ID][0].Description)
		assert.Empty(t, runs[second.ID])
	})

	t.Run("does not find pull requests from another factory", func(t *testing.T) {
		factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order := createOrder(t, factoryModel)
		created, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{URL: "https://github.com/acme/app/pull/51"})
		require.NoError(t, err)

		_, err = other.FindPullRequest(db, models.FactoryPullRequestLookup{ID: created.ID})
		assert.ErrorIs(t, err, models.ErrFactoryPullRequestNotFound)
		assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func findWorkOrderEventType(t *testing.T, events []models.FactoryWorkOrderEvent, eventType string) models.FactoryWorkOrderEvent {
	t.Helper()
	for _, event := range events {
		if event.Type == eventType {
			return event
		}
	}
	t.Fatalf("missing event %s", eventType)
	return models.FactoryWorkOrderEvent{}
}
