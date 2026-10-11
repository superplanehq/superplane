package public

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/bitbucket"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func TestHandleBitbucketForgeEventRoutesOnlyTheInstalledRepository(t *testing.T) {
	for _, commentEvent := range []bool{false, true} {
		t.Run(fmt.Sprintf("comment=%t", commentEvent), func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			require.NoError(t, err)
			useForgeTestApp(t, key)
			r.Registry.Triggers["bitbucket.onPullRequest"] = &bitbucket.OnPullRequest{}
			r.Registry.Triggers["bitbucket.onPullRequestComment"] = &bitbucket.OnPullRequestComment{}
			triggerName, webhookEvent, forgeEvent := "bitbucket.onPullRequest", "pullrequest:fulfilled", "avi:bitbucket:fulfilled:pullrequest"
			if commentEvent {
				triggerName, webhookEvent, forgeEvent = "bitbucket.onPullRequestComment", "pullrequest:comment_created", "avi:bitbucket:created:pullrequest-comment"
			}
			server := linearAppWebhookServer(t, r)
			installationID := uuid.NewString()
			workspaceID := uuid.NewString()
			repositoryID := uuid.NewString()

			createSubscription := func(installation, repository string) *models.Canvas {
				integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "bitbucket", uuid.NewString(), nil)
				require.NoError(t, err)
				require.NoError(t, database.Conn().Model(integration).Updates(map[string]any{
					"state":    models.IntegrationStateReady,
					"metadata": datatypes.NewJSONType(map[string]any{"authType": "forgeApp", "forgeInstallationId": installation}),
				}).Error)
				webhookID := uuid.New()
				secret, err := r.Encryptor.Encrypt(t.Context(), []byte("secret"), []byte(webhookID.String()))
				require.NoError(t, err)
				require.NoError(t, database.Conn().Create(&models.Webhook{
					ID: webhookID, State: models.WebhookStateReady, Secret: secret, AppInstallationID: &integration.ID,
					Configuration: datatypes.NewJSONType[any](bitbucket.WebhookConfiguration{RepositorySlug: "widgets", EventTypes: []string{webhookEvent}}),
					Metadata:      datatypes.NewJSONType[any](bitbucket.BitbucketWebhook{RepositoryUUID: repository, RepositoryFullName: "acme/widgets"}),
				}).Error)
				canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{{
					NodeID: "on-closed", Name: "on-closed", Type: models.NodeTypeTrigger,
					Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
					Configuration: datatypes.NewJSONType(map[string]any{"repository": "acme/widgets", "actions": []string{"merged"}}),
				}}, nil)
				require.NoError(t, database.Conn().Model(&models.CanvasNode{}).Where("workflow_id = ?", canvas.ID).
					Updates(map[string]any{"webhook_id": webhookID, "app_installation_id": integration.ID}).Error)
				return canvas
			}
			matched := createSubscription(installationID, repositoryID)
			otherInstallation := createSubscription(uuid.NewString(), repositoryID)
			otherRepository := createSubscription(installationID, uuid.NewString())
			timestamp := time.Now().Format(time.RFC3339)
			selfGenerated := false
			post := func(token, workspace string) *httptest.ResponseRecorder {
				body, err := json.Marshal(map[string]any{
					"eventType":     forgeEvent,
					"selfGenerated": selfGenerated,
					"comment":       map[string]any{"id": 7, "content": map[string]any{"raw": "@superplaneagent fix conflicts"}, "user": map[string]any{"nickname": "ada"}},
					"timestamp":     timestamp,
					"workspace":     map[string]any{"uuid": "{" + workspace + "}"},
					"repository":    map[string]any{"uuid": "{" + repositoryID + "}"},
					"pullrequest": map[string]any{"id": 2, "state": "MERGED",
						"title":       map[string]any{"value": "Fix import"},
						"source":      map[string]any{"branch": "fix-import", "commit": map[string]any{"hash": "abc123"}},
						"destination": map[string]any{"branch": "main", "commit": map[string]any{"hash": "def456"}},
					},
				})
				require.NoError(t, err)
				request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/events", bytes.NewReader(body))
				request.Header.Set("Authorization", "Bearer "+token)
				request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
				response := httptest.NewRecorder()
				server.HandleBitbucketForgeEvent(response, request)
				return response
			}
			token := signForgeInvocation(t, key, installationID, workspaceID)
			require.Equal(t, http.StatusUnauthorized, post("invalid", workspaceID).Code)
			require.Equal(t, http.StatusBadRequest, post(token, uuid.NewString()).Code)
			require.Equal(t, http.StatusNoContent, post(token, workspaceID).Code)
			support.VerifyCanvasEventsCount(t, matched.ID, 1)
			support.VerifyCanvasEventsCount(t, otherInstallation.ID, 0)
			support.VerifyCanvasEventsCount(t, otherRepository.ID, 0)
			var event models.CanvasEvent
			require.NoError(t, database.Conn().Where("workflow_id = ?", matched.ID).First(&event).Error)
			data := event.Data.Data().(map[string]any)["data"].(map[string]any)
			if commentEvent {
				comment := data["comment"].(map[string]any)
				assert.Equal(t, "@superplaneagent fix conflicts", comment["body"])
				assert.Equal(t, "https://bitbucket.org/acme/widgets/pull-requests/2#comment-7", comment["html_url"])
				selfGenerated = true
				require.Equal(t, http.StatusNoContent, post(token, workspaceID).Code)
				support.VerifyCanvasEventsCount(t, matched.ID, 1)
				selfGenerated = false
			} else {
				assert.Equal(t, "merged", data["action"])
				view := data["pull_request"].(map[string]any)
				assert.Equal(t, true, view["merged"])
				assert.Equal(t, "https://bitbucket.org/acme/widgets/pull-requests/2", view["html_url"])
				assert.Equal(t, timestamp, view["merged_at"])
				assert.Equal(t, "Fix import", view["title"])
				assert.Equal(t, map[string]any{"ref": "fix-import", "sha": "abc123"}, view["head"])
			}
			_, err = models.SaveBitbucketForgeDelivery(database.Conn(), models.BitbucketForgeDelivery{
				InstallationID: installationID, DeliveredAt: time.Now(), Uninstall: true,
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, post(token, workspaceID).Code)
			support.VerifyCanvasEventsCount(t, matched.ID, 1)
		})
	}
}

