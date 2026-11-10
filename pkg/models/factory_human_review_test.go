package models_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestHumanReviewPollingFiltersWorkspacesAndPagesByID(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	var expected []uuid.UUID
	for i, scenario := range []struct {
		name                                     string
		installed, enabled, unpublished, deleted bool
	}{
		{name: "enabled", installed: true, enabled: true},
		{name: "disabled", installed: true},
		{name: "not installed"},
		{name: "unpublished", installed: true, unpublished: true},
		{name: "deleted", installed: true, deleted: true},
	} {
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, scenario.name, "", &r.User, nil, nil)
		require.NoError(t, err)
		if scenario.installed {
			metadata := models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 1)
			if scenario.unpublished {
				metadata = nil
			}
			canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
				NodeID: models.HumanReviewTriggerNodeID, Type: models.NodeTypeTrigger,
				Metadata:      datatypes.NewJSONType(metadata),
				Configuration: datatypes.NewJSONType(map[string]any{"enabled": scenario.enabled}),
			}}, nil)
			require.NoError(t, db.Model(canvas).Update("factory_id", factory.ID).Error)
			if scenario.unpublished {
				node, err := models.FindCanvasNode(db, canvas.ID, models.HumanReviewTriggerNodeID)
				require.NoError(t, err)
				require.NoError(t, db.Model(node).Update("metadata", datatypes.NewJSONType(models.FactoryAppTemplateMetadata(models.FactoryAppTemplateHumanReviewID, 1))).Error)
			}
			if scenario.deleted {
				require.NoError(t, db.Delete(canvas).Error)
			}
		}
		for j, state := range []string{models.FactoryPullRequestStateOpen, models.FactoryPullRequestStateDraft, models.FactoryPullRequestStateClosed} {
			pr, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
				URL: fmt.Sprintf("https://github.com/acme/app/pull/%d", i*10+j+1), State: state,
			})
			require.NoError(t, err)
			if scenario.installed && !scenario.unpublished && !scenario.deleted && state != models.FactoryPullRequestStateClosed {
				expected = append(expected, pr.ID)
			}
		}
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: models.FactoryPullRequestProviderBitbucket, URL: fmt.Sprintf("https://bitbucket.org/acme/app/pull-requests/%d", i+1)})
		require.NoError(t, err)
	}
	var found []uuid.UUID
	cursor := uuid.Nil
	for range len(expected) + 1 {
		page, err := models.ListFactoryHumanReviewPullRequestsToReconcile(db, cursor, 1)
		require.NoError(t, err)
		require.LessOrEqual(t, len(page), 1)
		if len(page) == 0 {
			break
		}
		require.Greater(t, page[0].ID.String(), cursor.String())
		found = append(found, page[0].ID)
		cursor = page[0].ID
		// Reconciliation changes timestamps; the cursor must remain stable.
		require.NoError(t, db.Model(&page[0]).Update("updated_at", page[0].UpdatedAt.AddDate(0, 0, 1)).Error)
	}
	require.ElementsMatch(t, expected, found)
}
