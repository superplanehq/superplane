package public

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

type analyticsRecorder struct {
	mu     sync.Mutex
	events []productanalytics.Event
	groups []productanalytics.OrganizationGroupIdentity
}

func (r *analyticsRecorder) Capture(event productanalytics.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *analyticsRecorder) IdentifyOrganizationGroup(identity productanalytics.OrganizationGroupIdentity) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups = append(r.groups, identity)
}

func (r *analyticsRecorder) Close() error { return nil }

func (r *analyticsRecorder) Enabled() bool { return true }

func (r *analyticsRecorder) snapshot() []productanalytics.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]productanalytics.Event, len(r.events))
	copy(copied, r.events)
	return copied
}

func (r *analyticsRecorder) groupIdentities() []productanalytics.OrganizationGroupIdentity {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]productanalytics.OrganizationGroupIdentity, len(r.groups))
	copy(copied, r.groups)
	return copied
}

func useAnalytics(t *testing.T) *analyticsRecorder {
	t.Helper()
	recorder := &analyticsRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})
	return recorder
}

func TestAccountDeletionAnalyticsWithoutOrganizations(t *testing.T) {
	r := support.Setup(t)
	account, err := models.CreateAccount("No workspace", support.RandomName("account")+"@example.com")
	require.NoError(t, err)
	r.Account = account
	server, _, token := setupTestServer(r, t)
	recorder := useAnalytics(t)
	body, err := json.Marshal(map[string]string{"email": account.Email})
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
	request.AddCookie(&http.Cookie{Name: "account_token", Value: token})
	response := httptest.NewRecorder()
	server.Router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
	events := recorder.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, productanalytics.EventAccountDeleted, events[0].Name)
	assert.Empty(t, events[0].OrganizationID)
	assert.Empty(t, events[0].Properties["deleted_organization_ids"])
	assert.Equal(t, account.ID.String(), events[0].DistinctID)
	assert.Equal(t, productanalytics.StableEventID(productanalytics.EventAccountDeleted, account.ID.String()), events[0].EventID)
	deletedAt, err := time.Parse(time.RFC3339Nano, events[0].Properties["deleted_at"].(string))
	require.NoError(t, err)
	assert.WithinDuration(t, time.Now(), deletedAt, time.Minute)
}

func TestAccountDeletionAnalytics_ConcurrentOwnerPromotionKeepsOrganization(t *testing.T) {
	for _, source := range []string{"self_service", "admin"} {
		t.Run(source, func(t *testing.T) {
			r := support.Setup(t)
			db := database.DB(t.Context())
			require.NoError(t, db.Model(r.Account).Update("installation_admin", false).Error)
			otherAccount, err := models.CreateAccount("Other Owner", uuid.NewString()+"@example.com")
			require.NoError(t, err)
			otherUser, err := models.CreateUserInTransaction(db, r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
			require.NoError(t, err)
			server, account, _ := setupTestServer(r, t)
			recorder := useAnalytics(t)
			body, err := json.Marshal(map[string]string{"email": account.Email})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodDelete, "/account", bytes.NewReader(body))
			request = request.WithContext(accountContext(account))
			handler := server.deleteAccount
			if source == "admin" {
				require.NoError(t, models.PromoteToInstallationAdmin(otherAccount.ID.String()))
				request = request.WithContext(accountContext(otherAccount))
				request = mux.SetURLVars(request, map[string]string{"accountId": account.ID.String()})
				handler = server.adminDeleteAccount
			}

			// Hold the organization lock while deletion reads its candidate list.
			// Commit another owner only after deletion reaches the lock.
			promotion := db.Begin()
			require.NoError(t, promotion.Error)
			defer promotion.Rollback()
			_, err = models.LockOrganization(promotion, r.Organization.ID)
			require.NoError(t, err)
			var promotionPID int
			require.NoError(t, promotion.Raw("SELECT pg_backend_pid()").Scan(&promotionPID).Error)
			response := httptest.NewRecorder()
			deletion := run(func() error {
				handler(response, request)
				return nil
			})
			require.Eventually(t, func() bool {
				var waiting bool
				require.NoError(t, db.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE ? = ANY(pg_blocking_pids(pid)))", promotionPID).Scan(&waiting).Error)
				return waiting
			}, 10*time.Second, 10*time.Millisecond)
			require.NoError(t, models.SetUserIsOwner(promotion, otherUser.ID, true))
			require.NoError(t, promotion.Commit().Error)
			require.NoError(t, finished(t, deletion))

			if source == "admin" {
				require.Equal(t, http.StatusOK, response.Code)
				var result struct {
					DeletedOrganizationIDs []string `json:"deleted_organization_ids"`
				}
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				assert.Empty(t, result.DeletedOrganizationIDs)
			} else {
				require.Equal(t, http.StatusNoContent, response.Code)
			}
			_, err = models.FindOrganizationByID(r.Organization.ID.String())
			require.NoError(t, err, "the promoted owner keeps the organization")
			_, err = models.FindAccountByID(account.ID.String())
			require.ErrorIs(t, err, gorm.ErrRecordNotFound)
			events := recorder.snapshot()
			require.Len(t, events, 1, "a retained organization must not emit organization_deleted")
			assert.Equal(t, productanalytics.EventAccountDeleted, events[0].Name)
			assert.Equal(t, source, events[0].Properties["deletion_source"])
			assert.Empty(t, events[0].Properties["deleted_organization_ids"])
		})
	}
}