func TestHandleBitbucketForgeBuildEventValidation(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	useForgeTestApp(t, key)
	r.Registry.Actions["bitbucket.waitForBuilds"] = &bitbucket.WaitForBuilds{}
	server := linearAppWebhookServer(t, r)
	installationID := uuid.NewString()
	workspaceID := uuid.NewString()
	repositoryID := uuid.NewString()
	sha := "b88c4b490b648bf960eba6f59123456797960e55"

	post := func(token, workspace string, body map[string]any) *httptest.ResponseRecorder {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/events", bytes.NewReader(encoded))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
		response := httptest.NewRecorder()
		server.HandleBitbucketForgeEvent(response, request)
		return response
	}
	validBody := func() map[string]any {
		return map[string]any{
			"eventType":  "avi:bitbucket:created:build-status",
			"timestamp":  time.Now().Format(time.RFC3339),
			"workspace":  map[string]any{"uuid": "{" + workspaceID + "}"},
			"repository": map[string]any{"uuid": "{" + repositoryID + "}"},
			"buildStatus": map[string]any{
				"key": "my-build1", "state": "FAILED",
				"commit":    map[string]any{"hash": sha},
				"url":       "https://ci.example/1",
				"createdOn": time.Now().Format(time.RFC3339),
				"updatedOn": time.Now().Format(time.RFC3339),
			},
		}
	}
	token := signForgeInvocation(t, key, installationID, workspaceID)

	// Invalid FIT
	require.Equal(t, http.StatusUnauthorized, post("invalid", workspaceID, validBody()).Code)
	// Workspace mismatch
	mismatch := validBody()
	mismatch["workspace"] = map[string]any{"uuid": "{" + uuid.NewString() + "}"}
	require.Equal(t, http.StatusBadRequest, post(token, workspaceID, mismatch).Code)
	// Missing build key
	missingKey := validBody()
	missingKey["buildStatus"] = map[string]any{"commit": map[string]any{"hash": sha}}
	require.Equal(t, http.StatusBadRequest, post(token, workspaceID, missingKey).Code)
	// Invalid SHA
	badSHA := validBody()
	badSHA["buildStatus"] = map[string]any{"key": "my-build1", "commit": map[string]any{"hash": "abc"}}
	require.Equal(t, http.StatusBadRequest, post(token, workspaceID, badSHA).Code)
	// Valid build event without a PR ID dispatches (no subscriptions, still accepted)
	require.Equal(t, http.StatusNoContent, post(token, workspaceID, validBody()).Code)
	// Updated build event is also accepted
	updated := validBody()
	updated["eventType"] = "avi:bitbucket:updated:build-status"
	require.Equal(t, http.StatusNoContent, post(token, workspaceID, updated).Code)
	// Unknown event type is rejected
	unknown := validBody()
	unknown["eventType"] = "avi:bitbucket:created:repository"
	require.Equal(t, http.StatusBadRequest, post(token, workspaceID, unknown).Code)
}

