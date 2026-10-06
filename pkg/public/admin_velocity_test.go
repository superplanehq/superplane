package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestAdminOrganizationVelocity(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.DB(t.Context())

	const (
		repository = "hidden-owner/hidden-velocity-repo"
		authorName = "Hidden Velocity Author"
		taskTitle  = "Hidden velocity task title"
	)

	older, err := models.CreateFactory(db, r.Organization.ID, "Older workspace", "", "")
	require.NoError(t, err)
	newer, err := models.CreateFactory(db, r.Organization.ID, "Newer workspace", "", "")
	require.NoError(t, err)
	require.NoError(t, newer.UpdateOnboarding(db, models.FactoryOnboardingPatch{AppRepository: ptr(repository)}))
	setFactoryUpdatedAt(t, older, time.Now().Add(-2*time.Hour))
	setFactoryUpdatedAt(t, newer, time.Now())

	now := time.Now()
	seedAdminVelocityPullRequest(t, newer, repository, taskTitle, now.Add(-time.Hour))
	seedAdminVelocityPeopleMerge(t, r, newer, repository, authorName, now.Add(-2*time.Hour))
	canvas, _ := support.CreateFactoryAppWithOnRunTrigger(t, r, newer.ID, "Nightly close", "start")
	seedAdminVelocityAutomationRun(t, canvas, "start", now.Add(-3*time.Hour))

	t.Run("non-admin gets 404", func(t *testing.T) {
		account, err := models.CreateAccount("Regular User", support.RandomName("regular-velocity")+"@example.com")
		require.NoError(t, err)
		signer := jwt.NewSigner("test-client-secret")
		regularToken, err := authentication.GenerateAccountToken(signer, account.ID.String(), time.Now(), time.Hour)
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), ""),
			authCookie: regularToken,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("returns totals points intake and automations without people or repository", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "period_days=7"),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		var body adminVelocityBody
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		require.Len(t, body.Factories, 2)
		assert.Equal(t, newer.ID.String(), body.Factories[0].ID)
		assert.Equal(t, "Newer workspace", body.Factories[0].Name)
		assert.Equal(t, older.ID.String(), body.Factories[1].ID)
		assert.Equal(t, newer.ID.String(), body.FactoryID)
		assert.Equal(t, 7, body.PeriodDays)
		assert.True(t, body.HasPeopleCohort)
		assert.GreaterOrEqual(t, body.Totals.SuperplaneMerged, 1)
		assert.GreaterOrEqual(t, body.Totals.PeopleMerged, 1)
		assert.GreaterOrEqual(t, body.Totals.TasksClosed, 1)
		require.Len(t, body.Points, 7)
		assert.NotEmpty(t, body.Points[len(body.Points)-1].Day)
		require.NotEmpty(t, body.IntakeSources)
		assert.NotEmpty(t, body.IntakeSources[0].Key)
		assert.NotEmpty(t, body.IntakeSources[0].Label)
		assert.Greater(t, body.IntakeSources[0].Merged, 0)
		require.Len(t, body.Automations, 1)
		assert.Equal(t, canvas.ID.String(), body.Automations[0].ID)
		assert.Equal(t, canvas.Name, body.Automations[0].Name)
		assert.GreaterOrEqual(t, body.Automations[0].Runs, 1)

		assertAdminVelocityOmitsPrivateFields(t, response.Body.Bytes(), repository, authorName, taskTitle)
	})

	t.Run("defaults a missing period to 30 days", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), ""),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)
		var body adminVelocityBody
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, 30, body.PeriodDays)
		assert.Len(t, body.Points, 30)
	})

	t.Run("clamps a period above 30 days", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "period_days=90"),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code)
		var body adminVelocityBody
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
		assert.Equal(t, 30, body.PeriodDays)
		assert.Len(t, body.Points, 30)
	})

	t.Run("unknown organization returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath("00000000-0000-0000-0000-000000000000", ""),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("unknown workspace returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "factory_id="+uuid.NewString()),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("malformed workspace returns 404", func(t *testing.T) {
		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "factory_id=not-a-workspace"),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("deleted workspace returns 404", func(t *testing.T) {
		deleted, err := models.CreateFactory(db, r.Organization.ID, "Deleted workspace", "", "")
		require.NoError(t, err)
		require.NoError(t, deleted.SoftDelete(db))

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "factory_id="+deleted.ID.String()),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("foreign workspace returns 404", func(t *testing.T) {
		foreign, err := models.CreateOrganization(support.RandomName("foreign-velocity"), "")
		require.NoError(t, err)
		foreignFactory, err := models.CreateFactory(db, foreign.ID, "Foreign workspace", "", "")
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(r.Organization.ID.String(), "factory_id="+foreignFactory.ID.String()),
			authCookie: token,
		})
		assert.Equal(t, http.StatusNotFound, response.Code)
	})

	t.Run("organization with no workspaces returns an empty list", func(t *testing.T) {
		empty, err := models.CreateOrganization(support.RandomName("empty-velocity"), "")
		require.NoError(t, err)

		response := execRequest(server, requestParams{
			method:     "GET",
			path:       adminVelocityPath(empty.ID.String(), ""),
			authCookie: token,
		})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())

		var raw map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &raw))
		factories, ok := raw["factories"].([]any)
		require.True(t, ok)
		assert.Empty(t, factories)
		assert.NotContains(t, raw, "factoryId")
		assert.NotContains(t, raw, "totals")
		assert.NotContains(t, raw, "repository")
		assert.NotContains(t, raw, "people")
	})

	t.Run("supplied workspace on an empty organization returns 404", func(t *testing.T) {
		empty, err := models.CreateOrganization(support.RandomName("empty-velocity-id"), "")
		require.NoError(t, err)

		for _, query := range []string{
			"factory_id=not-a-workspace",
			"factory_id=" + uuid.NewString(),
		} {
			response := execRequest(server, requestParams{
				method:     "GET",
				path:       adminVelocityPath(empty.ID.String(), query),
				authCookie: token,
			})
			assert.Equal(t, http.StatusNotFound, response.Code, query)
		}
	})
}