func TestOrganizationCreationAnalytics(t *testing.T) {
	for _, tc := range []struct {
		label    string
		name     string
		email    string
		identity productanalytics.Properties
	}{
		{"email and name", "Metrics User", "metrics@example.com", productanalytics.Properties{"email": "metrics@example.com", "name": "Metrics User"}},
		{"blank name", "", "metrics@example.com", productanalytics.Properties{"email": "metrics@example.com"}},
		{"blank email", "Metrics User", "", productanalytics.Properties{"name": "Metrics User"}},
		{"blank identity", " \t", " \t", productanalytics.Properties{}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			recorder := useAnalytics(t)
			require.NoError(t, database.TruncateTables())

			account, err := models.CreateAccount(tc.name, tc.email)
			require.NoError(t, err)
			if tc.email == "metrics@example.com" {
				assert.Equal(t, "metrics@example.com", account.Email)
			}
			signer := jwt.NewSigner("test")

			authService, err := authorization.NewAuthService()
			require.NoError(t, err)
			encryptor := &crypto.NoOpEncryptor{}
			componentRegistry, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
			require.NoError(t, err)
			server, err := NewServer(encryptor, componentRegistry, signer, support.NewOIDCProvider(), "", "localhost", "", "test", "/app/templates", authService, false)
			require.NoError(t, err)

			postOrganization := func(body []byte) *httptest.ResponseRecorder {
				request := httptest.NewRequest(http.MethodPost, "/organizations", bytes.NewReader(body))
				request = request.WithContext(accountContext(account))
				response := httptest.NewRecorder()
				server.createOrganization(response, request)
				return response
			}

			t.Run("rejected creation emits nothing", func(t *testing.T) {
				response := postOrganization([]byte(`{"name":""}`))
				assert.Equal(t, http.StatusBadRequest, response.Code)
				assert.Empty(t, recorder.snapshot())
				assert.Empty(t, recorder.groupIdentities())
			})

			t.Run("later organization creation emits one event", func(t *testing.T) {
				body, err := json.Marshal(OrganizationCreationRequest{Name: "Metrics Organization"})
				require.NoError(t, err)
				response := postOrganization(body)
				require.Equal(t, http.StatusOK, response.Code)

				events := recorder.snapshot()
				require.Len(t, events, 1)
				assert.Equal(t, productanalytics.EventOrganizationCreated, events[0].Name)
				assert.Equal(t, "workspace_switcher", events[0].Properties["creation_path"])
				assert.Equal(t, account.ID.String(), events[0].DistinctID)
				assert.Equal(t, productanalytics.ActorUser, events[0].ActorType)
				assert.Equal(t, productanalytics.SourceUI, events[0].Source)
				assert.NotEmpty(t, events[0].OrganizationID)
				assert.Equal(t, productanalytics.StableEventID(productanalytics.EventOrganizationCreated, events[0].OrganizationID), events[0].EventID)
				for _, key := range []string{"email", "name"} {
					if expected, present := tc.identity[key]; present {
						assert.Equal(t, expected, events[0].Properties[key])
						continue
					}
					assert.NotContains(t, events[0].Properties, key)
				}
				groups := recorder.groupIdentities()
				require.Len(t, groups, 1)
				assert.Equal(t, events[0].OrganizationID, groups[0].ID)
				assert.Equal(t, "Metrics Organization", groups[0].Name)
				assert.Equal(t, account.ID.String(), groups[0].AccountID)
				assert.NotEmpty(t, groups[0].Slug)
				require.NotNil(t, groups[0].CreatedAt)
				if len(tc.identity) == 0 {
					assert.NotContains(t, events[0].Properties, "$set")
					return
				}
				assert.Equal(t, tc.identity, events[0].Properties["$set"])
			})
		})
	}
}

