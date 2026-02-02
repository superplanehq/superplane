package aws

import (
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
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecr"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	defaultSessionDurationSecs = 3600
)

func init() {
	registry.RegisterIntegration("aws", &AWS{})
}

type AWS struct{}

type Configuration struct {
	RoleArn                string       `json:"roleArn" mapstructure:"roleArn"`
	Region                 string       `json:"region" mapstructure:"region"`
	SessionDurationSeconds int          `json:"sessionDurationSeconds" mapstructure:"sessionDurationSeconds"`
	Tags                   []common.Tag `json:"tags" mapstructure:"tags"`
}

func (a *AWS) Name() string {
	return "aws"
}

func (a *AWS) Label() string {
	return "AWS"
}

func (a *AWS) Icon() string {
	return "aws"
}

func (a *AWS) Description() string {
	return "Manage resources and execute AWS commands in workflows"
}

func (a *AWS) Instructions() string {
	return "Initially, you can leave the **\"IAM Role ARN\"** field empty, as you will be guided through the identity provider and IAM role creation process."
}

func (a *AWS) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "region",
			Label:       "STS Region or Endpoint",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     "us-east-1",
			Description: "AWS region for STS",
		},
		{
			Name:        "sessionDurationSeconds",
			Label:       "Session Duration (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     fmt.Sprintf("%d", defaultSessionDurationSecs),
			Description: "Requested duration for the AWS session (up to the role max session duration)",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: func() *int { min := 900; return &min }(),
					Max: func() *int { max := 43200; return &max }(),
				},
			},
		},
		{
			Name:        "roleArn",
			Label:       "IAM Role ARN",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "ARN for the IAM role that SuperPlane should assume. Leave empty to be guided through the identity provider and IAM role creation process.",
		},
		{
			Name:        "tags",
			Label:       "Tags",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Tags to apply to AWS resources created by this integration",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Tag",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:               "key",
								Label:              "Key",
								Type:               configuration.FieldTypeString,
								Required:           true,
								DisallowExpression: true,
							},
							{
								Name:     "value",
								Label:    "Value",
								Type:     configuration.FieldTypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func (a *AWS) Components() []core.Component {
	return []core.Component{
		&lambda.RunFunction{},
	}
}

func (a *AWS) Triggers() []core.Trigger {
	return []core.Trigger{
		&ecr.OnImageScan{},
		&ecr.OnImagePush{},
	}
}

func (a *AWS) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	metadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if config.RoleArn == "" {
		return a.showBrowserAction(ctx)
	}

	err := a.generateCredentials(ctx, config, &metadata)
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %v", err)
	}

	err = a.configureEventBridge(ctx, config, &metadata)
	if err != nil {
		return fmt.Errorf("failed to configure event bridge: %v", err)
	}

	ctx.Integration.SetMetadata(metadata)
	ctx.Integration.Ready()
	ctx.Integration.RemoveBrowserAction()

	return nil
}

func (a *AWS) Cleanup(ctx core.IntegrationCleanupContext) error {
	return nil
}

func (a *AWS) showBrowserAction(ctx core.SyncContext) error {
	ctx.Integration.NewBrowserAction(core.BrowserAction{
		Description: fmt.Sprintf(`
**1. Create Identity Provider**

- Go to AWS IAM Console → Identity Providers → Add provider
- Choose "OpenID Connect" as the provider type
- Provider URL: **%s**
- Audience: **%s**

**2. Create IAM Role**

- Go to AWS IAM Console → Roles → Create role
- Choose "Web identity" as trusted entity type
- Select the identity provider created in step 1
- Add the permissions to the role
- Give it a name and description, and create it

**3. Complete the installation setup**

- Copy the ARN of the IAM role created in step 2
- Paste it into the "Role ARN" field in the installation configuration
`, ctx.BaseURL, ctx.Integration.ID().String()),
	})

	return nil
}

