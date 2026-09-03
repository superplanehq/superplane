package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/artifactregistry"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudbuild"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/clouddns"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudfunctions"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/cloudsql"
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/compute"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/monitoring"
	gcpprometheus "github.com/superplanehq/superplane/pkg/integrations/gcp/prometheus"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/storage"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("gcp", &GCP{}, &WebhookHandler{})
}

type GCP struct{}

const (
	ConnectionMethodServiceAccountKey = "serviceAccountKey"
	ConnectionMethodWIF               = "workloadIdentityFederation"

	PubSubSecretName            = "pubsub.events.secret"
	CloudBuildSecretName        = "cloudbuild.events.secret"
	ArtifactPushSecretName      = "artifactregistry.push.secret"
	ContainerAnalysisSecretName = "containeranalysis.occurrences.secret"
	CloudBuildTopicID           = "cloud-builds"
	ArtifactPushTopicID         = "gcr"
	ContainerAnalysisTopicID    = "container-analysis-occurrences-v1"
)

type Configuration struct {
	ConnectionMethod          string `json:"connectionMethod" mapstructure:"connectionMethod"`
	ServiceAccountKey         string `json:"serviceAccountKey" mapstructure:"serviceAccountKey"`
	WorkloadIdentityProvider  string `json:"workloadIdentityProvider" mapstructure:"workloadIdentityProvider"`
	WorkloadIdentityProjectID string `json:"workloadIdentityProjectId" mapstructure:"workloadIdentityProjectId"`
}

func (g *GCP) Name() string {
	return "gcp"
}

func (g *GCP) Label() string {
	return "Google Cloud"
}

func (g *GCP) Icon() string {
	return "gcp"
}

func (g *GCP) Description() string {
	return "Manage and use Google Cloud resources in your workflows"
}

func (g *GCP) Instructions() string {
	return `## Connection method

### Service Account Key

1. Go to [IAM & Admin → Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts) in the Google Cloud Console.
2. Select a service account → **Keys** → **Add Key** → **JSON**.
3. Paste the downloaded JSON below.

### Workload Identity Federation (keyless)

1. Create a [Workload Identity Pool](https://cloud.google.com/iam/docs/workload-identity-federation) with an OIDC provider.
2. Set the **Issuer URL** to this SuperPlane instance's URL.
3. Set the **Audience** to the pool provider resource name.
4. Grant the federated identity permission to [impersonate a service account](https://cloud.google.com/iam/docs/workload-identity-federation-with-other-providers#mapping) with the roles your workflows need.
5. Enter the **pool provider resource name** and **Project ID** below.

## Required IAM roles

- ` + "`roles/logging.configWriter`" + ` — create logging sinks for event triggers
- ` + "`roles/pubsub.admin`" + ` — manage Pub/Sub topics, subscriptions, and IAM policies for event delivery
- Additional roles depending on which components you use (e.g. ` + "`roles/compute.admin`" + ` for VM management, ` + "`roles/compute.securityAdmin`" + ` to create, update, and delete firewall rules, ` + "`roles/iam.serviceAccountViewer`" + ` to populate the firewall service-account picker, ` + "`roles/monitoring.viewer`" + ` to read VM metrics, ` + "`roles/cloudsql.admin`" + ` to manage Cloud SQL databases and instances, ` + "`roles/storage.admin`" + ` to manage Cloud Storage buckets)`
}

func (g *GCP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "connectionMethod",
			Label:       "Connection method",
			Type:        configuration.FieldTypeSelect,
			Required:    true,
			Description: "Authenticate with a service account key (JSON) or Workload Identity Federation (keyless).",
			Default:     ConnectionMethodServiceAccountKey,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Service Account Key", Value: ConnectionMethodServiceAccountKey},
						{Label: "Workload Identity Federation", Value: ConnectionMethodWIF},
					},
				},
			},
		},
		{
			Name:        "serviceAccountKey",
			Label:       "Service Account Key (JSON)",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Sensitive:   true,
			Description: "Paste the full contents of your GCP service account JSON key file",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodServiceAccountKey}},
			},
		},
		{
			Name:        "workloadIdentityProvider",
			Label:       "Workload Identity Pool Provider Resource Name",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Full resource name of the OIDC provider. Must match the audience configured in the provider.",
			Placeholder: "//iam.googleapis.com/projects/123/locations/global/workloadIdentityPools/my-pool/providers/superplane",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodWIF}},
			},
		},
		{
			Name:        "workloadIdentityProjectId",
			Label:       "Project ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "GCP project ID",
			Placeholder: "e.g. my-project",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "connectionMethod", Values: []string{ConnectionMethodWIF}},
			},
		},
	}
}

