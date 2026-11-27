package public

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestGitHubAppCreationRejectsWrongState(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, provider := creationTestServer(t)
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "wizard", true: "legacy"}[legacy], func(t *testing.T) {
			integration := seedGitHubCreation(t, r.Organization.ID, legacy)
			if legacy {
				integration.Metadata = datatypes.NewJSONType(map[string]any{"state": "expected-state", "githubApp": map[string]any{"id": 99, "slug": "sentinel", "clientId": "sentinel-client"}})
				require.NoError(t, database.Conn().Create(&models.IntegrationSecret{OrganizationID: integration.OrganizationID, InstallationID: integration.ID, Name: "clientSecret", Value: []byte("sentinel-secret")}).Error)
				require.NoError(t, database.Conn().Save(integration).Error)
			}
			before, err := models.FindUnscopedIntegration(integration.ID)
			require.NoError(t, err)
			secrets, err := models.ListIntegrationSecrets(integration.ID)
			require.NoError(t, err)
			response := execRequest(server, requestParams{method: "GET", path: "/api/v1/integrations/" + integration.ID.String() + "/redirect?code=dummy-code&state=wrong-state"})
			assert.Equal(t, http.StatusBadRequest, response.Code)
			after, err := models.FindUnscopedIntegration(integration.ID)
			require.NoError(t, err)
			assert.Equal(t, before.Metadata, after.Metadata)
			assert.Equal(t, before.Properties, after.Properties)
			assert.Equal(t, before.Capabilities, after.Capabilities)
			assert.Equal(t, before.SetupState, after.SetupState)
			afterSecrets, err := models.ListIntegrationSecrets(integration.ID)
			require.NoError(t, err)
			assert.Equal(t, secrets, afterSecrets)
		})
	}
	assert.Zero(t, provider.calls.Load())
}

type creationGitHubFixture struct {
	*github.GitHub
	http core.HTTPContext
}

func (g *creationGitHubFixture) HandleRequest(ctx core.HTTPRequestContext) {
	ctx.HTTP = g.http
	g.GitHub.HandleRequest(ctx)
}

type creationHTTPFixture struct {
	calls      atomic.Int64
	privateKey string
	status     int
	entered    chan struct{}
	release    chan struct{}
}

func (f *creationHTTPFixture) Do(r *http.Request) (*http.Response, error) {
	if r.URL.Host != "api.github.com" {
		return nil, fmt.Errorf("unexpected provider host: %s", r.URL.Host)
	}
	if r.URL.Path == "/app/installations/99/access_tokens" {
		return fixtureJSONResponse(201, `{"token":"dummy-token","expires_at":"`+time.Now().Add(time.Hour).UTC().Format(time.RFC3339)+`"}`), nil
	}
	if r.URL.Path == "/installation/repositories" {
		return fixtureJSONResponse(200, `{"total_count":0,"repositories":[]}`), nil
	}
	if r.URL.Path != "/app-manifests/dummy-code/conversions" || r.Method != "POST" {
		return nil, fmt.Errorf("unexpected provider request: %s %s", r.Method, r.URL.Path)
	}
	f.calls.Add(1)
	if f.entered != nil {
		f.entered <- struct{}{}
		<-f.release
	}
	status := f.status
	if status == 0 {
		status = 201
	}
	body, err := json.Marshal(map[string]any{"id": 123, "slug": "dummy-app", "client_id": "dummy-client", "client_secret": "dummy-secret", "webhook_secret": "dummy-webhook", "pem": f.privateKey})
	if err != nil {
		return nil, err
	}
	return fixtureJSONResponse(status, string(body)), nil
}

func fixtureJSONResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": []string{"application/json"}}}
}