func (a *AWS) generateCredentials(ctx core.SyncContext, config Configuration, metadata *common.IntegrationMetadata) error {
	durationSeconds := config.SessionDurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultSessionDurationSecs
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.InstallationID)
	if strings.TrimSpace(ctx.InstallationID) == "" {
		subject = fmt.Sprintf("app-installation:%s", ctx.Integration.ID())
	}

	oidcToken, err := ctx.OIDC.Sign(subject, 5*time.Minute, ctx.Integration.ID().String(), nil)
	if err != nil {
		return fmt.Errorf("failed to generate OIDC token: %w", err)
	}

	sessionName := fmt.Sprintf("SuperPlane-%s", ctx.Integration.ID())
	credentials, err := assumeRoleWithWebIdentity(ctx.HTTP, config.Region, config.RoleArn, sessionName, oidcToken, durationSeconds)
	if err != nil {
		return err
	}

	if err := ctx.Integration.SetSecret("accessKeyId", []byte(credentials.AccessKeyID)); err != nil {
		return err
	}
	if err := ctx.Integration.SetSecret("secretAccessKey", []byte(credentials.SecretAccessKey)); err != nil {
		return err
	}
	if err := ctx.Integration.SetSecret("sessionToken", []byte(credentials.SessionToken)); err != nil {
		return err
	}

	refreshAfter := time.Until(credentials.Expiration) / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	metadata.Session = &common.SessionMetadata{
		RoleArn:   config.RoleArn,
		Region:    strings.TrimSpace(config.Region),
		ExpiresAt: credentials.Expiration.Format(time.RFC3339),
	}

	return ctx.Integration.ScheduleResync(refreshAfter)
}

func (a *AWS) configureEventBridge(ctx core.SyncContext, config Configuration, metadata *common.IntegrationMetadata) error {
	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return fmt.Errorf("region is required")
	}

	tags := common.NormalizeTags(config.Tags)
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return err
	}

	client := eventbridge.NewClient(ctx.HTTP, creds, region)
	secret, err := crypto.Base64String(32)
	if err != nil {
		return fmt.Errorf("failed to generate random string for connection secret: %w", err)
	}

	err = ctx.Integration.SetSecret(EventBridgeConnectionSecretName, []byte(secret))
	if err != nil {
		return fmt.Errorf("failed to save connection secret: %w", err)
	}

	name := fmt.Sprintf("superplane-%s", ctx.Integration.ID().String())
	connectionArn, err := ensureConnection(client, name, []byte(secret), tags)
	if err != nil {
		return err
	}

	apiDestinationArn, err := ensureApiDestination(
		client,
		fmt.Sprintf("superplane-%s", ctx.Integration.ID().String()),
		connectionArn,
		ctx.WebhooksBaseURL+"/api/v1/integrations/"+ctx.Integration.ID().String()+"/events",
		tags,
	)

	if err != nil {
		return err
	}

	metadata.EventBridge = &common.EventBridgeMetadata{
		APIDestinations: map[string]common.APIDestinationMetadata{
			region: {
				ConnectionArn:     connectionArn,
				ApiDestinationArn: apiDestinationArn,
			},
		},
	}

	return nil
}

func (a *AWS) HandleRequest(ctx core.HTTPRequestContext) {
	if strings.HasSuffix(ctx.Request.URL.Path, "/events") {
		a.handleEvent(ctx)
		return
	}

	ctx.Logger.Warnf("unknown path: %s", ctx.Request.URL.Path)
	ctx.Response.WriteHeader(http.StatusNotFound)
}

func (a *AWS) handleEvent(ctx core.HTTPRequestContext) {
	apiKey := ctx.Request.Header.Get(APIKeyHeaderName)
	if apiKey == "" {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		ctx.Response.Write([]byte("missing " + APIKeyHeaderName + " header"))
		return
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("error finding integration secrets: " + err.Error()))
		return
	}

	var secret string
	for _, s := range secrets {
		if s.Name == EventBridgeConnectionSecretName {
			secret = string(s.Value)
			break
		}
	}

	if apiKey != secret {
		ctx.Response.WriteHeader(http.StatusForbidden)
		ctx.Response.Write([]byte("invalid " + APIKeyHeaderName + " header"))
		return
	}

	subscriptions, err := ctx.Integration.ListSubscriptions()
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("error listing integration subscriptions: " + err.Error()))
		return
	}

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Response.WriteHeader(http.StatusInternalServerError)
		ctx.Response.Write([]byte("error reading request body: " + err.Error()))
		return
	}

	data := map[string]any{}
	if err := json.Unmarshal(body, &data); err != nil {
		ctx.Response.WriteHeader(http.StatusBadRequest)
		ctx.Response.Write([]byte("error parsing request body: " + err.Error()))
		return
	}

	for _, subscription := range subscriptions {
		if !a.subscriptionApplies(subscription, data) {
			continue
		}

		err = subscription.SendMessage(data)
		if err != nil {
			ctx.Logger.Errorf("error sending message from app: %v", err)
		}
	}

	ctx.Response.WriteHeader(http.StatusOK)
}