func (g *GCP) Actions() []core.Action {
	return []core.Action{
		&compute.CreateVM{},
		&compute.DeleteVMInstance{},
		&compute.GetVMInstance{},
		&compute.ManageVMInstancePower{},
		&compute.UpdateVMInstanceType{},
		&compute.GetVMInstanceMetrics{},
		&compute.CreateImage{},
		&compute.UpdateImage{},
		&compute.DeleteImage{},
		&compute.CreateStaticIP{},
		&compute.DeleteStaticIP{},
		&compute.ManageStaticIP{},
		&compute.CreateLoadBalancer{},
		&compute.DeleteLoadBalancer{},
		&compute.CreateFirewall{},
		&compute.UpdateFirewall{},
		&compute.DeleteFirewall{},
		&cloudbuild.CreateBuild{},
		&cloudbuild.GetBuild{},
		&cloudbuild.RunTrigger{},
		&cloudfunctions.InvokeFunction{},
		&artifactregistry.GetArtifact{},
		&artifactregistry.GetArtifactAnalysis{},
		&gcppubsub.PublishMessage{},
		&gcppubsub.CreateTopicComponent{},
		&gcppubsub.DeleteTopicComponent{},
		&gcppubsub.CreateSubscriptionComponent{},
		&gcppubsub.DeleteSubscriptionComponent{},
		&clouddns.CreateRecord{},
		&clouddns.DeleteRecord{},
		&clouddns.UpdateRecord{},
		&monitoring.CreateAlertingPolicy{},
		&monitoring.GetAlertingPolicy{},
		&monitoring.DeleteAlertingPolicy{},
		&monitoring.UpdateAlertingPolicy{},
		&monitoring.CreateSnooze{},
		&monitoring.GetSnooze{},
		&monitoring.ExpireSnooze{},
		&cloudsql.CreateDatabase{},
		&cloudsql.GetDatabase{},
		&cloudsql.DeleteDatabase{},
		&cloudsql.CreateInstance{},
		&cloudsql.GetInstance{},
		&cloudsql.DeleteInstance{},
		&storage.CreateBucket{},
		&storage.GetBucket{},
		&storage.DeleteBucket{},
		&gcpprometheus.Query{},
		&gcpprometheus.QueryRange{},
	}
}

func (g *GCP) Triggers() []core.Trigger {
	return []core.Trigger{
		&compute.OnVMInstance{},
		&cloudbuild.OnBuildComplete{},
		&artifactregistry.OnArtifactPush{},
		&artifactregistry.OnArtifactAnalysis{},
		&gcppubsub.OnMessage{},
		&monitoring.OnAlert{},
	}
}

func (g *GCP) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	switch strings.TrimSpace(config.ConnectionMethod) {
	case ConnectionMethodServiceAccountKey:
		return g.syncServiceAccountKey(ctx, config)
	case ConnectionMethodWIF:
		return g.syncWIF(ctx, config)
	default:
		return fmt.Errorf("unknown connection method: %s", config.ConnectionMethod)
	}
}

func (g *GCP) syncWIF(ctx core.SyncContext, config Configuration) error {
	provider := strings.TrimSpace(config.WorkloadIdentityProvider)
	if provider == "" {
		return fmt.Errorf("Workload Identity Pool provider resource name is required")
	}
	projectID := strings.TrimSpace(config.WorkloadIdentityProjectID)
	if projectID == "" {
		return fmt.Errorf("Project ID is required for Workload Identity Federation")
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.Integration.ID())
	oidcToken, err := ctx.OIDC.Sign(subject, 5*time.Minute, provider, nil)
	if err != nil {
		return fmt.Errorf("failed to generate OIDC token: %w", err)
	}

	callCtx := context.Background()
	accessToken, expiresIn, err := ExchangeToken(callCtx, ctx.HTTP, oidcToken, provider)
	if err != nil {
		return fmt.Errorf("Workload Identity Federation token exchange failed. Ensure your SuperPlane instance URL is set as the OIDC issuer in GCP, the audience matches the provider resource name, and the URL is reachable by Google: %w", err)
	}

	if err := ctx.Integration.SetSecret(gcpcommon.SecretNameAccessToken, []byte(accessToken)); err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	expiresAt := time.Now().Add(expiresIn)
	refreshAfter := expiresIn / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	metadata := gcpcommon.Metadata{
		ProjectID:            projectID,
		ClientEmail:          "",
		AuthMethod:           gcpcommon.AuthMethodWIF,
		AccessTokenExpiresAt: expiresAt.Format(time.RFC3339),
	}
	ctx.Integration.SetMetadata(metadata)

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client after token exchange: %w", err)
	}
	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", projectID)
	if _, err := client.GetURL(callCtx, crmURL); err != nil {
		return fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled and the federated identity has 'Viewer' (or equivalent) on the project: %w", err)
	}

	if err := g.finishSetup(ctx, client, metadata); err != nil {
		return err
	}

	if err := ctx.Integration.ScheduleResync(refreshAfter); err != nil {
		ctx.Logger.Warnf("could not schedule GCP WIF resync: %v", err)
	}
	ctx.Integration.Ready()
	return nil
}

func (g *GCP) syncServiceAccountKey(ctx core.SyncContext, config Configuration) error {
	keyJSON, err := ctx.Integration.GetConfig("serviceAccountKey")
	if err != nil {
		return fmt.Errorf("failed to read service account key: %w", err)
	}

	if len(keyJSON) == 0 {
		return fmt.Errorf("service account key is required")
	}

	metadata, err := validateAndParseServiceAccountKey(keyJSON)
	if err != nil {
		return fmt.Errorf("invalid service account key: %w", err)
	}
	metadata.AuthMethod = gcpcommon.AuthMethodServiceAccountKey

	if err := ctx.Integration.SetSecret(gcpcommon.SecretNameServiceAccountKey, keyJSON); err != nil {
		return fmt.Errorf("failed to store service account key: %w", err)
	}

	ctx.Integration.SetMetadata(metadata)
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	crmURL := fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", metadata.ProjectID)
	if _, err := client.GetURL(context.Background(), crmURL); err != nil {
		return fmt.Errorf("connection failed. Ensure the 'Cloud Resource Manager API' is enabled on your project and the service account has 'Viewer' permissions: %w", err)
	}

	if err := g.finishSetup(ctx, client, metadata); err != nil {
		return err
	}

	ctx.Integration.Ready()
	return nil
}

