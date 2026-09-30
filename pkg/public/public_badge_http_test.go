package public

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestPublicBadge_ReturnsVelocitySVG(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))
	require.NotNil(t, factoryModel.PublicBadgeToken)

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	seedBadgePullRequest(t, factoryModel, 1, models.FactoryPullRequestStateMerged, today, 4_200_000, 0)
	seedBadgePullRequest(t, factoryModel, 2, models.FactoryPullRequestStateMerged, today.AddDate(0, 0, -20), 0, 0)
	seedBadgePeopleMerge(t, r, factoryModel, "example/repo", "Quinlan Private", today.Add(-time.Hour))

	token := *factoryModel.PublicBadgeToken
	body := getBadge(t, server, token, "period=30&size=large")
	assert.Equal(t, http.StatusOK, body.status)
	assert.Contains(t, body.contentType, "image/svg+xml")
	assert.Equal(t, badgeCacheControl, body.cacheControl)
	assert.NotContains(t, body.svg, "Quinlan")
	assert.NotContains(t, body.svg, "987654321")

	velocity := describeVelocity(t, r, factoryModel, 30, "example/repo")
	assert.Contains(t, body.svg, fmt.Sprintf("%d%%", velocity.Totals.GetSuperplaneSharePct()))
	assert.Contains(t, body.svg, fmt.Sprintf("%d PRs merged via SuperPlane · last 30 days", velocity.Totals.GetSuperplaneMerged()))

	fallback := getBadge(t, server, token, "period=15&size=large")
	assert.Contains(t, fallback.svg, "last 30 days")
	assert.Contains(t, fallback.svg, fmt.Sprintf("%d PRs merged via SuperPlane · last 30 days", velocity.Totals.GetSuperplaneMerged()))

	week := describeVelocity(t, r, factoryModel, 7, "example/repo")
	weekBadge := getBadge(t, server, token, "period=7&size=large")
	assert.Contains(t, weekBadge.svg, "last 7 days")
	assert.Contains(t, weekBadge.svg, fmt.Sprintf("%d PRs merged via SuperPlane · last 7 days", week.Totals.GetSuperplaneMerged()))
	assert.NotEqual(t, velocity.Totals.GetSuperplaneMerged(), week.Totals.GetSuperplaneMerged())

	small := getBadge(t, server, token, "size=banana")
	assert.Contains(t, small.svg, "% PRs · 30d")
	assert.NotContains(t, small.svg, "of merged PRs via SuperPlane")
}

func TestPublicBadge_AppliesTheThemeAndAccentFromTheURL(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))
	token := *factoryModel.PublicBadgeToken

	themed := getBadge(t, server, token, "size=large&theme=github_light")
	assert.Equal(t, http.StatusOK, themed.status)
	assert.Contains(t, themed.svg, "#ffffff")
	assert.Contains(t, themed.svg, "#0969da")

	accented := getBadge(t, server, token, "size=large&accent=F59E0B")
	assert.Contains(t, accented.svg, "#f59e0b")

	// An unusable theme or accent still renders the default card.
	fallback := getBadge(t, server, token, "size=large&theme=banana&accent=red")
	assert.Equal(t, http.StatusOK, fallback.status)
	assert.Contains(t, fallback.svg, "#10b981")
}

func TestPublicBadge_HidesCostUnlessTheSwitchIsOn(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	seedBadgePullRequest(t, factoryModel, 1, models.FactoryPullRequestStateMerged, today, 4_200_000, 0)
	seedBadgePullRequest(t, factoryModel, 2, models.FactoryPullRequestStateMerged, today, 4_200_000, 0)

	token := *factoryModel.PublicBadgeToken
	hidden := getBadge(t, server, token, "size=large&cost=1&show_cost=true")
	assert.NotContains(t, hidden.svg, "$")
	wideHidden := getBadge(t, server, token, "size=wide&show_cost=true")
	assert.NotContains(t, wideHidden.svg, "$")

	require.NoError(t, factoryModel.UpdatePublicBadgeShowCost(database.DB(t.Context()), true))
	velocity := describeVelocity(t, r, factoryModel, 30, "example/repo")
	require.Equal(t, int32(2), velocity.Totals.GetSuperplaneMerged())
	require.Equal(t, int64(840), velocity.Totals.GetCostCents())

	shown := getBadge(t, server, token, "size=large")
	assert.Contains(t, shown.svg, "$4.20 per merged PR")
	assert.NotContains(t, shown.svg, "$8.40")
	wide := getBadge(t, server, token, "size=wide&cost=0")
	assert.Contains(t, wide.svg, "$4.20 per merged PR")
	small := getBadge(t, server, token, "size=small")
	assert.NotContains(t, small.svg, "$")
	assert.NotContains(t, small.svg, "per merged PR")
}

