package e2e

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	gogithub "github.com/google/go-github/v74/github"
	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	githubintegration "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestGithubIntegrationSetup(t *testing.T) {
	t.Run("connecting github integration", func(t *testing.T) {
		patches := mockGitHubHTTP(t)
		defer patches.Reset()

		steps := &githubIntegrationSetupSteps{t: t}
		integrationName := fmt.Sprintf("github-e2e-%d", time.Now().UnixNano())

		steps.givenILogIntoANewOrganization()
		steps.whenIVisitIntegrationsSettingsPage()
		steps.whenIOpenGithubConnectModal()
		steps.whenIConnectGithubIntegration(integrationName, "")
		steps.thenIAmOnIntegrationDetailsPage()
		steps.thenISeePendingSetupWithContinueAction()

		integration := steps.thenGithubIntegrationIsCreated(integrationName, "", "https://github.com/settings/apps/new")

		steps.whenITriggerGithubRedirectCallback(integration, "fake-manifest-code")
		steps.whenITriggerGithubSetupCallback(integration, "999999")
		steps.thenGithubIntegrationIsReady(integrationName, "999999")
	})
}

func mockGitHubHTTP(t *testing.T) *gomonkey.Patches {
	t.Helper()

	patches := gomonkey.NewPatches()
	patches.ApplyMethod(
		reflect.TypeOf(&registry.HTTPContext{}),
		"Do",
		func(_ *registry.HTTPContext, req *http.Request) (*http.Response, error) {
			if req.URL.Host != "api.github.com" {
				return nil, fmt.Errorf("unexpected outbound host: %s", req.URL.Host)
			}

			if req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/app-manifests/") &&
				strings.HasSuffix(req.URL.Path, "/conversions") {
				return mockJSONResponse(http.StatusCreated, `{"id":123456,"slug":"superplane-e2e-app","client_id":"Iv1.e2eclientid","client_secret":"e2e-client-secret","webhook_secret":"e2e-webhook-secret","pem":"-----BEGIN PRIVATE KEY-----\nignored-for-e2e-mock\n-----END PRIVATE KEY-----\n"}`), nil
			}

			return nil, fmt.Errorf("unexpected github request: %s %s", req.Method, req.URL.String())
		},
	)

	patches.ApplyFunc(
		githubintegration.NewClient,
		func(_ core.IntegrationContext, _ int64, _ string) (*gogithub.Client, error) {
			httpClient := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					switch {
					case req.Method == http.MethodGet && (req.URL.Path == "/app" || strings.HasPrefix(req.URL.Path, "/apps/")):
						return mockJSONResponse(http.StatusOK, `{"id":123456,"slug":"superplane-e2e-app","owner":{"login":"superplane-e2e-owner"}}`), nil
					case req.Method == http.MethodGet && req.URL.Path == "/installation/repositories":
						return mockJSONResponse(http.StatusOK, `{"total_count":1,"repositories":[{"id":987654321,"name":"demo-repo","html_url":"https://github.com/superplane-e2e-owner/demo-repo"}]}`), nil
					default:
						return nil, fmt.Errorf("unexpected github client request: %s %s", req.Method, req.URL.Path)
					}
				}),
			}
			return gogithub.NewClient(httpClient), nil
		},
	)

	return patches
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func mockJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type githubIntegrationSetupSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *githubIntegrationSetupSteps) givenILogIntoANewOrganization() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *githubIntegrationSetupSteps) whenIVisitIntegrationsSettingsPage() {
	s.session.Visit("/" + s.session.OrgID.String() + "/settings/integrations")
	s.session.Sleep(500)
}

func (s *githubIntegrationSetupSteps) whenIOpenGithubConnectModal() {
	filterInput := q.TestID("integrations-filter-input").Run(s.session)
	err := filterInput.Fill("GitHub")
	require.NoError(s.t, err)

	githubCard := q.TestID("integrations-provider-card-github").Run(s.session)
	err = githubCard.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)

	connectButton := q.TestID("integrations-provider-connect-button-github").Run(s.session)
	err = connectButton.Click()
	require.NoError(s.t, err)

	connectTitle := q.TestID("integrations-connect-modal-title").Run(s.session)
	err = connectTitle.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *githubIntegrationSetupSteps) whenIConnectGithubIntegration(name, organization string) {
	integrationNameInput := q.TestID("integrations-connect-name-input").Run(s.session)
	err := integrationNameInput.Fill(name)
	require.NoError(s.t, err)

	organizationInput := q.TestID("string-field-organization").Run(s.session)
	if organization != "" {
		err = organizationInput.Fill(organization)
		require.NoError(s.t, err)
	} else {
		err = organizationInput.Fill("")
		require.NoError(s.t, err)
	}

	connectButton := q.TestID("integrations-connect-submit-button").Run(s.session)
	err = connectButton.Click()
	require.NoError(s.t, err)

	s.session.Sleep(1000)
}