func TestInitialWorkspaceAnalytics(t *testing.T) {
	t.Setenv(config.EnvGitHubAppID, "")
	t.Setenv(config.EnvGitHubAppSlug, "")
	t.Setenv(config.EnvGitHubAppPrivateKey, "")
	t.Setenv(config.EnvGitHubAppWebhookSecret, "")
	recorder := useAnalytics(t)
	resource := support.Setup(t)
	account, err := models.CreateAccount("Onboarding User", "onboarding@example.com")
	require.NoError(t, err)
	require.NoError(t, models.SaveAccountLinkedAccount(
		database.DB(t.Context()),
		models.NewAccountLinkedAccount(account.ID, models.ProviderGitHub, "github-owner-id", "github-owner", "GitHub Owner", ""),
	))

	server, err := NewServer(
		resource.Encryptor,
		resource.Registry,
		jwt.NewSigner("test"),
		support.NewOIDCProvider(),
		"",
		"localhost",
		"",
		"test",
		"/app/templates",
		resource.AuthService,
		false,
	)
	require.NoError(t, err)

	attemptID := uuid.New()
	post := func() *httptest.ResponseRecorder {
		body, err := json.Marshal(initialWorkspaceRequest{Owner: "GitHub Owner", AttemptID: attemptID.String()})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/account/onboarding", bytes.NewReader(body))
		request = request.WithContext(accountContext(account))
		response := httptest.NewRecorder()
		server.createInitialWorkspace(response, request)
		return response
	}

	require.Equal(t, http.StatusOK, post().Code)
	require.Equal(t, http.StatusOK, post().Code)

	events := recorder.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, productanalytics.EventOrganizationCreated, events[0].Name)
	assert.Equal(t, "initial_onboarding", events[0].Properties["creation_path"])
	assert.Equal(t, account.ID.String(), events[0].DistinctID)
	assert.Equal(t, productanalytics.ActorUser, events[0].ActorType)
	assert.Equal(t, "onboarding@example.com", events[0].Properties["email"])
	assert.Equal(t, "Onboarding User", events[0].Properties["name"])
	assert.Equal(t, productanalytics.Properties{
		"email": "onboarding@example.com",
		"name":  "Onboarding User",
	}, events[0].Properties["$set"])
	assert.Equal(t, productanalytics.SourceUI, events[0].Source)
	assert.Equal(t, productanalytics.StableEventID(productanalytics.EventOrganizationCreated, events[0].OrganizationID), events[0].EventID)

	groups := recorder.groupIdentities()
	require.Len(t, groups, 1)
	assert.Equal(t, events[0].OrganizationID, groups[0].ID)
	assert.Equal(t, "GitHub Owner", groups[0].Name)
	assert.Equal(t, account.ID.String(), groups[0].AccountID)
	assert.NotEmpty(t, groups[0].Slug)
}