func TestGitHubAppCreationClaimsOnce(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "wizard", true: "legacy"}[legacy], func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			server, provider := creationTestServer(t)
			i := seedGitHubCreation(t, r.Organization.ID, legacy)
			path := "/api/v1/integrations/" + i.ID.String() + "/redirect?code=dummy-code&state=expected-state"
			provider.entered = make(chan struct{}, 1)
			provider.release = make(chan struct{})
			winner := make(chan *httptest.ResponseRecorder, 1)
			go func() { winner <- execRequest(server, requestParams{method: "GET", path: path}) }()
			select {
			case <-provider.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("conversion did not start")
			}
			loser := execRequest(server, requestParams{method: "GET", path: path})
			require.Equal(t, 400, loser.Code)
			close(provider.release)
			var response *httptest.ResponseRecorder
			select {
			case response = <-winner:
			case <-time.After(5 * time.Second):
				t.Fatal("conversion did not finish")
			}
			require.Equal(t, 303, response.Code, response.Body.String())
			require.Equal(t, int64(1), provider.calls.Load())
			fresh, err := models.FindUnscopedIntegration(i.ID)
			require.NoError(t, err)
			redirect, err := url.Parse(response.Header().Get("Location"))
			require.NoError(t, err)
			state := redirect.Query().Get("state")
			require.NotEmpty(t, state)
			require.NotEqual(t, "expected-state", state)
			if legacy {
				assert.Equal(t, state, fresh.Metadata.Data()["state"])
				assert.Equal(t, map[string]any{"id": float64(123), "slug": "dummy-app", "clientId": "dummy-client"}, fresh.Metadata.Data()["githubApp"])
			} else {
				assert.Equal(t, github.SetupStepInstallApp, fresh.SetupState.Data().CurrentStep.Name)
				for _, p := range fresh.Properties {
					if p.Name == "appState" {
						assert.Equal(t, state, p.Value)
					}
				}
			}
			secrets, err := models.ListIntegrationSecrets(i.ID)
			require.NoError(t, err)
			require.Len(t, secrets, 3)
			for _, secret := range secrets {
				switch secret.Name {
				case "clientSecret", "appClientSecret":
					assert.Equal(t, []byte("dummy-secret"), secret.Value)
				case "webhookSecret", "appWebhookSecret":
					assert.Equal(t, []byte("dummy-webhook"), secret.Value)
				case "pem", "appPEM":
					assert.True(t, string(secret.Value) == provider.privateKey)
				default:
					t.Fatalf("unexpected secret name: %s", secret.Name)
				}
			}
			replay := execRequest(server, requestParams{method: "GET", path: path})
			assert.Equal(t, 400, replay.Code)
			assert.Equal(t, int64(1), provider.calls.Load())
			installation := execRequest(server, requestParams{method: "GET", path: "/api/v1/integrations/" + i.ID.String() + "/setup?installation_id=99&setup_action=install&state=" + url.QueryEscape(state)})
			require.Equal(t, 303, installation.Code, installation.Body.String())
			fresh, err = models.FindUnscopedIntegration(i.ID)
			require.NoError(t, err)
			assert.Equal(t, models.IntegrationStateReady, fresh.State)
		})
	}
}

func TestGitHubAppCreationFailureAndRestart(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			server, provider := creationTestServer(t)
			i := seedGitHubCreation(t, r.Organization.ID, legacy)
			path := "/api/v1/integrations/" + i.ID.String() + "/redirect?code=dummy-code&state=expected-state"
			provider.status = 500
			assert.Equal(t, 500, execRequest(server, requestParams{method: "GET", path: path}).Code)
			assert.Equal(t, 400, execRequest(server, requestParams{method: "GET", path: path}).Code)
			assert.Equal(t, int64(1), provider.calls.Load())
			provider.status = 201
			state := restartGitHubCreation(t, server, i)
			assert.NotEqual(t, "expected-state", state)
			assert.Equal(t, 303, execRequest(server, requestParams{method: "GET", path: strings.ReplaceAll(path, "expected-state", url.QueryEscape(state))}).Code)
		})
	}
}

func TestGitHubAppCreationRejectsSupersededAttempt(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			server, provider := creationTestServer(t)
			i := seedGitHubCreation(t, r.Organization.ID, legacy)
			path := "/api/v1/integrations/" + i.ID.String() + "/redirect?code=dummy-code&state=expected-state"
			provider.entered = make(chan struct{}, 1)
			provider.release = make(chan struct{})
			response := make(chan *httptest.ResponseRecorder, 1)
			go func() { response <- execRequest(server, requestParams{method: "GET", path: path}) }()
			select {
			case <-provider.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("conversion did not start")
			}
			state := restartGitHubCreation(t, server, i)
			close(provider.release)
			select {
			case result := <-response:
				require.Equal(t, 400, result.Code)
			case <-time.After(5 * time.Second):
				t.Fatal("callback did not finish")
			}
			fresh, err := models.FindUnscopedIntegration(i.ID)
			require.NoError(t, err)
			for _, p := range fresh.Properties {
				require.NotEqual(t, "appID", p.Name)
			}
			secrets, err := models.ListIntegrationSecrets(i.ID)
			require.NoError(t, err)
			require.Empty(t, secrets)
			provider.entered = nil
			assert.Equal(t, 303, execRequest(server, requestParams{method: "GET", path: strings.ReplaceAll(path, "expected-state", url.QueryEscape(state))}).Code)
		})
	}
}