func adminVelocityPath(orgID, query string) string {
	path := "/admin/api/organizations/" + orgID + "/velocity"
	if query == "" {
		return path
	}
	return path + "?" + query
}

func setFactoryUpdatedAt(t *testing.T, factory *models.Factory, at time.Time) {
	t.Helper()
	require.NoError(t, database.DB(t.Context()).Model(factory).UpdateColumn("updated_at", at).Error)
}

func seedAdminVelocityPullRequest(t *testing.T, factory *models.Factory, repository, title string, at time.Time) {
	t.Helper()
	db := database.DB(t.Context())
	order, err := factory.CreateWorkOrder(db, title, "", nil, nil, nil)
	require.NoError(t, err)
	_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
		Repository: repository,
		URL:        fmt.Sprintf("https://github.com/%s/pull/4", repository),
		Title:      title,
		State:      models.FactoryPullRequestStateMerged,
		MergedAt:   &at,
	})
	require.NoError(t, err)
}

func seedAdminVelocityPeopleMerge(
	t *testing.T,
	r *support.ResourceRegistry,
	factory *models.Factory,
	repository, authorName string,
	at time.Time,
) {
	t.Helper()
	db := database.DB(t.Context())
	merge := models.NewFactoryVelocityRepositoryMerge(
		r.Organization.ID,
		factory.ID,
		repository,
		80,
		models.FactoryVelocityMergeSourcePeople,
		at,
	)
	merge.AuthorName = authorName
	merge.AuthorLogin = "hidden-velocity-author"
	from := time.Now().AddDate(0, 0, -90)
	to := time.Now().Add(time.Hour)
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factory.ID, from, to, []models.FactoryVelocityRepositoryMerge{merge}))
	sync, err := models.ClaimFactoryVelocitySync(db, factory.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, sync.RecordSuccess(db, repository, time.Now(), from))
}

func seedAdminVelocityAutomationRun(t *testing.T, canvas *models.Canvas, entrypoint string, startedAt time.Time) {
	t.Helper()
	db := database.DB(t.Context())
	run, err := models.CreateCanvasRunInTransaction(db, canvas.ID, entrypoint, models.CanvasRunStateFinished, models.CanvasRunResultPassed)
	require.NoError(t, err)
	require.NoError(t, db.Model(run).Updates(map[string]any{
		"created_at":  startedAt,
		"finished_at": startedAt.Add(30 * time.Minute),
	}).Error)
}

func assertAdminVelocityOmitsPrivateFields(t *testing.T, raw []byte, secrets ...string) {
	t.Helper()
	body := string(raw)
	for _, secret := range secrets {
		assert.NotContains(t, body, secret)
	}
	assert.NotContains(t, body, "hidden-velocity-author")

	found := map[string]bool{}
	var decoded any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	collectJSONKeys(decoded, found)
	for _, key := range []string{
		"repository",
		"people",
		"peopleTotal",
		"peopleHasMore",
		"peopleSyncedAt",
		"peopleSyncPending",
		"yesterday",
		"medianCycleHours",
	} {
		assert.False(t, found[key], "response includes %s", key)
	}
}

func collectJSONKeys(value any, found map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			found[key] = true
			collectJSONKeys(child, found)
		}
	case []any:
		for _, child := range typed {
			collectJSONKeys(child, found)
		}
	}
}

func ptr(value string) *string {
	return &value
}

type adminVelocityBody struct {
	Factories []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"factories"`
	FactoryID       string `json:"factoryId"`
	PeriodDays      int    `json:"periodDays"`
	HasPeopleCohort bool   `json:"hasPeopleCohort"`
	Totals          struct {
		SuperplaneMerged int `json:"superplaneMerged"`
		PeopleMerged     int `json:"peopleMerged"`
		TasksClosed      int `json:"tasksClosed"`
	} `json:"totals"`
	Points []struct {
		Day string `json:"day"`
	} `json:"points"`
	IntakeSources []struct {
		Key    string `json:"key"`
		Label  string `json:"label"`
		Merged int    `json:"merged"`
	} `json:"intakeSources"`
	Automations []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Runs int    `json:"runs"`
	} `json:"automations"`
}