// finishSetup configures the Pub/Sub event bus and best-effort Cloud Build
// and Artifact Registry subscriptions, then persists the updated metadata.
func (g *GCP) finishSetup(ctx core.SyncContext, client *gcpcommon.Client, metadata gcpcommon.Metadata) error {
	if err := g.configurePubSub(ctx, client, &metadata); err != nil {
		return fmt.Errorf("failed to configure Pub/Sub event bus: %w", err)
	}
	if err := g.configureCloudBuild(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Cloud Build subscription: %v", err)
	}
	if err := g.configureArtifactRegistry(ctx, client, &metadata); err != nil {
		ctx.Logger.Warnf("failed to configure Artifact Registry subscription: %v", err)
	}
	ctx.Integration.SetMetadata(metadata)
	return nil
}

func validateAndParseServiceAccountKey(keyJSON []byte) (gcpcommon.Metadata, error) {
	var raw map[string]any
	if err := json.Unmarshal(keyJSON, &raw); err != nil {
		return gcpcommon.Metadata{}, fmt.Errorf("invalid JSON: %w", err)
	}

	var projectID, clientEmail string

	if v, ok := raw["project_id"].(string); ok {
		projectID = strings.TrimSpace(v)
	}

	if v, ok := raw["client_email"].(string); ok {
		clientEmail = strings.TrimSpace(v)
	}

	if projectID == "" {
		return gcpcommon.Metadata{}, fmt.Errorf("missing required field project_id in service account key")
	}

	if clientEmail == "" {
		return gcpcommon.Metadata{}, fmt.Errorf("missing required field client_email in service account key")
	}

	return gcpcommon.Metadata{
		ProjectID:   projectID,
		ClientEmail: clientEmail,
	}, nil
}

func (g *GCP) configurePubSub(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	if metadata.PubSubTopic != "" {
		secret, err := g.eventsSecret(ctx.Integration)
		if err != nil {
			return fmt.Errorf("generate events secret: %w", err)
		}
		pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/events?token=%s", ctx.WebhooksBaseURL, ctx.Integration.ID(), secret)
		return gcppubsub.UpdatePushEndpoint(context.Background(), client, client.ProjectID(), metadata.PubSubSubscription, pushEndpoint)
	}

	projectID := client.ProjectID()
	reqCtx := context.Background()

	if err := ensureAPI(reqCtx, client, projectID, "pubsub.googleapis.com", "Pub/Sub"); err != nil {
		return err
	}

	secret, err := g.eventsSecret(ctx.Integration)
	if err != nil {
		return fmt.Errorf("generate events secret: %w", err)
	}

	sanitized := sanitizeID(ctx.Integration.ID().String())
	topicID := "sp-events-" + sanitized
	subscriptionID := "sp-sub-" + sanitized

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, topicID); err != nil {
		return fmt.Errorf("create Pub/Sub topic: %w", err)
	}

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/events?token=%s", ctx.WebhooksBaseURL, ctx.Integration.ID(), secret)
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, topicID, pushEndpoint); err != nil {
		return fmt.Errorf("create Pub/Sub push subscription: %w", err)
	}

	ctx.Logger.Infof("Created Pub/Sub topic %s and subscription %s for event routing", topicID, subscriptionID)

	metadata.PubSubTopic = topicID
	metadata.PubSubSubscription = subscriptionID
	return nil
}

func (g *GCP) configureCloudBuild(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	return g.ensureCloudBuildSetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, metadata)
}

func (g *GCP) configureArtifactRegistry(ctx core.SyncContext, client *gcpcommon.Client, metadata *gcpcommon.Metadata) error {
	return g.ensureArtifactRegistrySetup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, metadata)
}

func (g *GCP) ensureCloudBuildSetup(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
) error {
	projectID := client.ProjectID()

	if metadata.CloudBuildSubscription != "" {
		secret, err := g.cloudBuildSecret(integration)
		if err != nil {
			return fmt.Errorf("generate cloud build secret: %w", err)
		}
		pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/cloud-build-events?token=%s", webhooksBaseURL, integration.ID(), secret)
		return gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.CloudBuildSubscription, pushEndpoint)
	}

	if err := ensureAPI(reqCtx, client, projectID, "pubsub.googleapis.com", "Pub/Sub"); err != nil {
		return err
	}
	if err := ensureAPI(reqCtx, client, projectID, "cloudbuild.googleapis.com", "Cloud Build"); err != nil {
		return err
	}

	secret, err := g.cloudBuildSecret(integration)
	if err != nil {
		return fmt.Errorf("generate cloud build secret: %w", err)
	}

	sanitized := sanitizeID(integration.ID().String())
	subscriptionID := "sp-cb-sub-" + sanitized
	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/cloud-build-events?token=%s", webhooksBaseURL, integration.ID(), secret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, CloudBuildTopicID); err != nil {
		return fmt.Errorf("create Cloud Build topic: %w", err)
	}

	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, CloudBuildTopicID, pushEndpoint); err != nil {
		return fmt.Errorf("create Cloud Build push subscription: %w", err)
	}

	metadata.CloudBuildSubscription = subscriptionID
	return nil
}

