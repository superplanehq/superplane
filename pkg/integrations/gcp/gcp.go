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
	gcpcommon "github.com/superplanehq/superplane/pkg/integrations/gcp/common"
	"github.com/superplanehq/superplane/pkg/integrations/gcp/compute"
	gcppubsub "github.com/superplanehq/superplane/pkg/integrations/gcp/pubsub"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterIntegration("gcp", &GCP{})
	compute.SetClientFactory(func(ctx core.ExecutionContext) (compute.Client, error) {
		return gcpcommon.NewClient(ctx.HTTP, ctx.Integration)
	})
}

type GCP struct{}

const (
	ConnectionMethodServiceAccountKey = "serviceAccountKey"
	ConnectionMethodWIF               = "workloadIdentityFederation"

	PubSubSecretName = "pubsub.events.secret"
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
- Additional roles depending on which components you use (e.g. ` + "`roles/compute.admin`" + ` for VM management)`
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

func (g *GCP) Components() []core.Component {
	return []core.Component{
		&compute.CreateVM{},
	}
}

func (g *GCP) Triggers() []core.Trigger {
	return []core.Trigger{
		&compute.OnVMInstance{},
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

	if err := g.configurePubSub(ctx, client, &metadata); err != nil {
		return fmt.Errorf("failed to configure Pub/Sub event bus: %w", err)
	}
	ctx.Integration.SetMetadata(metadata)

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

	if err := g.configurePubSub(ctx, client, &metadata); err != nil {
		return fmt.Errorf("failed to configure Pub/Sub event bus: %w", err)
	}
	ctx.Integration.SetMetadata(metadata)

	ctx.Integration.Ready()
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
		return nil
	}

	projectID := client.ProjectID()
	reqCtx := context.Background()

	enabled, err := gcppubsub.IsAPIEnabled(reqCtx, client, projectID, "pubsub.googleapis.com")
	if err != nil {
		return fmt.Errorf("check Pub/Sub API: %w", err)
	}
	if !enabled {
		return fmt.Errorf("Pub/Sub API is not enabled in project %s. Enable it at https://console.cloud.google.com/apis/library/pubsub.googleapis.com?project=%s", projectID, projectID)
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

func (g *GCP) eventsSecret(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, s := range secrets {
		if s.Name == PubSubSecretName {
			return string(s.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("generate random secret: %w", err)
	}

	if err := integration.SetSecret(PubSubSecretName, []byte(secret)); err != nil {
		return "", fmt.Errorf("store events secret: %w", err)
	}
	return secret, nil
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
	if m.PubSubSubscription != "" {
		if err := gcppubsub.DeleteSubscription(reqCtx, client, m.ProjectID, m.PubSubSubscription); err != nil {
			if !gcpcommon.IsNotFoundError(err) {
				ctx.Logger.Warnf("failed to delete Pub/Sub subscription %s: %v", m.PubSubSubscription, err)
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

func (g *GCP) Actions() []core.Action {
	return nil
}

func (g *GCP) HandleAction(ctx core.IntegrationActionContext) error {
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
	case compute.ResourceTypeRegion:
		return compute.ListRegionResources(reqCtx, client)
	case compute.ResourceTypeZone:
		return compute.ListZoneResources(reqCtx, client, p["region"])
	case compute.ResourceTypeMachineFamily:
		return compute.ListMachineFamilyResources(reqCtx, client, p["zone"])
	case compute.ResourceTypeMachineType:
		return compute.ListMachineTypeResources(reqCtx, client, p["zone"], p["machineFamily"])
	case compute.ResourceTypePublicImages:
		return compute.ListPublicImageResources(reqCtx, client, p["project"])
	case compute.ResourceTypeCustomImages:
		return compute.ListCustomImageResources(reqCtx, client, p["project"])
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
	case compute.ResourceTypeFirewall:
		return compute.ListFirewallResources(reqCtx, client, p["project"])
	default:
		return nil, nil
	}
}

func (g *GCP) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		g.handleEvent(ctx)
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
		Data      string `json:"data"`
		MessageID string `json:"messageId"`
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

func (g *GCP) handleEvent(ctx core.HTTPRequestContext) {
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
		if s.Name == PubSubSecretName {
			secret = string(s.Value)
			break
		}
	}

	if token != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
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
		ctx.Logger.Warnf("failed to decode Pub/Sub message data: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var entry logEntry
	if err := json.Unmarshal(decoded, &entry); err != nil {
		ctx.Logger.Warnf("failed to parse log entry: %v", err)
		ctx.Response.WriteHeader(http.StatusOK)
		return
	}

	var rawData map[string]any
	_ = json.Unmarshal(decoded, &rawData)

	event := AuditLogEvent{
		ServiceName:  entry.ProtoPayload.ServiceName,
		MethodName:   strings.TrimSpace(entry.ProtoPayload.MethodName),
		ResourceName: entry.ProtoPayload.ResourceName,
		LogName:      entry.LogName,
		Timestamp:    entry.Timestamp,
		InsertID:     entry.InsertID,
		Data:         rawData,
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Logger.Errorf("error listing subscriptions: %v", err)
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, subscription := range subscriptions {
		if !g.subscriptionApplies(subscription, event) {
			continue
		}

		if err := subscription.SendMessage(event); err != nil {
			ctx.Logger.Errorf("error sending message to subscription: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
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

func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
