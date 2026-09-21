package factories

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	factoryevents "github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestGitHubMergeabilityWebhookRef(t *testing.T) {
	t.Run("reads a pull request synchronize event", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("pull_request", []byte(`{
			"action": "synchronize",
			"repository": {"full_name": "acme/app"},
			"pull_request": {"number": 42, "head": {"sha": "abc123"}}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Equal(t, []int64{42}, numbers)
		assert.Equal(t, "abc123", sha)
	})

	t.Run("reads a completed check run", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("check_run", []byte(`{
			"repository": {"full_name": "acme/app"},
			"check_run": {
				"head_sha": "def456",
				"pull_requests": [{"number": 7}, {"number": 7}]
			}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Equal(t, []int64{7}, numbers)
		assert.Equal(t, "def456", sha)
	})

	t.Run("reads a commit status", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("status", []byte(`{
			"sha": "fff999",
			"repository": {"full_name": "acme/app"}
		}`))
		assert.Equal(t, "acme/app", repository)
		assert.Empty(t, numbers)
		assert.Equal(t, "fff999", sha)
	})

	t.Run("ignores unrelated events", func(t *testing.T) {
		repository, numbers, sha := githubMergeabilityWebhookRef("issues", []byte(`{"repository":{"full_name":"acme/app"}}`))
		assert.Equal(t, "", repository)
		assert.Nil(t, numbers)
		assert.Equal(t, "", sha)
	})
}

func TestIsGitHubFactoryMergeabilityEvent(t *testing.T) {
	assert.True(t, IsGitHubFactoryMergeabilityEvent("check_run"))
	assert.True(t, IsGitHubFactoryMergeabilityEvent("ping"))
	assert.True(t, IsGitHubFactoryMergeabilityEvent("PULL_REQUEST"))
	assert.False(t, IsGitHubFactoryMergeabilityEvent("issues"))
	assert.False(t, IsGitHubFactoryMergeabilityEvent(""))
}

func TestEnsureGitHubFactoryMergeabilityWebhook(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)

	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))

	webhooks, err := models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, models.WebhookStatePending, webhooks[0].State)
	assert.True(t, isFactoryMergeabilityWebhook(webhooks[0].Configuration.Data()))
	assert.Equal(t, "acme/app", factoryMergeabilityWebhookRepository(webhooks[0].Configuration.Data()))

	firstID := webhooks[0].ID
	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, firstID, webhooks[0].ID)

	require.NoError(t, db.Model(&webhooks[0]).Update("state", models.WebhookStateFailed).Error)
	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/app"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 1)
	assert.Equal(t, firstID, webhooks[0].ID)
	assert.Equal(t, models.WebhookStatePending, webhooks[0].State)
	assert.Equal(t, "acme/app", factoryMergeabilityWebhookRepository(webhooks[0].Configuration.Data()))

	require.NoError(t, ensureGitHubFactoryMergeabilityWebhook(t.Context(), db, r.Encryptor, integration, "acme/other"))
	webhooks, err = models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	require.Len(t, webhooks, 2)
	repositories := map[string]struct{}{}
	for _, hook := range webhooks {
		assert.True(t, isFactoryMergeabilityWebhook(hook.Configuration.Data()))
		repositories[factoryMergeabilityWebhookRepository(hook.Configuration.Data())] = struct{}{}
	}
	assert.Equal(t, map[string]struct{}{"acme/app": {}, "acme/other": {}}, repositories)
}

func TestVerifyGitHubFactoryMergeabilitySignature(t *testing.T) {
	r := support.Setup(t)
	webhookID := uuid.New()
	secret := []byte("webhook-secret")
	encrypted, err := r.Encryptor.Encrypt(t.Context(), secret, []byte(webhookID.String()))
	require.NoError(t, err)
	webhook := &models.Webhook{ID: webhookID, Secret: encrypted}
	body := []byte(`{"ok":true}`)

	t.Run("accepts a matching GitHub signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+crypto.Sign(secret, body))
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, headers, body)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("rejects a missing signature", func(t *testing.T) {
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, http.Header{}, body)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, code)
	})

	t.Run("rejects a mismatched signature", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=deadbeef")
		code, err := VerifyGitHubFactoryMergeabilitySignature(t.Context(), r.Encryptor, webhook, headers, body)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, code)
	})
}

