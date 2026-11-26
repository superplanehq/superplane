package factories

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	goyaml "gopkg.in/yaml.v3"
)

// Setup must read grants made by this apply before the transaction commits.
// Replacing provider HTTP preserves the real token scope and webhook setup.
func TestFactoryConfigurationAppliesHostedRepositoryGrantsAtomically(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	original := f.export(t)
	previous := f.factory.OnboardingConfigValue()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	require.NoError(t, models.SaveAccountLinkedAccount(db, models.NewAccountLinkedAccount(
		f.r.Account.ID, models.ProviderGitHub, "42", "octocat", "The Octocat", "https://github.com/octocat.png",
	)))
	const installationID = int64(101)
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{
		Provider: models.ProviderGitHub, InstallationID: installationID, AccountLogin: "acme",
	}))
	repositories := []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/new-app", DefaultBranch: "main"},
		{RepositoryID: 202, FullName: "acme/new-backlog", DefaultBranch: "main"},
		{RepositoryID: 203, FullName: "acme/new-intake", DefaultBranch: "main"},
		{RepositoryID: 204, FullName: "acme/new-docs", DefaultBranch: "main"},
		{RepositoryID: 205, FullName: "acme/new-tools", DefaultBranch: "main"},
		{RepositoryID: 206, FullName: "acme/new-disabled", DefaultBranch: "main"},
	}
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, repositories))
	for _, repository := range repositories {
		require.NoError(t, models.ReplaceVCSProviderRepositoryCollaborators(db, models.ProviderGitHub, repository.RepositoryID,
			[]models.VCSProviderRepositoryCollaborator{{ProviderUserID: 42, ProviderLogin: "octocat"}}))
	}
	hosted, err := models.FindOrCreateVCSProviderBinding(db, f.factory.OrganizationID, models.ProviderGitHub, installationID, "acme")
	require.NoError(t, err)
	granted, err := models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Empty(t, granted)

	implementation, err := f.r.Registry.GetIntegration(models.ProviderGitHub)
	require.NoError(t, err)
	for {
		adapter, ok := implementation.(configurationCatalogIntegration)
		if !ok {
			break
		}
		implementation = adapter.Integration
	}
	client := &configurationHostedRepositoryHTTP{}
	f.r.Registry.Integrations[models.ProviderGitHub] = configurationHostedRepositoryIntegration{Integration: implementation, http: client}
	f.ctx = withRepositoryBranchResolver(f.ctx, repositoryBranchResolver{gitHubBranches: func(string) ([]string, error) {
		return []string{"main"}, nil
	}})
	bundle := f.export(t)
	var manifest factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &manifest))
	disabled := false
	manifest.Repositories = &factoryConfigurationRepositories{
		Binding:           factoryConfigurationBinding{IntegrationName: hosted.InstallationName},
		Primary:           factoryConfigurationRepository{Repository: "acme/new-app", Ref: "main"},
		BacklogRepository: "acme/new-backlog",
		Additional: []factoryConfigurationContextRepository{
			{Repository: "acme/new-docs", Access: "read"},
			{Repository: "acme/new-tools", Access: "write"},
			{Repository: "acme/new-disabled", Access: "read", Enabled: &disabled},
		},
	}
	for index := range manifest.Stages {
		if manifest.Stages[index].Key == "backlog" {
			manifest.Stages[index].Automations = append(manifest.Stages[index].Automations, factoryConfigurationAutomation{
				ID: "issues", Name: "Hosted issues", Uses: "@superplane/intake/github-issues",
				Binding: &factoryConfigurationBinding{IntegrationID: hosted.ID.String(), ResourceID: "acme/new-intake"},
			})
		}
	}
	content, err := goyaml.Marshal(manifest)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.NotEmpty(t, plan.Changes)
	require.Empty(t, client.tokenGrants, "plan must not request installation tokens")
	granted, err = models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Empty(t, granted, "plan must not grant repository access")
	before := configurationResourceCounts(t)
	revisions := configurationRevisionCount(t, f.canvas)

	client.failRepositoryReads = true
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.ErrorContains(t, err, "repository lookup unavailable")
	require.NotEmpty(t, client.tokenGrants)
	reloaded, err := models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, previous, reloaded.OnboardingConfigValue())
	granted, err = models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Empty(t, granted)
	require.Equal(t, before, configurationResourceCounts(t))
	require.Equal(t, revisions, configurationRevisionCount(t, f.canvas))

	client.failRepositoryReads = false
	client.tokenGrants = nil
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	granted, err = models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	grantedIDs := make([]int64, 0, len(granted))
	for _, grant := range granted {
		grantedIDs = append(grantedIDs, grant.RepositoryID)
	}
	require.ElementsMatch(t, []int64{201, 202, 203, 204, 205, 206}, grantedIDs)
	require.NotEmpty(t, client.tokenGrants, "real trigger setup must obtain a hosted installation token")
	for _, selected := range client.tokenGrants {
		require.ElementsMatch(t, []int64{201, 202, 203, 204, 205, 206}, selected)
	}
	intakes, err := reloaded.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, intakes, 1)
	live := resetTestLiveVersion(t, intakes[0].Canvas)
	trigger := findIntakeNode(live.Nodes, intakeTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, "acme/new-intake", trigger.Configuration["repository"])
	require.Equal(t, hosted.ID.String(), *trigger.IntegrationID)
	for _, node := range live.Nodes {
		require.Nil(t, node.ErrorMessage, node.ID)
	}

	// Organization keys must be able to reuse repositories added by a personal token.
	organizationKey, err := models.CreateAPIKey(db, f.factory.OrganizationID, "YAML round trip", nil, f.r.User, nil, nil)
	require.NoError(t, err)
	require.NoError(t, f.r.AuthService.AssignRole(organizationKey.ID.String(), models.RoleOrgAdmin, f.factory.OrganizationID.String(), models.DomainTypeOrganization))
	organizationCaller := f
	organizationCaller.ctx = authentication.SetUserIdInMetadata(f.ctx, organizationKey.ID.String())
	roundTrip := organizationCaller.plan(t, organizationCaller.export(t))
	require.Empty(t, roundTrip.Diagnostics)
	require.Empty(t, roundTrip.Changes)

	// A pinned intake must retain access when the workspace changes connections.
	bundle = f.export(t)
	var pinned factoryConfigurationManifest
	require.NoError(t, goyaml.Unmarshal([]byte(bundle.Files[0].Content), &pinned))
	var issues *factoryConfigurationAutomation
	for stage := range pinned.Stages {
		for automation := range pinned.Stages[stage].Automations {
			entry := &pinned.Stages[stage].Automations[automation]
			if entry.ID == "issues" {
				issues = entry
			}
		}
	}
	require.NotNil(t, issues)
	issues.Binding = &factoryConfigurationBinding{IntegrationID: hosted.ID.String(), ResourceID: "acme/new-intake"}
	content, err = goyaml.Marshal(pinned)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)

	pinned.Repositories = &factoryConfigurationRepositories{
		Binding:           factoryConfigurationBinding{IntegrationID: previous.VCSIntegrationID},
		Primary:           factoryConfigurationRepository{Repository: previous.AppRepository, Ref: previous.DefaultBranch},
		BacklogRepository: previous.BacklogRepository,
	}
	content, err = goyaml.Marshal(pinned)
	require.NoError(t, err)
	bundle.Files[0].Content = string(content)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	granted, err = models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Len(t, granted, 1)
	require.Equal(t, int64(203), granted[0].RepositoryID)
	live = resetTestLiveVersion(t, intakes[0].Canvas)
	trigger = findIntakeNode(live.Nodes, intakeTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, "acme/new-intake", trigger.Configuration["repository"])
	require.Equal(t, hosted.ID.String(), *trigger.IntegrationID)
	client.tokenGrants = nil
	githubClient, err := githubcommon.NewClient(contexts.NewIntegrationContext(db, nil, hosted, f.r.Encryptor, f.r.Registry, nil), client)
	require.NoError(t, err)
	repository, err := githubClient.FindRepository("acme/new-intake")
	require.NoError(t, err)
	require.Equal(t, "acme/new-intake", repository.GetFullName())
	require.Len(t, client.tokenGrants, 1)
	require.Equal(t, []int64{203}, client.tokenGrants[0])

	// Removing the pinned intake must remove its remaining hosted grant.
	plan = f.plan(t, original)
	require.Empty(t, plan.Diagnostics)
	_, err = f.apply(original, plan.BaseStateToken)
	require.NoError(t, err)
	granted, err = models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Empty(t, granted)
	reloaded, err = models.FindFactory(db, f.factory.OrganizationID, f.factory.ID)
	require.NoError(t, err)
	require.Equal(t, previous.VCSIntegrationID, reloaded.OnboardingConfigValue().VCSIntegrationID)
}