func TestHandleBitbucketForgeBuildEventRoutesOnlyRequestedTypes(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	useForgeTestApp(t, key)
	r.Registry.Triggers["bitbucket.onPullRequest"] = &bitbucket.OnPullRequest{}
	r.Registry.Actions["bitbucket.waitForBuilds"] = &bitbucket.WaitForBuilds{}
	server := linearAppWebhookServer(t, r)
	installationID := uuid.NewString()
	workspaceID := uuid.NewString()
	repositoryID := uuid.NewString()
	sha := "b88c4b490b648bf960eba6f59123456797960e55"

	createWebhook := func(eventTypes []string) {
		integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "bitbucket", uuid.NewString(), nil)
		require.NoError(t, err)
		require.NoError(t, database.Conn().Model(integration).Updates(map[string]any{
			"state":    models.IntegrationStateReady,
			"metadata": datatypes.NewJSONType(map[string]any{"authType": "forgeApp", "forgeInstallationId": installationID}),
		}).Error)
		webhookID := uuid.New()
		secret, err := r.Encryptor.Encrypt(t.Context(), []byte("secret"), []byte(webhookID.String()))
		require.NoError(t, err)
		require.NoError(t, database.Conn().Create(&models.Webhook{
			ID: webhookID, State: models.WebhookStateReady, Secret: secret, AppInstallationID: &integration.ID,
			Configuration: datatypes.NewJSONType[any](bitbucket.WebhookConfiguration{RepositorySlug: "widgets", EventTypes: eventTypes}),
			Metadata:      datatypes.NewJSONType[any](bitbucket.BitbucketWebhook{RepositoryUUID: repositoryID, RepositoryFullName: "acme/widgets"}),
		}).Error)
	}
	// One PR-only subscription and one build-only subscription on the same repository
	createWebhook([]string{"pullrequest:fulfilled"})
	createWebhook([]string{"repo:commit_status_created", "repo:commit_status_updated"})

	post := func(eventType string, body map[string]any) *httptest.ResponseRecorder {
		body["eventType"] = eventType
		body["timestamp"] = time.Now().Format(time.RFC3339)
		body["workspace"] = map[string]any{"uuid": "{" + workspaceID + "}"}
		body["repository"] = map[string]any{"uuid": "{" + repositoryID + "}"}
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		token := signForgeInvocation(t, key, installationID, workspaceID)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/events", bytes.NewReader(encoded))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
		response := httptest.NewRecorder()
		server.HandleBitbucketForgeEvent(response, request)
		return response
	}

	// Build event reaches build subscription without requiring a PR ID
	buildBody := map[string]any{
		"buildStatus": map[string]any{
			"key": "my-build1", "state": "FAILED",
			"commit": map[string]any{"hash": sha},
		},
	}
	require.Equal(t, http.StatusNoContent, post("avi:bitbucket:created:build-status", buildBody).Code)
	require.Equal(t, http.StatusNoContent, post("avi:bitbucket:updated:build-status", buildBody).Code)

	// Unrelated repository cannot dispatch (no matching webhook, still accepted)
	otherRepo := map[string]any{
		"buildStatus": map[string]any{
			"key": "my-build1", "state": "FAILED",
			"commit": map[string]any{"hash": sha},
		},
		"repository": map[string]any{"uuid": "{" + uuid.NewString() + "}"},
	}
	encoded, _ := json.Marshal(otherRepo)
	_ = encoded
	token := signForgeInvocation(t, key, installationID, workspaceID)
	otherBody, _ := json.Marshal(map[string]any{
		"eventType":   "avi:bitbucket:created:build-status",
		"timestamp":   time.Now().Format(time.RFC3339),
		"workspace":   map[string]any{"uuid": "{" + workspaceID + "}"},
		"repository":  map[string]any{"uuid": "{" + uuid.NewString() + "}"},
		"buildStatus": map[string]any{"key": "my-build1", "commit": map[string]any{"hash": sha}},
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/events", bytes.NewReader(otherBody))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
	response := httptest.NewRecorder()
	server.HandleBitbucketForgeEvent(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)
}

const forgeTestAppID = "ari:cloud:ecosystem::app/11111111-1111-1111-1111-111111111111"

func TestHandleBitbucketForgeUninstallClearsTheCachedToken(t *testing.T) {
	r := support.Setup(t)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	useForgeTestApp(t, privateKey)

	installationID := uuid.NewString()
	_, err = models.SaveBitbucketForgeDelivery(database.Conn(), models.BitbucketForgeDelivery{
		InstallationID: installationID,
		SystemToken:    []byte("cached-token"),
		TokenExpiresAt: time.Now().Add(2 * time.Hour),
		DeliveredAt:    time.Now(),
	})
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodPost, "/api/v1/bitbucket/forge/uninstall", nil)
	request.Header.Set("Authorization", "Bearer "+signForgeInvocation(t, privateKey, installationID))
	request.Header.Set("x-forge-oauth-system", signForgeSystemToken(t, time.Now().Add(time.Hour)))
	response := httptest.NewRecorder()

	(&Server{encryptor: r.Encryptor}).HandleBitbucketForgeUninstall(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	installation, err := models.FindBitbucketForgeInstallation(database.Conn(), installationID)
	require.NoError(t, err)
	assert.NotNil(t, installation.UninstalledAt)
	assert.Empty(t, installation.SystemToken)
}

func useForgeTestApp(t *testing.T, privateKey *rsa.PrivateKey) {
	t.Helper()
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": "forge-test-key",
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
			}},
		})
	}))
	t.Cleanup(jwks.Close)
	t.Setenv(config.EnvBitbucketForgeAppID, forgeTestAppID)
	t.Setenv(config.EnvBitbucketForgeInstallURL, "https://bitbucket.example/install")
	t.Setenv(config.EnvBitbucketForgeJWKSURL, jwks.URL)
}

func signForgeInvocation(t *testing.T, privateKey *rsa.PrivateKey, installationID string, workspaceIDs ...string) string {
	t.Helper()
	workspaceID := uuid.NewString()
	if len(workspaceIDs) > 0 {
		workspaceID = workspaceIDs[0]
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodRS256, jwtlib.MapClaims{
		"aud": forgeTestAppID,
		"exp": time.Now().Add(time.Minute).Unix(),
		"app": map[string]any{
			"id":             forgeTestAppID,
			"installationId": installationID,
		},
		"context": map[string]any{
			"installContext": "ari:cloud:bitbucket::workspace/" + workspaceID,
		},
	})
	token.Header["kid"] = "forge-test-key"
	signed, err := token.SignedString(privateKey)
	require.NoError(t, err)
	return signed
}

func signForgeSystemToken(t *testing.T, expires time.Time) string {
	t.Helper()
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.RegisteredClaims{
		ExpiresAt: jwtlib.NewNumericDate(expires),
	})
	signed, err := token.SignedString([]byte("system-token-test-secret"))
	require.NoError(t, err)
	return signed
}