func TestRefreshFactoryPullRequestMergeabilityAtRecordTime(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/other", 91)
	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		support.RandomName("github"),
		map[string]any{},
	)
	require.NoError(t, err)
	require.NoError(t, db.Model(integration).Update("state", models.IntegrationStateReady).Error)
	vcsID := integration.ID.String()
	appRepo := "acme/app"
	require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{
		VCSIntegrationID: &vcsID,
		AppRepository:    &appRepo,
	}))

	const headSHA = "abc123def456"
	stubFactoryGitHub(t, mergeableFactoryGitHub(headSHA))

	require.NoError(t, refreshFactoryPullRequestMergeabilityByID(
		t.Context(),
		IntakeDependencies{Encryptor: r.Encryptor},
		factory.OrganizationID,
		factory.ID,
		pullRequest.ID,
	))

	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.True(t, stored.Mergeable)
	assert.Equal(t, headSHA, stored.MergeableHeadSHA)
	assert.Empty(t, stored.MergeBlockedReason)

	webhooks, err := models.ListIntegrationWebhooks(db, integration.ID)
	require.NoError(t, err)
	repositories := map[string]struct{}{}
	for _, hook := range webhooks {
		repositories[factoryMergeabilityWebhookRepository(hook.Configuration.Data())] = struct{}{}
	}
	assert.Contains(t, repositories, "acme/other")
}

func TestRefreshFactoryPullRequestMergeabilityAfterRunFinishes(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 92)
	const headSHA = "abc123def456"
	stubFactoryGitHub(t, mergeableFactoryGitHub(headSHA))
	require.NoError(t, pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable:      true,
		HeadSHA:        headSHA,
		AllowedMethods: "SQUASH,MERGE,REBASE",
	}))

	grantExclusivePullRequestAccess(t, db, r, factory, &pb.FactoryPullRequest{Id: pullRequest.ID.String()})
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	require.NotNil(t, pullRequest.ActiveMutationRunID)
	assert.True(t, pullRequest.Mergeable)
	assert.False(t, serializeFactoryPullRequest(pullRequest, 1, nil, nil, nil).GetMergeable())

	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	blocked, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.True(t, blocked.Mergeable)
	assert.Equal(t, headSHA, blocked.MergeableHeadSHA)
	assert.False(t, serializeFactoryPullRequest(blocked, 1, nil, nil, nil).GetMergeable())

	require.NoError(t, db.Model(pullRequest).Update("active_mutation_run_id", nil).Error)
	pullRequest.ActiveMutationRunID = nil

	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	assert.True(t, stored.Mergeable)
	assert.Equal(t, headSHA, stored.MergeableHeadSHA)
	assert.True(t, serializeFactoryPullRequest(stored, 1, nil, nil, nil).GetMergeable())
}

func TestListOpenGitHubFactoryPullRequestsForWebhookMatching(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 93)
	const headSHA = "headcommitsha"
	require.NoError(t, pullRequest.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		Mergeable: true,
		HeadSHA:   headSHA,
	}))

	closedOrder, err := factory.CreateWorkOrder(db, "Closed", "", &r.User, nil, nil)
	require.NoError(t, err)
	closed, err := closedOrder.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL:   "https://github.com/acme/app/pull/94",
		State: models.FactoryPullRequestStateClosed,
	})
	require.NoError(t, err)
	require.NoError(t, closed.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		HeadSHA: headSHA,
	}))

	otherRepoOrder, err := factory.CreateWorkOrder(db, "Other repo", "", &r.User, nil, nil)
	require.NoError(t, err)
	otherRepo, err := otherRepoOrder.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: "https://github.com/acme/other/pull/93",
	})
	require.NoError(t, err)
	require.NoError(t, otherRepo.SetMergeability(db, models.FactoryPullRequestMergeabilitySnapshot{
		HeadSHA: headSHA,
	}))

	otherOrg, err := models.CreateOrganization(support.RandomName("org"), "")
	require.NoError(t, err)

	tests := []struct {
		name           string
		organizationID uuid.UUID
		repository     string
		numbers        []int64
		sha            string
		wantIDs        []uuid.UUID
	}{
		{
			name:           "matches a head commit with no revision row",
			organizationID: r.Organization.ID,
			repository:     "acme/app",
			sha:            headSHA,
			wantIDs:        []uuid.UUID{pullRequest.ID},
		},
		{
			name:           "matches by number or head sha",
			organizationID: r.Organization.ID,
			repository:     "acme/app",
			numbers:        []int64{93},
			sha:            "missingsha",
			wantIDs:        []uuid.UUID{pullRequest.ID},
		},
		{
			name:           "ignores a closed pull request with the same head sha",
			organizationID: r.Organization.ID,
			repository:     "acme/app",
			sha:            headSHA,
			wantIDs:        []uuid.UUID{pullRequest.ID},
		},
		{
			name:           "ignores another repository",
			organizationID: r.Organization.ID,
			repository:     "acme/other",
			sha:            headSHA,
			wantIDs:        []uuid.UUID{otherRepo.ID},
		},
		{
			name:           "ignores another organization",
			organizationID: otherOrg.ID,
			repository:     "acme/app",
			numbers:        []int64{93},
			sha:            headSHA,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := models.ListOpenGitHubFactoryPullRequestsForWebhook(
				db,
				tc.organizationID,
				tc.repository,
				tc.numbers,
				tc.sha,
			)
			require.NoError(t, err)
			gotIDs := make([]uuid.UUID, 0, len(matched))
			for _, item := range matched {
				gotIDs = append(gotIDs, item.ID)
			}
			assert.ElementsMatch(t, tc.wantIDs, gotIDs)
		})
	}
}