func restartGitHubCreation(t *testing.T, server *Server, integration *models.Integration) string {
	t.Helper()
	i, err := models.FindUnscopedIntegration(integration.ID)
	require.NoError(t, err)
	ctx := contexts.NewIntegrationContext(database.Conn(), nil, i, server.encryptor, server.registry, nil)
	if i.IsLegacy() {
		require.NoError(t, (&github.GitHub{}).Sync(core.SyncContext{OrganizationID: i.OrganizationID.String(), Integration: ctx, Configuration: i.Configuration.Data()}))
		require.NoError(t, database.Conn().Save(i).Error)
		return i.Metadata.Data()["state"].(string)
	}
	setup := core.SetupStepContext{
		Step:           core.StepInfo{Name: github.SetupStepSelectAuthMethod, Inputs: map[string]any{"authMethod": "app"}},
		OrganizationID: i.OrganizationID.String(), IntegrationID: i.ID, Logger: logging.ForIntegration(*i),
		Properties: ctx.Properties(), Secrets: ctx.Secrets(), HTTP: server.registry.HTTPContext(),
		Capabilities: contexts.NewCapabilityContext(server.registry.AllCapabilities("github"), i.Capabilities),
	}
	provider := &github.SetupProvider{}
	require.NoError(t, provider.OnStepRevert(setup))
	next, err := provider.OnStepSubmit(setup)
	require.NoError(t, err)
	state := datatypes.NewJSONType(models.SetupState{CurrentStep: next})
	i.SetupState = &state
	require.NoError(t, database.Conn().Save(i).Error)
	return next.RedirectPrompt.FormData["state"]
}

func creationTestServer(t *testing.T) (*Server, *creationHTTPFixture) {
	t.Helper()
	encryptor := &crypto.NoOpEncryptor{}
	reg, err := registry.NewRegistry(encryptor, registry.HTTPOptions{})
	require.NoError(t, err)
	auth, err := authorization.NewAuthService()
	require.NoError(t, err)
	server, err := NewServer(encryptor, reg, jwt.NewSigner("test"), support.NewOIDCProvider(), "/api/v1", "", "", "test", "/app/templates", auth, false)
	require.NoError(t, err)
	server.RegisterWebRoutes("")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	provider := &creationHTTPFixture{privateKey: string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))}
	reg.Integrations["github"] = &creationGitHubFixture{GitHub: &github.GitHub{}, http: provider}
	return server, provider
}

func TestGitHubAppCreationRejectsIneligibleCallbacks(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, provider := creationTestServer(t)
	for _, legacy := range []bool{false, true} {
		for _, name := range []string{"missing code", "missing state", "another state", "empty stored state", "malformed stored state", "missing stored state", "no pending step", "existing App", "existing secret", "ready integration"} {
			t.Run(fmt.Sprintf("%t/%s", legacy, name), func(t *testing.T) {
				i := seedGitHubCreation(t, r.Organization.ID, legacy)
				query := "code=dummy-code&state=expected-state"
				switch name {
				case "ready integration":
					i.State = models.IntegrationStateReady
				case "missing stored state":
					if legacy {
						i.Metadata = datatypes.NewJSONType(map[string]any{})
					} else {
						i.Properties = i.Properties[1:]
					}
				case "missing code":
					query = "state=expected-state"
				case "missing state":
					query = "code=dummy-code"
				case "another state":
					query = "code=dummy-code&state=another-integration-state"
				case "empty stored state", "malformed stored state":
					var value any = ""
					if name == "malformed stored state" {
						value = 42
					}
					if legacy {
						i.Metadata = datatypes.NewJSONType(map[string]any{"state": value})
					} else {
						i.Properties[0].Value = value
					}
				case "no pending step":
					if legacy {
						i.BrowserAction = nil
					} else {
						setup := datatypes.NewJSONType(models.SetupState{CurrentStep: &core.SetupStep{Name: github.SetupStepInstallApp}})
						i.SetupState = &setup
					}
				case "existing App":
					if legacy {
						i.Metadata = datatypes.NewJSONType(map[string]any{"state": "expected-state", "githubApp": map[string]any{"id": 99, "slug": "sentinel", "clientId": "sentinel"}})
					} else {
						i.Properties = append(i.Properties, core.IntegrationPropertyDefinition{Name: "appID", Value: "99"})
					}
				case "existing secret":
					require.NoError(t, database.Conn().Create(&models.IntegrationSecret{OrganizationID: i.OrganizationID, InstallationID: i.ID, Name: "pem", Value: []byte("sentinel")}).Error)
				}
				require.NoError(t, database.Conn().Save(i).Error)
				before, err := models.FindUnscopedIntegration(i.ID)
				require.NoError(t, err)
				beforeSecrets, err := models.ListIntegrationSecrets(i.ID)
				require.NoError(t, err)
				response := execRequest(server, requestParams{method: "GET", path: "/api/v1/integrations/" + i.ID.String() + "/redirect?" + query})
				require.Equal(t, 400, response.Code)
				after, err := models.FindUnscopedIntegration(i.ID)
				require.NoError(t, err)
				require.Equal(t, before, after)
				afterSecrets, err := models.ListIntegrationSecrets(i.ID)
				require.NoError(t, err)
				require.Equal(t, beforeSecrets, afterSecrets)
			})
		}
	}
	assert.Zero(t, provider.calls.Load())
}