func (g *GCP) ensureArtifactRegistrySetup(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
) error {
	projectID := client.ProjectID()

	if metadata.ArtifactPushSubscription != "" {
		synced, err := g.syncArtifactRegistrySubscriptions(reqCtx, client, integration, webhooksBaseURL, metadata, projectID)
		if err != nil {
			return err
		}
		if synced {
			return nil
		}
	}

	return g.bootstrapArtifactRegistrySubscriptions(reqCtx, client, integration, webhooksBaseURL, metadata, projectID)
}

func (g *GCP) syncArtifactRegistrySubscriptions(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
	projectID string,
) (bool, error) {
	secret, err := g.artifactPushSecret(integration)
	if err != nil {
		return false, fmt.Errorf("generate artifact push secret: %w", err)
	}

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-push-events?token=%s", webhooksBaseURL, integration.ID(), secret)
	updateErr := gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.ArtifactPushSubscription, pushEndpoint)
	if updateErr != nil {
		if !gcpcommon.IsNotFoundError(updateErr) {
			return false, fmt.Errorf("update artifact push endpoint: %w", updateErr)
		}
		// Subscription no longer exists in GCP — recreate everything.
		metadata.ArtifactPushSubscription = ""
		metadata.ContainerAnalysisSubscription = ""
		return false, nil
	}

	if metadata.ContainerAnalysisSubscription != "" {
		caSecret, err := g.containerAnalysisSecret(integration)
		if err != nil {
			return false, fmt.Errorf("generate container analysis secret: %w", err)
		}
		caPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-analysis-events?token=%s", webhooksBaseURL, integration.ID(), caSecret)
		caUpdateErr := gcppubsub.UpdatePushEndpoint(reqCtx, client, projectID, metadata.ContainerAnalysisSubscription, caPushEndpoint)
		if caUpdateErr == nil {
			return true, nil
		}
		if !gcpcommon.IsNotFoundError(caUpdateErr) {
			return false, fmt.Errorf("update container analysis endpoint: %w", caUpdateErr)
		}
		// Subscription no longer exists in GCP — recreate it below.
		metadata.ContainerAnalysisSubscription = ""
	}

	caEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "containeranalysis.googleapis.com")
	if err != nil {
		return false, fmt.Errorf("check Container Analysis API: %w", err)
	}
	if !caEnabled {
		return true, nil
	}

	sanitized := sanitizeID(integration.ID().String())
	caSubscriptionID := "sp-ca-sub-" + sanitized
	if err := g.createContainerAnalysisSubscription(reqCtx, client, projectID, integration, webhooksBaseURL, caSubscriptionID); err != nil {
		return false, err
	}
	metadata.ContainerAnalysisSubscription = caSubscriptionID
	return true, nil
}

func (g *GCP) bootstrapArtifactRegistrySubscriptions(
	reqCtx context.Context,
	client *gcpcommon.Client,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	metadata *gcpcommon.Metadata,
	projectID string,
) error {
	if err := ensureAPI(reqCtx, client, projectID, "pubsub.googleapis.com", "Pub/Sub"); err != nil {
		return err
	}
	if err := ensureAPI(reqCtx, client, projectID, "artifactregistry.googleapis.com", "Artifact Registry"); err != nil {
		return err
	}

	sanitized := sanitizeID(integration.ID().String())

	arSecret, err := g.artifactPushSecret(integration)
	if err != nil {
		return fmt.Errorf("generate artifact push secret: %w", err)
	}
	arSubscriptionID := "sp-ar-sub-" + sanitized
	arPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-push-events?token=%s", webhooksBaseURL, integration.ID(), arSecret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, ArtifactPushTopicID); err != nil {
		return fmt.Errorf("create Artifact Registry gcr topic: %w", err)
	}
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, arSubscriptionID, ArtifactPushTopicID, arPushEndpoint); err != nil {
		return fmt.Errorf("create Artifact Registry push subscription: %w", err)
	}
	metadata.ArtifactPushSubscription = arSubscriptionID

	caEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "containeranalysis.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Container Analysis API: %w", err)
	}
	if !caEnabled {
		return nil
	}

	caSubscriptionID := "sp-ca-sub-" + sanitized
	if err := g.createContainerAnalysisSubscription(reqCtx, client, projectID, integration, webhooksBaseURL, caSubscriptionID); err != nil {
		return err
	}
	metadata.ContainerAnalysisSubscription = caSubscriptionID

	return nil
}

func (g *GCP) createContainerAnalysisSubscription(
	reqCtx context.Context,
	client *gcpcommon.Client,
	projectID string,
	integration core.IntegrationContext,
	webhooksBaseURL string,
	subscriptionID string,
) error {
	caSecret, err := g.containerAnalysisSecret(integration)
	if err != nil {
		return fmt.Errorf("generate container analysis secret: %w", err)
	}
	caPushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/artifact-analysis-events?token=%s", webhooksBaseURL, integration.ID(), caSecret)

	if err := gcppubsub.CreateTopic(reqCtx, client, projectID, ContainerAnalysisTopicID); err != nil {
		return fmt.Errorf("create Container Analysis topic: %w", err)
	}
	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, subscriptionID, ContainerAnalysisTopicID, caPushEndpoint); err != nil {
		return fmt.Errorf("create Container Analysis push subscription: %w", err)
	}

	return nil
}

func (g *GCP) cloudBuildSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, CloudBuildSecretName)
}