func TestPublicBadge_OmitsShareWhilePeopleSyncIsPending(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	seedBadgePullRequest(t, factoryModel, 1, models.FactoryPullRequestStateMerged, today, 0, 0)
	seedBadgePullRequest(t, factoryModel, 2, models.FactoryPullRequestStateClosed, today, 0, 0)

	velocity := describeVelocity(t, r, factoryModel, 30, "example/repo")
	assert.False(t, velocity.HasPeopleCohort)
	assert.Equal(t, int32(0), velocity.Totals.GetSuperplaneSharePct())
	assert.Greater(t, velocity.Totals.GetSuperplaneMerged(), int32(0))

	token := *factoryModel.PublicBadgeToken
	for _, size := range []string{"small", "large", "wide"} {
		body := getBadge(t, server, token, "size="+size)
		assert.NotContains(t, body.svg, ">100%<", size)
		assert.NotContains(t, body.svg, "100% of merged PRs", size)
		assert.NotContains(t, body.svg, "100% PRs", size)
		assert.NotContains(t, body.svg, "Manual work", size)
	}
	small := getBadge(t, server, token, "size=small")
	assert.Contains(t, small.svg, "1 PRs · 30d")
	assert.NotContains(t, small.svg, "%")
}

func TestPublicBadge_ShowsFullShareAfterSyncWithNoManualMerges(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	seedBadgePullRequest(t, factoryModel, 1, models.FactoryPullRequestStateMerged, today, 0, 0)
	markBadgePeopleSyncComplete(t, factoryModel, "example/repo")

	velocity := describeVelocity(t, r, factoryModel, 30, "example/repo")
	require.True(t, velocity.HasPeopleCohort)
	require.Equal(t, int32(100), velocity.Totals.GetSuperplaneSharePct())

	body := getBadge(t, server, *factoryModel.PublicBadgeToken, "size=large")
	assert.Contains(t, body.svg, "100%")
}

func markBadgePeopleSyncComplete(t *testing.T, factoryModel *models.Factory, repo string) {
	t.Helper()
	db := database.DB(t.Context())
	from := time.Now().AddDate(0, 0, -90)
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factoryModel.ID, from, time.Now().Add(time.Hour), nil))
	sync, err := models.ClaimFactoryVelocitySync(db, factoryModel.ID, time.Now())
	require.NoError(t, err)
	require.NotNil(t, sync)
	require.NoError(t, sync.RecordSuccess(db, repo, time.Now(), from))
}

func TestPublicBadge_NotFoundWhenDisabledOrUnknown(t *testing.T) {
	r := support.Setup(t)
	server := newPublicTestServer(t, r)
	factoryModel := newBadgeFactory(t, r, "example/repo")
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), true))
	token := *factoryModel.PublicBadgeToken
	require.NoError(t, factoryModel.UpdatePublicBadgeEnabled(database.DB(t.Context()), false))

	disabledRec := doBadge(t, server, token, "")
	assert.Equal(t, http.StatusNotFound, disabledRec.Code)
	assert.Equal(t, badgeCacheControl, disabledRec.Header().Get("Cache-Control"))
	assert.NotContains(t, disabledRec.Body.String(), factoryModel.Name)

	unknown := getBadgeStatus(t, server, "not-a-real-token", "")
	assert.Equal(t, http.StatusNotFound, unknown)
}

type badgeResponse struct {
	status       int
	contentType  string
	cacheControl string
	svg          string
}

func newPublicTestServer(t *testing.T, r *support.ResourceRegistry) *Server {
	t.Helper()
	signer := jwt.NewSigner("test")
	server, err := NewServer(
		r.Encryptor,
		r.Registry,
		signer,
		support.NewOIDCProvider(),
		"",
		"http://localhost",
		"http://localhost",
		"test",
		"/app/templates",
		r.AuthService,
		false,
	)
	require.NoError(t, err)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	return server
}

func newBadgeFactory(t *testing.T, r *support.ResourceRegistry, repo string) *models.Factory {
	t.Helper()
	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, factoryModel.UpdateOnboarding(database.DB(t.Context()), models.FactoryOnboardingPatch{AppRepository: &repo}))
	return factoryModel
}