func TestRefreshFactoryPullRequestMergeabilityPublishesOnlyOnStoredChange(t *testing.T) {
	r := support.Setup(t)
	db := database.Conn()
	factory, pullRequest := createOpenGitHubFactoryPullRequest(t, db, r, "acme/app", 95)
	const headSHA = "abc123def456"
	stubFactoryGitHub(t, mergeableFactoryGitHub(headSHA))

	published := 0
	original := publishFactoryWorkOrderUpdated
	publishFactoryWorkOrderUpdated = func(string, string, string) error {
		published++
		return nil
	}
	t.Cleanup(func() { publishFactoryWorkOrderUpdated = original })

	grantExclusivePullRequestAccess(t, db, r, factory, &pb.FactoryPullRequest{Id: pullRequest.ID.String()})
	pullRequest, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{ID: pullRequest.ID})
	require.NoError(t, err)
	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, 0, published)

	require.NoError(t, db.Model(pullRequest).Update("active_mutation_run_id", nil).Error)
	pullRequest.ActiveMutationRunID = nil
	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, 1, published)

	require.NoError(t, refreshFactoryPullRequestMergeability(t.Context(), db, IntakeDependencies{}, factory, pullRequest))
	assert.Equal(t, 1, published)
}

func createOpenGitHubFactoryPullRequest(
	t *testing.T,
	db *gorm.DB,
	r *support.ResourceRegistry,
	repository string,
	number int64,
) (*models.Factory, *models.FactoryPullRequest) {
	t.Helper()
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
	require.NoError(t, err)
	pullRequest, err := order.CreatePullRequest(db, models.FactoryPullRequestParams{
		URL: fmt.Sprintf("https://github.com/%s/pull/%d", repository, number),
	})
	require.NoError(t, err)
	return factory, pullRequest
}

func stubFactoryGitHub(t *testing.T, api factoryGitHubAPI) {
	t.Helper()
	original := newFactoryGitHubAPI
	newFactoryGitHubAPI = func(*gorm.DB, IntakeDependencies, *models.Factory) (factoryGitHubAPI, error) {
		return api, nil
	}
	t.Cleanup(func() { newFactoryGitHubAPI = original })
}

func mergeableFactoryGitHub(headSHA string) *fakeFactoryGitHub {
	combined, checks := successChecks()
	return &fakeFactoryGitHub{
		pullRequest: mergeableGitHubPullRequest(headSHA),
		combined:    combined,
		checkRuns:   checks,
		repository:  allMethodsRepository(),
	}
}