func (g *GCP) artifactPushSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, ArtifactPushSecretName)
}

func (g *GCP) containerAnalysisSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, ContainerAnalysisSecretName)
}

func (g *GCP) getOrCreateSecret(integration core.IntegrationContext, secretName string) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == secretName {
			return string(s.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	if err := integration.SetSecret(secretName, []byte(secret)); err != nil {
		return "", fmt.Errorf("store secret %s: %w", secretName, err)
	}
	return secret, nil
}

func (g *GCP) eventsSecret(integration core.IntegrationContext) (string, error) {
	return g.getOrCreateSecret(integration, PubSubSecretName)
}

// ensureAPI verifies that a Google API is enabled in the project, returning
// an actionable error with the API's console URL when it is not.
func ensureAPI(reqCtx context.Context, client *gcpcommon.Client, projectID, service, label string) error {
	enabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, service)
	if err != nil {
		return fmt.Errorf("check %s API: %w", label, err)
	}
	if !enabled {
		return fmt.Errorf("%s API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/%s?project=%s", label, projectID, service, projectID)
	}
	return nil
}

func sanitizeID(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if len(result) > 40 {
		result = result[:40]
	}
	return result
}

func (g *GCP) Cleanup(ctx core.IntegrationCleanupContext) error {
	var m gcpcommon.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &m); err != nil || m.ProjectID == "" {
		return nil
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		ctx.Logger.Warnf("failed to create GCP client for cleanup: %v", err)
		return nil
	}

	reqCtx := context.Background()
	subscriptions := []struct{ label, id string }{
		{"Pub/Sub subscription", m.PubSubSubscription},
		{"Cloud Build subscription", m.CloudBuildSubscription},
		{"Artifact Registry push subscription", m.ArtifactPushSubscription},
		{"Container Analysis subscription", m.ContainerAnalysisSubscription},
	}
	for _, sub := range subscriptions {
		if sub.id == "" {
			continue
		}
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, sub.id); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete %s %s: %v", sub.label, sub.id, err)
			}
		}
	}
	if m.PubSubTopic != "" {
		if err := gcppubsub.DeleteTopic(reqCtx, client, m.ProjectID, m.PubSubTopic); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Pub/Sub topic %s: %v", m.PubSubTopic, err)
			}
		}
	}

	return nil
}

func (g *GCP) Hooks() []core.Hook {
	return []core.Hook{
		{Name: gcpcommon.ActionNameEnsureCloudBuild, Type: core.HookTypeInternal},
		{Name: gcpcommon.ActionNameEnsureArtifactRegistry, Type: core.HookTypeInternal},
		{Name: gcpcommon.ActionNameEnsurePubSubOnMessage, Type: core.HookTypeInternal},
	}
}

func (g *GCP) HandleHook(ctx core.IntegrationHookContext) error {
	switch ctx.Name {
	case gcpcommon.ActionNameEnsureCloudBuild:
		return g.handleEnsureCloudBuild(ctx)
	case gcpcommon.ActionNameEnsureArtifactRegistry:
		return g.handleEnsureArtifactRegistry(ctx)
	case gcpcommon.ActionNameEnsurePubSubOnMessage:
		return g.handleEnsurePubSubOnMessage(ctx)
	default:
		return fmt.Errorf("unknown hook: %s", ctx.Name)
	}
}

func (g *GCP) handleEnsurePubSubOnMessage(ctx core.IntegrationHookContext) error {
	var params struct {
		Topic      string `mapstructure:"topic"`
		GCPSubName string `mapstructure:"gcpSubName"`
	}
	if err := mapstructure.Decode(ctx.Parameters, &params); err != nil {
		return fmt.Errorf("failed to decode action params: %w", err)
	}
	if params.Topic == "" || params.GCPSubName == "" {
		return fmt.Errorf("topic and gcpSubName are required")
	}

	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	projectID := client.ProjectID()
	secret, err := g.eventsSecret(ctx.Integration)
	if err != nil {
		return fmt.Errorf("get events secret: %w", err)
	}

	reqCtx := context.Background()

	// Delete existing subscription (handles topic changes and idempotency)
	_ = gcppubsub.DeleteSubscription(reqCtx, client, projectID, params.GCPSubName)

	pushEndpoint := fmt.Sprintf("%s/api/v1/integrations/%s/pubsub-events?token=%s&gcpSubName=%s",
		ctx.WebhooksBaseURL, ctx.Integration.ID(), secret, params.GCPSubName)

	if err := gcppubsub.CreatePushSubscription(reqCtx, client, projectID, params.GCPSubName, params.Topic, pushEndpoint); err != nil {
		return fmt.Errorf("create push subscription on topic %q: %w", params.Topic, err)
	}

	ctx.Logger.Infof("Created Pub/Sub push subscription %s on topic %s", params.GCPSubName, params.Topic)
	return nil
}

func (g *GCP) handleEnsureCloudBuild(ctx core.IntegrationHookContext) error {
	return g.ensureSubscriptionSetup(ctx, g.ensureCloudBuildSetup)
}

func (g *GCP) handleEnsureArtifactRegistry(ctx core.IntegrationHookContext) error {
	return g.ensureSubscriptionSetup(ctx, g.ensureArtifactRegistrySetup)
}