func TestGitHubAppCreationStopsOnClaimStorageFailure(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	server, provider := creationTestServer(t)
	i := seedGitHubCreation(t, r.Organization.ID, false)
	before, err := models.FindUnscopedIntegration(i.ID)
	require.NoError(t, err)
	callback := "test:fail-app-creation-claim"
	require.NoError(t, database.Conn().Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "app_installations" {
			tx.AddError(fmt.Errorf("fixture claim storage failure"))
		}
	}))
	t.Cleanup(func() { database.Conn().Callback().Update().Remove(callback) })
	result := execRequest(server, requestParams{method: "GET", path: "/api/v1/integrations/" + i.ID.String() + "/redirect?code=dummy-code&state=expected-state"})
	require.Equal(t, 500, result.Code)
	assert.Zero(t, provider.calls.Load())
	after, err := models.FindUnscopedIntegration(i.ID)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestGitHubAppCreationRollsBackFailedFinalization(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("legacy=%t", legacy), func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			server, provider := creationTestServer(t)
			i := seedGitHubCreation(t, r.Organization.ID, legacy)
			callback := "test:fail-second-app-secret"
			var writes int
			require.NoError(t, database.Conn().Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == "app_installation_secrets" {
					writes++
					if writes == 2 {
						tx.AddError(fmt.Errorf("fixture secret write failure"))
					}
				}
			}))
			t.Cleanup(func() { database.Conn().Callback().Create().Remove(callback) })
			path := "/api/v1/integrations/" + i.ID.String() + "/redirect?code=dummy-code&state=expected-state"
			assert.Equal(t, 500, execRequest(server, requestParams{method: "GET", path: path}).Code)
			assert.Equal(t, 400, execRequest(server, requestParams{method: "GET", path: path}).Code)
			assert.Equal(t, int64(1), provider.calls.Load())
			secrets, err := models.ListIntegrationSecrets(i.ID)
			require.NoError(t, err)
			assert.Empty(t, secrets)
			fresh, err := models.FindUnscopedIntegration(i.ID)
			require.NoError(t, err)
			assert.Empty(t, fresh.Metadata.Data()["githubApp"])
			for _, p := range fresh.Properties {
				assert.NotEqual(t, "appID", p.Name)
			}
		})
	}
}

func seedGitHubCreation(t *testing.T, orgID uuid.UUID, legacy bool) *models.Integration {
	t.Helper()
	i, err := models.CreateIntegration(uuid.New(), orgID, "github", uuid.NewString(), map[string]any{"organization": "dummy-owner", "privateApp": true})
	require.NoError(t, err)
	if legacy {
		i.Metadata = datatypes.NewJSONType(map[string]any{"state": "expected-state", "owner": "dummy-owner"})
		action := datatypes.NewJSONType(models.BrowserAction{Method: "POST", FormFields: map[string]string{"manifest": "{}", "state": "expected-state"}})
		i.BrowserAction = &action
	} else {
		setup := datatypes.NewJSONType(models.SetupState{CurrentStep: &core.SetupStep{Name: github.SetupStepCreateApp, Type: core.SetupStepTypeRedirectPrompt}})
		i.SetupState = &setup
		i.Properties = datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{{Name: "appState", Value: "expected-state", Type: core.IntegrationPropertyTypeString}, {Name: "owner", Value: "dummy-owner", Type: core.IntegrationPropertyTypeString}})
		i.Properties = append(i.Properties, core.IntegrationPropertyDefinition{Name: "ownerType", Value: "Organization", Type: core.IntegrationPropertyTypeString})
	}
	require.NoError(t, database.Conn().Save(i).Error)
	return i
}
