package factories

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	testcontexts "github.com/superplanehq/superplane/test/support/contexts"
	"google.golang.org/protobuf/proto"
)

// Restore a UI-created intake using existing hosted grants. The real GitHub
// setup and event persistence exercise credentials without changing their scope.
func TestFactoryConfigurationRestoresHostedGitHubIntake(t *testing.T) {
	f := newConfigurationFixture(t)
	db := database.DB(t.Context())
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	t.Setenv(config.EnvGitHubAppID, "123")
	t.Setenv(config.EnvGitHubAppSlug, "superplane-test")
	t.Setenv(config.EnvGitHubAppPrivateKey, string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})))
	t.Setenv(config.EnvGitHubAppWebhookSecret, "test-webhook-secret")
	const installationID = int64(101)
	require.NoError(t, models.UpsertVCSProviderInstallation(db, &models.VCSProviderInstallation{Provider: models.ProviderGitHub, InstallationID: installationID, AccountLogin: "acme"}))
	require.NoError(t, models.ReplaceVCSProviderRepositories(db, models.ProviderGitHub, installationID, []models.VCSProviderRepository{
		{RepositoryID: 201, FullName: "acme/app", DefaultBranch: "main"},
		{RepositoryID: 202, FullName: "acme/backlog", DefaultBranch: "main"},
		{RepositoryID: 203, FullName: "acme/ungranted", DefaultBranch: "main"},
	}))
	hosted, err := models.FindOrCreateVCSProviderBinding(db, f.factory.OrganizationID, models.ProviderGitHub, installationID, "acme")
	require.NoError(t, err)
	integrationID := hosted.ID.String()
	primaryID, backlogID := int64(201), int64(202)
	require.NoError(t, f.factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{VCSIntegrationID: &integrationID, AppRepositoryID: &primaryID, BacklogRepositoryID: &backlogID}))
	require.NoError(t, models.SyncVCSProviderBindingRepositories(db, hosted.ID, models.ProviderGitHub))
	grants, err := models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Len(t, grants, 2)
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
	created, err := CreateFactoryIntake(f.ctx, testIntakeDependencies(f.r), f.factory.OrganizationID.String(), &pb.CreateFactoryIntakeRequest{
		FactoryId: f.factory.ID.String(), Name: "Hosted issues", Source: pb.FactoryIntake_SOURCE_GITHUB_ISSUES,
		IntegrationId: integrationID, ResourceId: "acme/backlog", SkipInitialImport: true,
	})
	require.NoError(t, err)
	original, err := f.factory.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, original, 1)
	bundle := f.export(t)
	_, err = DeleteFactoryIntake(f.ctx, f.factory.OrganizationID.String(), &pb.DeleteFactoryIntakeRequest{FactoryId: f.factory.ID.String(), IntakeId: created.Intake.Id})
	require.NoError(t, err)
	before := configurationResourceCounts(t)
	client.tokenGrants = nil
	ungranted := proto.Clone(bundle).(*pb.FactoryConfigurationBundle)
	editConfigurationManifest(t, ungranted, func(m *factoryConfigurationManifest) {
		for i := range m.Stages {
			for j := range m.Stages[i].Automations {
				automation := &m.Stages[i].Automations[j]
				if automation.Uses == "@superplane/intake/github-issues" {
					automation.Binding.ResourceID = "acme/ungranted"
				}
			}
		}
	})
	denied := f.plan(t, ungranted)
	require.Len(t, denied.Diagnostics, 1, "restoration must use an existing repository grant")
	require.Contains(t, denied.Diagnostics[0].Field, "binding.resource_id")
	require.Empty(t, denied.BaseStateToken)
	plan := f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Len(t, plan.Changes, 1)
	require.Equal(t, before, configurationResourceCounts(t))
	require.Empty(t, client.tokenGrants)
	applied, err := f.apply(bundle, plan.BaseStateToken)
	require.NoError(t, err)
	require.True(t, applied.Changed)
	restored, err := f.factory.ListIntakes(db)
	require.NoError(t, err)
	require.Len(t, restored, 1)
	require.NotEqual(t, original[0].CanvasID, restored[0].CanvasID)
	require.Equal(t, models.FactoryIntakeInitialImportStatusSkipped, restored[0].InitialImportStatus)
	live := resetTestLiveVersion(t, restored[0].Canvas)
	trigger := findIntakeNode(live.Nodes, intakeTriggerNodeID)
	require.NotNil(t, trigger)
	require.Equal(t, "acme/backlog", trigger.Configuration["repository"])
	require.Equal(t, integrationID, *trigger.IntegrationID)
	require.Equal(t, []string{"acme/backlog"}, configurationWebhookRepositories(t, restored[0].Canvas, trigger.ID))
	for _, node := range live.Nodes {
		require.Nil(t, node.ErrorMessage, node.ID)
	}
	require.NotEmpty(t, client.tokenGrants)
	for _, selected := range client.tokenGrants {
		require.ElementsMatch(t, []int64{201, 202}, selected)
	}
	afterGrants, err := models.ListVCSProviderBindingRepositories(db, hosted.ID)
	require.NoError(t, err)
	require.Equal(t, grants, afterGrants)

	// Deliver an issue through the restored trigger and persist its root event.
	node, err := models.FindCanvasNode(db, restored[0].CanvasID, trigger.ID)
	require.NoError(t, err)
	body := []byte(`{"action":"opened","issue":{"number":99,"title":"New issue","state":"open","body":"Details","labels":[{"name":"superplane"}]},"repository":{"full_name":"acme/backlog"}}`)
	signature := hmac.New(sha256.New, []byte("test-webhook-secret"))
	signature.Write(body)
	issueTrigger, err := f.r.Registry.GetTrigger("github.onIssue")
	require.NoError(t, err)
	code, _, err := issueTrigger.HandleWebhook(core.WebhookRequestContext{
		Body: body, Headers: http.Header{"X-Github-Event": []string{"issues"}, "X-Hub-Signature-256": []string{fmt.Sprintf("sha256=%x", signature.Sum(nil))}},
		Configuration: node.Configuration.Data(), Logger: logrus.NewEntry(logrus.New()),
		Webhook: &testcontexts.NodeWebhookContext{Secret: "test-webhook-secret"},
		Events:  contexts.NewEventContext(db, node, nil, nil),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, code)
	var events int64
	require.NoError(t, db.Model(&models.CanvasEvent{}).Where("workflow_id = ? AND node_id = ?", restored[0].CanvasID, trigger.ID).Count(&events).Error)
	require.EqualValues(t, 1, events)
	resources := configurationResourceCounts(t)
	revision := configurationRevisionCount(t, restored[0].Canvas)
	plan = f.plan(t, bundle)
	require.Empty(t, plan.Diagnostics)
	require.Empty(t, plan.Changes)
	writes := configurationCountApplyWrites(t, func() {
		result, err := f.apply(bundle, plan.BaseStateToken)
		require.NoError(t, err)
		require.False(t, result.Changed)
	})
	require.Zero(t, writes)
	require.Equal(t, resources, configurationResourceCounts(t))
	require.Equal(t, revision, configurationRevisionCount(t, restored[0].Canvas))
}

type configurationHostedRepositoryHTTP struct {
	configurationCatalogHTTP
	tokenGrants [][]int64
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