// ensureSubscriptionSetup creates the client, runs the given subscription
// setup step against the current metadata, and persists the result.
func (g *GCP) ensureSubscriptionSetup(
	ctx core.IntegrationHookContext,
	setup func(reqCtx context.Context, client *gcpcommon.Client, integration core.IntegrationContext, webhooksBaseURL string, metadata *gcpcommon.Metadata) error,
) error {
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create GCP client: %w", err)
	}

	var metadata gcpcommon.Metadata
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode integration metadata: %w", err)
	}

	if err := setup(context.Background(), client, ctx.Integration, ctx.WebhooksBaseURL, &metadata); err != nil {
		return err
	}

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (g *GCP) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	client, err := gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return nil, err
	}
	reqCtx := context.Background()

	p := ctx.Parameters

	switch resourceType {
	case cloudfunctions.ResourceTypeLocation, cloudfunctions.ResourceTypeFunction:
		projectID := p["projectId"]
		if projectID == "" {
			projectID = client.ProjectID()
		}
		cfEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "cloudfunctions.googleapis.com")
		if err != nil {
			return nil, fmt.Errorf("failed to check Cloud Functions API status: %w", err)
		}
		crEnabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "run.googleapis.com")
		if err != nil {
			return nil, fmt.Errorf("failed to check Cloud Run API status: %w", err)
		}
		if !cfEnabled && !crEnabled {
			return nil, fmt.Errorf("Neither Cloud Functions nor Cloud Run API is enabled in project %s", projectID)
		}
		if resourceType == cloudfunctions.ResourceTypeLocation {
			return cloudfunctions.ListLocationResources(reqCtx, client, p["projectId"])
		}
		return cloudfunctions.ListFunctionResources(reqCtx, client, p["projectId"], p["location"])
	case compute.ResourceTypeRegion:
		return compute.ListRegionResources(reqCtx, client)
	case compute.ResourceTypeZone:
		return compute.ListZoneResources(reqCtx, client, p["region"])
	case compute.ResourceTypeMachineFamily:
		return compute.ListMachineFamilyResources(reqCtx, client, p["zone"])
	case compute.ResourceTypeMachineType:
		return compute.ListMachineTypeResources(reqCtx, client, p["zone"], p["machineFamily"])
	case compute.ResourceTypeInstanceMachineType:
		return compute.ListMachineTypeResourcesForInstance(reqCtx, client, p["instance"])
	case compute.ResourceTypePublicImages:
		return compute.ListPublicImageResources(reqCtx, client, p["project"])
	case compute.ResourceTypeCustomImages:
		return compute.ListCustomImageResources(reqCtx, client, p["project"])
	case compute.ResourceTypeImageStorageLocation:
		return compute.ListImageStorageLocationResources(reqCtx, client)
	case compute.ResourceTypeSnapshots:
		return compute.ListSnapshotResources(reqCtx, client, p["project"])
	case compute.ResourceTypeDisks:
		return compute.ListDiskResources(reqCtx, client, p["project"], p["zone"])
	case compute.ResourceTypeDiskTypes:
		return compute.ListDiskTypeResources(reqCtx, client, p["project"], p["zone"], p["bootDiskOnly"] == "true")
	case compute.ResourceTypeSnapshotSchedules:
		return compute.ListSnapshotScheduleResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeNetwork:
		return compute.ListNetworkResources(reqCtx, client, p["project"])
	case compute.ResourceTypeSubnetwork:
		return compute.ListSubnetworkResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeAddress:
		return compute.ListAddressResources(reqCtx, client, p["project"], p["region"])
	case compute.ResourceTypeStaticIP:
		return compute.ListStaticIPResources(reqCtx, client, p["project"], p["instance"])
	case compute.ResourceTypeInstanceGroup:
		return compute.ListInstanceGroupResources(reqCtx, client, p["project"])
	case compute.ResourceTypeForwardingRule:
		return compute.ListForwardingRuleResources(reqCtx, client, p["project"])
	case compute.ResourceTypeFirewall:
		return compute.ListFirewallResources(reqCtx, client, p["project"])
	case compute.ResourceTypeServiceAccount:
		return compute.ListServiceAccountResources(reqCtx, client, p["project"])
	case compute.ResourceTypeInstance:
		return compute.ListInstanceResources(reqCtx, client, p["project"])
	case clouddns.ResourceTypeManagedZone:
		return clouddns.ListManagedZoneResources(reqCtx, client, p["projectId"])
	case monitoring.ResourceTypeAlertPolicy:
		return monitoring.ListAlertingPolicyResources(reqCtx, client)
	case monitoring.ResourceTypeNotificationChannel:
		return monitoring.ListNotificationChannelResources(reqCtx, client)
	case monitoring.ResourceTypeSnooze:
		return monitoring.ListSnoozeResources(reqCtx, client)
	case cloudbuild.ResourceTypeTrigger:
		return cloudbuild.ListTriggerResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeBuild:
		return cloudbuild.ListBuildResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeLocation:
		return cloudbuild.ListLocationResources(reqCtx, client, p["projectId"])
	case cloudbuild.ResourceTypeConnection:
		return cloudbuild.ListConnectionResources(reqCtx, client, p["projectId"], p["location"])
	case cloudbuild.ResourceTypeRepository:
		return cloudbuild.ListRepositoryResources(reqCtx, client, p["connection"])
	case cloudbuild.ResourceTypeBranch:
		return cloudbuild.ListBranchResources(reqCtx, client, p["repository"])
	case cloudbuild.ResourceTypeTag:
		return cloudbuild.ListTagResources(reqCtx, client, p["repository"])
	case artifactregistry.ResourceTypeLocation:
		return artifactregistry.ListLocationResources(reqCtx, client, p["projectId"])
	case artifactregistry.ResourceTypeRepository:
		return artifactregistry.ListRepositoryResources(reqCtx, client, p["projectId"], p["location"])
	case artifactregistry.ResourceTypePackage:
		return artifactregistry.ListPackageResources(reqCtx, client, p["projectId"], p["location"], p["repository"])
	case artifactregistry.ResourceTypeVersion:
		return artifactregistry.ListVersionResources(reqCtx, client, p["projectId"], p["location"], p["repository"], p["package"])
	case gcppubsub.ResourceTypeTopic:
		return gcppubsub.ListTopicResources(reqCtx, client)
	case gcppubsub.ResourceTypeSubscription:
		return gcppubsub.ListSubscriptionResources(reqCtx, client, p["topic"])
	case cloudsql.ResourceTypeInstance:
		return cloudsql.ListInstanceResources(reqCtx, client)
	case cloudsql.ResourceTypeDatabase:
		return cloudsql.ListDatabaseResources(reqCtx, client, p["instance"])
	case cloudsql.ResourceTypeRegion:
		return cloudsql.ListRegionResources(reqCtx, client)
	case cloudsql.ResourceTypeTier:
		return cloudsql.ListTierResources(reqCtx, client, p["region"])
	case storage.ResourceTypeBucket:
		return storage.ListBucketResources(reqCtx, client)
	default:
		return nil, nil
	}
}