type configurationHostedRepositoryHTTP struct {
	configurationCatalogHTTP
	tokenGrants         [][]int64
	failRepositoryReads bool
}

func (client *configurationHostedRepositoryHTTP) Do(request *http.Request) (*http.Response, error) {
	if request.Method == http.MethodPost && request.URL.Host == "api.github.com" && request.URL.Path == "/app/installations/101/access_tokens" {
		var selection struct {
			RepositoryIDs []int64 `json:"repository_ids"`
		}
		if err := json.NewDecoder(request.Body).Decode(&selection); err != nil {
			return nil, err
		}
		client.tokenGrants = append(client.tokenGrants, selection.RepositoryIDs)
		body := fmt.Sprintf(`{"token":"hosted-configuration-token","expires_at":%q}`, time.Now().Add(time.Hour).Format(time.RFC3339))
		return &http.Response{StatusCode: http.StatusCreated, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
	}
	if client.failRepositoryReads && request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/repos/acme/new-") {
		return nil, fmt.Errorf("repository lookup unavailable")
	}
	return client.configurationCatalogHTTP.Do(request)
}

type configurationHostedRepositoryIntegration struct {
	core.Integration
	http core.HTTPContext
}

func (integration configurationHostedRepositoryIntegration) Triggers() []core.Trigger {
	triggers := integration.Integration.Triggers()
	result := make([]core.Trigger, len(triggers))
	for index, trigger := range triggers {
		result[index] = configurationHostedRepositoryTrigger{Trigger: trigger, http: integration.http}
	}
	return result
}

func (integration configurationHostedRepositoryIntegration) Actions() []core.Action {
	actions := integration.Integration.Actions()
	result := make([]core.Action, len(actions))
	for index, action := range actions {
		result[index] = configurationHostedRepositoryAction{Action: action, http: integration.http}
	}
	return result
}

type configurationHostedRepositoryTrigger struct {
	core.Trigger
	http core.HTTPContext
}

func (trigger configurationHostedRepositoryTrigger) Setup(ctx core.TriggerContext) error {
	ctx.HTTP = trigger.http
	return trigger.Trigger.Setup(ctx)
}

type configurationHostedRepositoryAction struct {
	core.Action
	http core.HTTPContext
}

func (action configurationHostedRepositoryAction) Setup(ctx core.SetupContext) error {
	ctx.HTTP = action.http
	return action.Action.Setup(ctx)
}

func (action configurationHostedRepositoryAction) ValidateNodeConfiguration(configuration map[string]any) error {
	if validator, ok := action.Action.(core.NodeConfigurationValidator); ok {
		return validator.ValidateNodeConfiguration(configuration)
	}
	return nil
}