func TestRefreshFactoryPullRequestMergeabilityFromGitHubEvent_ClosesWorkOrder(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	setupOpenOrder := func(t *testing.T, repository string, number int64) (
		*models.Factory,
		*models.FactoryWorkOrder,
		*models.Webhook,
	) {
		t.Helper()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		order, err := factory.CreateWorkOrder(db, "Tracked", "", &r.User, nil, nil)
		require.NoError(t, err)
		_, err = order.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
			ToState: models.FactoryWorkOrderStateOpen,
			Actor:   &r.User,
		})
		require.NoError(t, err)
		_, err = order.SetStatusNote(db, models.FactoryWorkOrderStatusNoteParams{
			Key:      "pr-review",
			Headline: "Waiting for user review",
		})
		require.NoError(t, err)
		_, err = order.CreatePullRequest(db, models.FactoryPullRequestParams{
			URL:        fmt.Sprintf("https://github.com/%s/pull/%d", repository, number),
			Repository: repository,
			Number:     number,
			State:      models.FactoryPullRequestStateOpen,
		})
		require.NoError(t, err)
		integration, err := models.CreateIntegration(
			uuid.New(),
			r.Organization.ID,
			"github",
			support.RandomName("github"),
			map[string]any{},
		)
		require.NoError(t, err)
		return factory, order, &models.Webhook{AppInstallationID: &integration.ID}
	}

	closedPayload := func(repository string, number int64, merged bool) []byte {
		closedAt := time.Now().UTC().Format(time.RFC3339)
		mergedJSON := "false"
		mergedAt := "null"
		if merged {
			mergedJSON = "true"
			mergedAt = fmt.Sprintf("%q", closedAt)
		}
		return []byte(fmt.Sprintf(`{
			"action": "closed",
			"repository": {"full_name": %q},
			"pull_request": {
				"number": %d,
				"merged": %s,
				"merged_at": %s,
				"closed_at": %q,
				"head": {"sha": "abc123"}
			}
		}`, repository, number, mergedJSON, mergedAt, closedAt))
	}

	deliver := func(webhook *models.Webhook, body []byte) {
		RefreshFactoryPullRequestMergeabilityFromGitHubEvent(
			t.Context(),
			IntakeDependencies{},
			webhook,
			"pull_request",
			body,
		)
	}

	countStatusEvents := func(t *testing.T, order *models.FactoryWorkOrder) int {
		t.Helper()
		events, err := order.ListEvents(db, 0, nil)
		require.NoError(t, err)
		count := 0
		for _, event := range events {
			if event.Type == factoryevents.EventTypeOrderStatusUpdated {
				count++
			}
		}
		return count
	}

	reload := func(t *testing.T, factory *models.Factory, orderID uuid.UUID) *models.FactoryWorkOrder {
		t.Helper()
		reloaded, err := factory.FindWorkOrder(db, orderID)
		require.NoError(t, err)
		return reloaded
	}

	t.Run("closes a merged pull request as completed and clears the review note", func(t *testing.T) {
		factory, order, webhook := setupOpenOrder(t, "acme/app", 61)
		deliver(webhook, closedPayload("acme/app", 61, true))

		reloaded := reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
		notes, err := reloaded.StatusNotes()
		require.NoError(t, err)
		assert.Empty(t, notes)

		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     61,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestStateMerged, stored.State)
		require.NotNil(t, stored.MergedAt)
	})

	t.Run("closes a pull request without a merge as rejected", func(t *testing.T) {
		factory, order, webhook := setupOpenOrder(t, "acme/app", 62)
		deliver(webhook, closedPayload("acme/app", 62, false))

		reloaded := reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultRejected, reloaded.Result)

		stored, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{
			Provider:   models.FactoryPullRequestProviderGitHub,
			Repository: "acme/app",
			Number:     62,
		})
		require.NoError(t, err)
		assert.Equal(t, models.FactoryPullRequestStateClosed, stored.State)
		require.NotNil(t, stored.ClosedAt)
	})

	t.Run("repeats do nothing and add no second status event", func(t *testing.T) {
		factory, order, webhook := setupOpenOrder(t, "acme/app", 63)
		body := closedPayload("acme/app", 63, true)
		deliver(webhook, body)

		reloaded := reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		afterFirst := countStatusEvents(t, reloaded)

		deliver(webhook, body)
		reloaded = reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
		assert.Equal(t, afterFirst, countStatusEvents(t, reloaded))
	})

	t.Run("concurrent closed deliveries record one close event", func(t *testing.T) {
		factory, order, webhook := setupOpenOrder(t, "acme/app", 65)
		body := closedPayload("acme/app", 65, true)
		before := countStatusEvents(t, order)

		var started sync.WaitGroup
		started.Add(2)
		var wg sync.WaitGroup
		wg.Add(2)
		deliverOnce := func() {
			defer wg.Done()
			started.Done()
			started.Wait()
			deliver(webhook, body)
		}
		go deliverOnce()
		go deliverOnce()
		wg.Wait()

		reloaded := reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateClosed, reloaded.State)
		assert.Equal(t, models.FactoryWorkOrderResultCompleted, reloaded.Result)
		assert.Equal(t, before+1, countStatusEvents(t, reloaded))
	})

	t.Run("ignores a repository with no matching pull request", func(t *testing.T) {
		factory, order, webhook := setupOpenOrder(t, "acme/app", 64)
		deliver(webhook, closedPayload("acme/other", 64, true))

		reloaded := reload(t, factory, order.ID)
		assert.Equal(t, models.FactoryWorkOrderStateOpen, reloaded.State)
		assert.Empty(t, reloaded.Result)
		notes, err := reloaded.StatusNotes()
		require.NoError(t, err)
		require.Len(t, notes, 1)
		assert.Equal(t, "pr-review", notes[0].Key)
	})
}