func (g *GCP) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		g.handleEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/cloud-build-events") {
		g.handleCloudBuildEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/artifact-push-events") {
		g.handleArtifactPushEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/artifact-analysis-events") {
		g.handleArtifactAnalysisEvent(ctx)
		return
	}

	if strings.HasSuffix(ctx.Request.URL.Path, "/pubsub-events") {
		g.handlePubSubEvent(ctx)
		return
	}

	ctx.Response.WriteHeader(http.StatusNotFound)
}

// AuditLogEvent is the normalized event structure extracted from a Cloud Logging
// audit log entry, used both for subscription pattern matching and as the message
// payload delivered to triggers via OnIntegrationMessage.
type AuditLogEvent struct {
	ServiceName  string `json:"serviceName" mapstructure:"serviceName"`
	MethodName   string `json:"methodName" mapstructure:"methodName"`
	ResourceName string `json:"resourceName" mapstructure:"resourceName"`
	LogName      string `json:"logName" mapstructure:"logName"`
	Timestamp    string `json:"timestamp" mapstructure:"timestamp"`
	InsertID     string `json:"insertId" mapstructure:"insertId"`
	Data         any    `json:"data" mapstructure:"data"`
}

// AuditLogEventPattern is the subscription pattern used to match incoming events
// against trigger subscriptions. Only non-empty fields are matched.
type AuditLogEventPattern struct {
	ServiceName string `json:"serviceName" mapstructure:"serviceName"`
	MethodName  string `json:"methodName" mapstructure:"methodName"`
}