func (s *githubIntegrationSetupSteps) thenIAmOnIntegrationDetailsPage() {
	s.session.AssertURLContains("/settings/integrations/")
}

func (s *githubIntegrationSetupSteps) thenISeePendingSetupWithContinueAction() {
	page := s.session.Page()

	pendingLabel := page.GetByText("Pending", pw.PageGetByTextOptions{Exact: pw.Bool(true)}).First()
	err := pendingLabel.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)

	continueButton := page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Continue"}).First()
	err = continueButton.WaitFor(pw.LocatorWaitForOptions{State: pw.WaitForSelectorStateVisible})
	require.NoError(s.t, err)
}

func (s *githubIntegrationSetupSteps) thenGithubIntegrationIsCreated(name, organization, expectedBrowserActionURL string) *models.Integration {
	integration, err := models.FindIntegrationByName(s.session.OrgID, name)
	require.NoError(s.t, err)

	assert.Equal(s.t, "github", integration.AppName)
	assert.Equal(s.t, models.IntegrationStatePending, integration.State)
	assert.NotNil(s.t, integration.BrowserAction)

	configuration := integration.Configuration.Data()
	gotOrganization, _ := configuration["organization"].(string)
	assert.Equal(s.t, organization, gotOrganization)

	browserAction := integration.BrowserAction.Data()
	assert.Equal(s.t, "POST", browserAction.Method)
	assert.Equal(s.t, expectedBrowserActionURL, browserAction.URL)
	assert.NotEmpty(s.t, browserAction.FormFields["manifest"])
	assert.NotEmpty(s.t, browserAction.FormFields["state"])

	return integration
}

func (s *githubIntegrationSetupSteps) whenITriggerGithubRedirectCallback(integration *models.Integration, code string) {
	metadata := integration.Metadata.Data()
	state, _ := metadata["state"].(string)
	require.NotEmpty(s.t, state)

	url := fmt.Sprintf("%s/api/v1/integrations/%s/redirect?code=%s&state=%s", s.session.BaseURL, integration.ID.String(), code, state)
	resp := s.callWithoutRedirect(url)
	defer resp.Body.Close()
	require.Equal(s.t, http.StatusSeeOther, resp.StatusCode)
}

func (s *githubIntegrationSetupSteps) whenITriggerGithubSetupCallback(integration *models.Integration, installationID string) {
	refreshed, err := models.FindIntegration(s.session.OrgID, integration.ID)
	require.NoError(s.t, err)
	metadata := refreshed.Metadata.Data()
	state, _ := metadata["state"].(string)
	require.NotEmpty(s.t, state)

	url := fmt.Sprintf(
		"%s/api/v1/integrations/%s/setup?installation_id=%s&setup_action=install&state=%s",
		s.session.BaseURL,
		refreshed.ID.String(),
		installationID,
		state,
	)

	resp := s.callWithoutRedirect(url)
	defer resp.Body.Close()
	require.Equal(s.t, http.StatusSeeOther, resp.StatusCode)
}

func (s *githubIntegrationSetupSteps) thenGithubIntegrationIsReady(name, installationID string) {
	integration, err := models.FindIntegrationByName(s.session.OrgID, name)
	require.NoError(s.t, err)
	assert.Equal(s.t, models.IntegrationStateReady, integration.State)
	assert.Nil(s.t, integration.BrowserAction)

	metadata := integration.Metadata.Data()
	gotInstallationID, _ := metadata["installationId"].(string)
	assert.Equal(s.t, installationID, gotInstallationID)
}

func (s *githubIntegrationSetupSteps) callWithoutRedirect(url string) *http.Response {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	require.NoError(s.t, err)
	return resp
}
