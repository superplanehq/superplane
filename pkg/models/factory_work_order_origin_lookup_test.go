package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactory_ListWorkOrderOriginURLsContaining(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, userID, factoryModel := setupFactoryWithUser(t, "origin-lookup")
	tx := database.DB(t.Context())

	otherFactory, err := CreateFactory(tx, org.ID, "origin-lookup-other", "", "")
	require.NoError(t, err)

	_, otherUserID, foreignFactory := setupFactoryWithUser(t, "origin-lookup-foreign")

	createOriginOrder := func(owner *Factory, title, rawURL, state, result string) {
		t.Helper()
		order, createErr := owner.CreateWorkOrderWithOrigin(
			tx,
			title,
			"",
			&userID,
			nil,
			nil,
			WorkOrderOrigin{URL: rawURL, Label: title},
		)
		require.NoError(t, createErr)
		if state == FactoryWorkOrderStateDraft {
			return
		}
		if state == FactoryWorkOrderStateOpen {
			_, updateErr := order.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{
				ToState: FactoryWorkOrderStateOpen,
			})
			require.NoError(t, updateErr)
			return
		}
		if state == FactoryWorkOrderStateClosed && result == FactoryWorkOrderResultRejected {
			_, updateErr := order.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{
				ToState: FactoryWorkOrderStateClosed,
				Result:  FactoryWorkOrderResultRejected,
			})
			require.NoError(t, updateErr)
			return
		}
		_, updateErr := order.UpdateStatus(tx, FactoryWorkOrderStatusUpdate{
			ToState: FactoryWorkOrderStateOpen,
		})
		require.NoError(t, updateErr)
		_, closeErr := order.Close(tx, result, &userID)
		require.NoError(t, closeErr)
	}

	createOriginOrder(factoryModel, "Draft issue", "https://acme.sentry.io/issues/11/", FactoryWorkOrderStateDraft, "")
	createOriginOrder(factoryModel, "Open issue", "https://acme.sentry.io/issues/22/", FactoryWorkOrderStateOpen, "")
	createOriginOrder(factoryModel, "Closed issue", "https://acme.sentry.io/issues/33/", FactoryWorkOrderStateClosed, FactoryWorkOrderResultCompleted)
	createOriginOrder(otherFactory, "Other factory", "https://acme.sentry.io/issues/11/", FactoryWorkOrderStateDraft, "")
	_, err = foreignFactory.CreateWorkOrderWithOrigin(
		tx,
		"Other org",
		"",
		&otherUserID,
		nil,
		nil,
		WorkOrderOrigin{URL: "https://acme.sentry.io/issues/11/", Label: "Other org"},
	)
	require.NoError(t, err)

	t.Run("returns matches in every state inside one factory", func(t *testing.T) {
		urls, listErr := factoryModel.ListWorkOrderOriginURLsContaining(tx, "/issues/11")
		require.NoError(t, listErr)
		assert.Equal(t, []string{"https://acme.sentry.io/issues/11/"}, urls)

		urls, listErr = factoryModel.ListWorkOrderOriginURLsContaining(tx, "/issues/22")
		require.NoError(t, listErr)
		assert.Equal(t, []string{"https://acme.sentry.io/issues/22/"}, urls)

		urls, listErr = factoryModel.ListWorkOrderOriginURLsContaining(tx, "/issues/33")
		require.NoError(t, listErr)
		assert.Equal(t, []string{"https://acme.sentry.io/issues/33/"}, urls)
	})

	t.Run("stays inside one factory and one organization", func(t *testing.T) {
		urls, listErr := otherFactory.ListWorkOrderOriginURLsContaining(tx, "/issues/22")
		require.NoError(t, listErr)
		assert.Empty(t, urls)

		urls, listErr = foreignFactory.ListWorkOrderOriginURLsContaining(tx, "/issues/22")
		require.NoError(t, listErr)
		assert.Empty(t, urls)
	})
}
