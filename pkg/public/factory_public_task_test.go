package public

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestPublicFactoryTaskAccessAndProjection(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, _, _ := setupTestServer(r, t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureFactories))
	factory, line := openPublicLine(t, r, true)
	db := database.DB(t.Context())
	order, err := factory.CreateWorkOrder(db, "Public task", "Public description", nil, nil, nil)
	require.NoError(t, err)
	comment := models.FactoryWorkOrderComment{ID: uuid.New(), OrganizationID: r.Organization.ID, FactoryID: factory.ID, WorkOrderID: order.ID, AuthorKind: "user", Body: "Public comment"}
	require.NoError(t, db.Create(&comment).Error)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{Provider: models.FactoryPullRequestProviderGitHub, Repository: "acme/app", Number: 42, URL: "https://github.com/acme/app/pull/42?token=private-token", State: models.FactoryPullRequestStateOpen})
	require.NoError(t, err)
	path := strings.TrimSuffix(publicBoardPath(r, factory, line), "/board") + "/tasks/" + strconv.FormatInt(order.Number, 10)
	response := execRequest(server, requestParams{method: http.MethodGet, path: path})
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	require.Contains(t, response.Body.String(), "Public description")
	var task publicTask
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &task))
	require.Equal(t, order.Number, task.Card.Number)
	require.Len(t, task.Comments, 1)
	require.Equal(t, "Public comment", task.Comments[0].Body)
	require.Len(t, task.PullRequests, 1)
	require.Equal(t, "https://github.com/acme/app/pull/42", task.PullRequests[0].URL)
	require.NotContains(t, response.Body.String(), "private-token")
	other, err := models.CreateFactory(db, r.Organization.ID, "Other", "other", "OTHER")
	require.NoError(t, err)
	require.NoError(t, db.Model(other).Updates(map[string]any{"public": true, "onboarding_completed_at": time.Now()}).Error)
	otherLine, err := other.CreateLine(db, "Other", nil)
	require.NoError(t, err)
	otherPath := strings.TrimSuffix(publicBoardPath(r, other, otherLine), "/board") + "/tasks/" + strconv.FormatInt(order.Number, 10)
	require.Equal(t, http.StatusNotFound, execRequest(server, requestParams{method: http.MethodGet, path: otherPath}).Code)
	wrongLinePath := strings.Replace(path, line.ID.String(), otherLine.ID.String(), 1)
	require.Equal(t, http.StatusNotFound, execRequest(server, requestParams{method: http.MethodGet, path: wrongLinePath}).Code)
	for _, private := range []string{order.ID.String(), "cost", "spend", "artifacts", "logs", "runId", "repository", "defaultBranch"} {
		require.NotContains(t, response.Body.String(), private)
	}
	require.NoError(t, db.Model(factory).Update("public", false).Error)
	response = execRequest(server, requestParams{method: http.MethodGet, path: path})
	require.Equal(t, http.StatusNotFound, response.Code)
	require.NotContains(t, response.Body.String(), "Public description")
	require.NoError(t, db.Model(factory).Update("public", true).Error)
	require.NoError(t, db.Model(order).Updates(map[string]any{"state": models.FactoryWorkOrderStateClosed, "result": models.FactoryWorkOrderResultArchived}).Error)
	response = execRequest(server, requestParams{method: http.MethodGet, path: path})
	require.Equal(t, http.StatusNotFound, response.Code)
}