func (a *AWS) subscriptionApplies(subscription core.IntegrationSubscriptionContext, data map[string]any) bool {
	var event common.EventBridgeEvent
	err := mapstructure.Decode(data, &event)
	if err != nil {
		return false
	}

	var configuration common.EventBridgeEvent
	err = mapstructure.Decode(subscription.Configuration(), &configuration)
	if err != nil {
		return false
	}

	if configuration.DetailType != event.DetailType {
		return false
	}

	if configuration.Source != event.Source {
		return false
	}

	if len(configuration.Detail) > 0 {
		for key, value := range configuration.Detail {
			if event.Detail[key] != value {
				return false
			}
		}
	}

	return true
}

func (a *AWS) CompareWebhookConfig(aConfig, bConfig any) (bool, error) {
	return false, nil
}

func (a *AWS) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.IntegrationResource, error) {
	switch resourceType {
	case "lambda.function":
		creds, err := common.CredentialsFromInstallation(ctx.Integration)
		if err != nil {
			return nil, err
		}

		region := common.RegionFromInstallation(ctx.Integration)
		if strings.TrimSpace(region) == "" {
			return nil, fmt.Errorf("region is required")
		}

		client := lambda.NewClient(ctx.HTTP, creds, region)
		functions, err := client.ListFunctions()
		if err != nil {
			return nil, fmt.Errorf("failed to list lambda functions: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(functions))
		for _, function := range functions {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: function.FunctionName,
				ID:   function.FunctionArn,
			})
		}

		return resources, nil

	case "ecr.repository":
		creds, err := common.CredentialsFromInstallation(ctx.Integration)
		if err != nil {
			return nil, err
		}

		region := common.RegionFromInstallation(ctx.Integration)
		if strings.TrimSpace(region) == "" {
			return nil, fmt.Errorf("region is required")
		}

		client := ecr.NewClient(ctx.HTTP, creds, region)
		repositories, err := client.ListRepositories()
		if err != nil {
			return nil, fmt.Errorf("failed to list ECR repositories: %w", err)
		}

		resources := make([]core.IntegrationResource, 0, len(repositories))
		for _, repository := range repositories {
			resources = append(resources, core.IntegrationResource{
				Type: resourceType,
				Name: repository.RepositoryName,
				ID:   repository.RepositoryArn,
			})
		}

		return resources, nil

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (a *AWS) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	// no-op
	return nil, nil
}

func (a *AWS) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	// no-op
	return nil
}

// func CleanupWebhook(ctx core.CleanupWebhookContext) error {
// 	metadata := WebhookMetadata{}
// 	if err := mapstructure.Decode(ctx.Webhook.GetMetadata(), &metadata); err != nil {
// 		return fmt.Errorf("error decoding webhook metadata: %w", err)
// 	}

// 	creds, err := common.CredentialsFromInstallation(ctx.Integration)
// 	if err != nil {
// 		return err
// 	}

// 	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
// 	if region == "" {
// 		return fmt.Errorf("region is required")
// 	}

// 	client := eventbridge.NewClient(ctx.HTTP, creds, region)

// 	if metadata.RuleName != "" && metadata.TargetID != "" {
// 		err := client.RemoveTargets(metadata.RuleName, []string{metadata.TargetID})
// 		if err != nil && !common.IsNotFoundErr(err) {
// 			return err
// 		}
// 	}

// 	if metadata.RuleName != "" {
// 		err := client.DeleteRule(metadata.RuleName)
// 		if err != nil && !common.IsNotFoundErr(err) {
// 			return err
// 		}
// 	}

// 	if metadata.ApiDestinationName != "" {
// 		err := client.DeleteApiDestination(metadata.ApiDestinationName)
// 		if err != nil && !common.IsNotFoundErr(err) {
// 			return err
// 		}
// 	}

// 	if metadata.ConnectionName != "" {
// 		err := client.DeleteConnection(metadata.ConnectionName)
// 		if err != nil && !common.IsNotFoundErr(err) {
// 			return err
// 		}
// 	}

// 	return nil
// }

func (a *AWS) Actions() []core.Action {
	return []core.Action{}
}

func (a *AWS) HandleAction(ctx core.IntegrationActionContext) error {
	return nil
}