func seedBadgePullRequest(
	t *testing.T,
	factoryModel *models.Factory,
	number int,
	state string,
	at time.Time,
	modelMicros, computeMicros int64,
) {
	t.Helper()
	db := database.DB(t.Context())
	order, err := factoryModel.CreateWorkOrder(db, "PR order", "", nil, nil, nil)
	require.NoError(t, err)
	params := models.FactoryPullRequestParams{
		URL:   fmt.Sprintf("https://github.com/example/repo/pull/%d", number),
		State: state,
	}
	if state == models.FactoryPullRequestStateMerged {
		params.MergedAt = &at
	}
	if state == models.FactoryPullRequestStateClosed {
		params.ClosedAt = &at
	}
	_, err = order.CreatePullRequest(db, params)
	require.NoError(t, err)
	if modelMicros == 0 && computeMicros == 0 {
		return
	}
	event := models.WorkspaceUsageEvent{
		ID:               uuid.New(),
		OrganizationID:   factoryModel.OrganizationID,
		FactoryID:        &factoryModel.ID,
		WorkOrderID:      &order.ID,
		CanvasRunID:      uuid.New(),
		NodeExecutionID:  uuid.New(),
		NodeID:           "prompt",
		Provider:         models.UsageProviderAnthropic,
		Model:            "claude-sonnet-4-6",
		UsageKind:        models.UsageKindModel,
		TotalTokens:      987654321,
		CostMicros:       modelMicros,
		FundingSource:    models.UsageFundingSourceBYOK,
		Currency:         "usd",
		PriceBookVersion: "test",
		IdempotencyKey:   uuid.NewString(),
		OccurredAt:       at,
	}
	require.NoError(t, db.Create(&event).Error)
	if computeMicros == 0 {
		return
	}
	compute := event
	compute.ID = uuid.New()
	compute.UsageKind = models.UsageKindCompute
	compute.Provider = models.UsageProviderRunner
	compute.CostMicros = computeMicros
	compute.TotalTokens = 0
	compute.IdempotencyKey = uuid.NewString()
	require.NoError(t, db.Create(&compute).Error)
}

func seedBadgePeopleMerge(t *testing.T, r *support.ResourceRegistry, factoryModel *models.Factory, repo, name string, at time.Time) {
	t.Helper()
	db := database.DB(t.Context())
	merge := models.NewFactoryVelocityRepositoryMerge(r.Organization.ID, factoryModel.ID, repo, 80, models.FactoryVelocityMergeSourcePeople, at)
	merge.AuthorName = name
	merge.AuthorLogin = "quinlan"
	from := time.Now().AddDate(0, 0, -90)
	to := time.Now().Add(time.Hour)
	require.NoError(t, models.ReplaceFactoryVelocityRepositoryMerges(db, factoryModel.ID, from, to, []models.FactoryVelocityRepositoryMerge{merge}))
	sync, err := models.ClaimFactoryVelocitySync(db, factoryModel.ID, time.Now())
	require.NoError(t, err)
	require.NoError(t, sync.RecordSuccess(db, repo, time.Now(), from))
}

func describeVelocity(t *testing.T, r *support.ResourceRegistry, factoryModel *models.Factory, period int, repo string) *pb.DescribeFactoryVelocityResponse {
	t.Helper()
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	resp, err := factoryactions.DescribeFactoryVelocity(ctx, r.Organization.ID.String(), &pb.DescribeFactoryVelocityRequest{
		FactoryId:  factoryModel.ID.String(),
		PeriodDays: int32(period),
		Repository: repo,
	})
	require.NoError(t, err)
	return resp
}

func getBadge(t *testing.T, server *Server, token, query string) badgeResponse {
	t.Helper()
	rec := doBadge(t, server, token, query)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	return badgeResponse{
		status:       rec.Code,
		contentType:  rec.Header().Get("Content-Type"),
		cacheControl: rec.Header().Get("Cache-Control"),
		svg:          rec.Body.String(),
	}
}

func getBadgeStatus(t *testing.T, server *Server, token, query string) int {
	t.Helper()
	return doBadge(t, server, token, query).Code
}

func doBadge(t *testing.T, server *Server, token, query string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/api/v1/public/badges/" + token + ".svg"
	if query != "" {
		path += "?" + query
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	return rec
}