type pubsubPushMessage struct {
	Message struct {
		Data        string            `json:"data"`
		MessageID   string            `json:"messageId"`
		PublishTime string            `json:"publishTime"`
		Attributes  map[string]string `json:"attributes"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type logEntryProtoPayload struct {
	ServiceName  string `json:"serviceName"`
	MethodName   string `json:"methodName"`
	ResourceName string `json:"resourceName"`
}

type logEntry struct {
	ProtoPayload logEntryProtoPayload `json:"protoPayload"`
	LogName      string               `json:"logName"`
	Timestamp    string               `json:"timestamp"`
	InsertID     string               `json:"insertId"`
}

// handlePush authenticates a Pub/Sub push request against the named secret,
// decodes the push envelope, and fans the event out to matching
// subscriptions. It preserves the endpoint contract: 400 for a missing token,
// missing required parameter, or malformed envelope; 500 for secret storage
// errors; 403 on a token mismatch; and 200 (with a log line) for undecodable
// message data, since Pub/Sub treats non-2xx as redelivery. sendModifier
// completes "error sending <modifier>message to subscription" in logs.
func (g *GCP) handlePush(
	ctx core.HTTPRequestContext,
	secretName string,
	requireParam func(*http.Request) bool,
	decodeWarn string,
	decode func(pushMsg pubsubPushMessage, decoded []byte) (event any, ok bool),
	applies func(subscription core.IntegrationSubscriptionContext, event any) bool,
	sendModifier string,
) {
	token := ctx.Request.URL.Query().Get("token")
	if token == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == secretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		return
	}

	if requireParam != nil && !requireParam(ctx.Request) {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	var pushMsg pubsubPushMessage
	if err := json.Unmarshal(body, &pushMsg); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		return
	}

	decoded, err := base64Decode(pushMsg.Message.Data)
	if err != nil {
		ctx.Logger.Warnf("%s: %v", decodeWarn, err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	event, ok := decode(pushMsg, decoded)
	if !ok {
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !applies(subscription, event) {
			continue
		}
		if err := subscription.SendMessage(event); err != nil {
			ctx.Logger.Errorf("error sending %smessage to subscription: %v", sendModifier, err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (g *GCP) handleEvent(ctx core.HTTPRequestContext) {
	g.handlePush(ctx, PubSubSecretName, nil, "failed to decode Pub/Sub message data",
		func(_ pubsubPushMessage, decoded []byte) (any, bool) {
			var entry logEntry
			if err := json.Unmarshal(decoded, &entry); err != nil {
				ctx.Logger.Warnf("failed to parse log entry: %v", err)
				return nil, false
			}

			var rawData map[string]any
			_ = json.Unmarshal(decoded, &rawData)

			return AuditLogEvent{
				ServiceName:  entry.ProtoPayload.ServiceName,
				MethodName:   strings.TrimSpace(entry.ProtoPayload.MethodName),
				ResourceName: entry.ProtoPayload.ResourceName,
				LogName:      entry.LogName,
				Timestamp:    entry.Timestamp,
				InsertID:     entry.InsertID,
				Data:         rawData,
			}, true
		},
		func(subscription core.IntegrationSubscriptionContext, event any) bool {
			auditEvent, ok := event.(AuditLogEvent)
			return ok && g.subscriptionApplies(subscription, auditEvent)
		},
		"")
}

func (g *GCP) subscriptionApplies(subscription core.IntegrationSubscriptionContext, event AuditLogEvent) bool {
	var pattern AuditLogEventPattern
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}

	if pattern.ServiceName != "" && pattern.ServiceName != event.ServiceName {
		return false
	}

	if pattern.MethodName != "" && pattern.MethodName != event.MethodName {
		return false
	}

	return true
}

func (g *GCP) handleCloudBuildEvent(ctx core.HTTPRequestContext) {
	g.handlePush(ctx, CloudBuildSecretName, nil, "failed to decode Cloud Build Pub/Sub message data",
		func(_ pubsubPushMessage, decoded []byte) (any, bool) {
			var build map[string]any
			if err := json.Unmarshal(decoded, &build); err != nil {
				ctx.Logger.Warnf("failed to parse Cloud Build notification: %v", err)
				return nil, false
			}
			return build, true
		},
		func(subscription core.IntegrationSubscriptionContext, _ any) bool {
			return subscriptionOfType(subscription, cloudbuild.SubscriptionType)
		},
		"cloud build ")
}

// subscriptionOfType reports whether the subscription's configuration type
// matches the given subscription type.
func subscriptionOfType(subscription core.IntegrationSubscriptionContext, subscriptionType string) bool {
	var pattern struct {
		Type string `mapstructure:"type"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == subscriptionType
}

func (g *GCP) handleArtifactPushEvent(ctx core.HTTPRequestContext) {
	g.handlePush(ctx, ArtifactPushSecretName, nil, "failed to decode Artifact Registry Pub/Sub message data",
		func(_ pubsubPushMessage, decoded []byte) (any, bool) {
			var event map[string]any
			if err := json.Unmarshal(decoded, &event); err != nil {
				ctx.Logger.Warnf("failed to parse Artifact Registry push event: %v", err)
				return nil, false
			}
			return event, true
		},
		func(subscription core.IntegrationSubscriptionContext, _ any) bool {
			return subscriptionOfType(subscription, artifactregistry.ArtifactPushSubscriptionType)
		},
		"artifact push ")
}

func (g *GCP) handleArtifactAnalysisEvent(ctx core.HTTPRequestContext) {
	g.handlePush(ctx, ContainerAnalysisSecretName, nil, "failed to decode Container Analysis Pub/Sub message data",
		func(_ pubsubPushMessage, decoded []byte) (any, bool) {
			var occurrence map[string]any
			if err := json.Unmarshal(decoded, &occurrence); err != nil {
				ctx.Logger.Warnf("failed to parse Container Analysis occurrence: %v", err)
				return nil, false
			}
			return occurrence, true
		},
		func(subscription core.IntegrationSubscriptionContext, _ any) bool {
			return subscriptionOfType(subscription, artifactregistry.ArtifactAnalysisSubscriptionType)
		},
		"container analysis ")
}

func (g *GCP) handlePubSubEvent(ctx core.HTTPRequestContext) {
	gcpSubName := ctx.Request.URL.Query().Get("gcpSubName")
	g.handlePush(ctx, PubSubSecretName,
		func(r *http.Request) bool { return r.URL.Query().Get("gcpSubName") != "" },
		"failed to decode Pub/Sub user message data",
		func(pushMsg pubsubPushMessage, decoded []byte) (any, bool) {
			var msgData any
			if err := json.Unmarshal(decoded, &msgData); err != nil {
				// Non-JSON payloads: deliver as raw string
				msgData = string(decoded)
			}

			return map[string]any{
				"messageId":   pushMsg.Message.MessageID,
				"publishTime": pushMsg.Message.PublishTime,
				"data":        msgData,
				"attributes":  pushMsg.Message.Attributes,
			}, true
		},
		func(subscription core.IntegrationSubscriptionContext, event any) bool {
			return g.pubsubOnMessageSubscriptionApplies(subscription, gcpSubName)
		},
		"pub/sub ")
}

func (g *GCP) pubsubOnMessageSubscriptionApplies(subscription core.IntegrationSubscriptionContext, gcpSubName string) bool {
	var pattern struct {
		Type       string `mapstructure:"type"`
		GCPSubName string `mapstructure:"gcpSubName"`
	}
	if err := mapstructure.Decode(subscription.Configuration(), &pattern); err != nil {
		return false
	}
	return pattern.Type == gcppubsub.OnMessageSubscriptionType && pattern.GCPSubName == gcpSubName
}

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
