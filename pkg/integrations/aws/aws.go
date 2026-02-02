package aws

import (
	"encoding/json"
	"errors"
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
	"github.com/superplanehq/superplane/pkg/integrations/aws/iam"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	defaultSessionDurationSecs      = 3600
	APIKeyHeaderName                = "X-Superplane-Secret"
	EventBridgeConnectionSecretName = "eventbridge.connection.secret"
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

	metadata.Tags = common.NormalizeTags(config.Tags)
	accountID, err := common.AccountIDFromRoleArn(config.RoleArn)
	if err != nil {
		return fmt.Errorf("failed to get account ID from role ARN: %v", err)
	}

	err = a.generateCredentials(ctx, config, accountID, &metadata)
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %v", err)
	}

	err = a.configureRole(ctx, &metadata)
	if err != nil {
		return fmt.Errorf("failed to configure IAM role: %w", err)
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
	metadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if metadata.EventBridge == nil && metadata.IAM == nil {
		return nil
	}

	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	var cleanupErr error
	destinationName := fmt.Sprintf("superplane-%s", ctx.Integration.ID().String())

	if metadata.EventBridge != nil {
		for region := range metadata.EventBridge.APIDestinations {
			client := eventbridge.NewClient(ctx.HTTP, creds, region)

			err := client.DeleteApiDestination(destinationName)
			if err != nil && !common.IsNotFoundErr(err) {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("failed to delete API destination in region %s: %w", region, err))
			}

			err = client.DeleteConnection(destinationName)
			if err != nil && !common.IsNotFoundErr(err) {
				cleanupErr = errors.Join(cleanupErr, fmt.Errorf("failed to delete connection in region %s: %w", region, err))
			}
		}
	}

	if metadata.IAM != nil {
		client := iam.NewClient(ctx.HTTP, creds)
		roleName := a.roleName(ctx.Integration)

		if parsedRoleName, ok := roleNameFromArn(metadata.IAM.TargetDestinationRoleArn); ok {
			roleName = parsedRoleName
		}

		err := client.DeleteRolePolicy(roleName, "invoke-api-destination")
		if err != nil && !iam.IsNoSuchEntityErr(err) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("failed to delete IAM role policy for %s: %w", roleName, err))
		}

		err = client.DeleteRole(roleName)
		if err != nil && !iam.IsNoSuchEntityErr(err) {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("failed to delete IAM role %s: %w", roleName, err))
		}
	}

	return cleanupErr
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
- Add permissions for the integration to manage EventBridge connections, API destinations, and rules. To get started, you can use the **AmazonEventBridgeFullAccess** managed policy
- Add permissions for the integration manage IAM roles needed for itself. To get started, you can use the **IAMFullAccess** managed policy
- Depending on the SuperPlane actions and triggers you will use, different permissions will be needed. Include the ones you need
- Give it a name and description, and create it

**3. Complete the installation setup**

- Copy the ARN of the IAM role created in step 2
- Paste it into the "Role ARN" field in the installation configuration
`, ctx.BaseURL, ctx.Integration.ID().String()),
	})

	return nil
}

func (a *AWS) generateCredentials(ctx core.SyncContext, config Configuration, accountID string, metadata *common.IntegrationMetadata) error {
	durationSeconds := config.SessionDurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultSessionDurationSecs
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.Integration.ID())
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
		AccountID: accountID,
		Region:    strings.TrimSpace(config.Region),
		ExpiresAt: credentials.Expiration.Format(time.RFC3339),
	}

	return ctx.Integration.ScheduleResync(refreshAfter)
}

func (a *AWS) configureEventBridge(ctx core.SyncContext, config Configuration, metadata *common.IntegrationMetadata) error {
	//
	// If event bridge metadata is already configured, do nothing.
	//
	if metadata.EventBridge != nil {
		return nil
	}

	//
	// If the region is not set, do nothing.
	//
	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return nil
	}

	tags := common.NormalizeTags(config.Tags)
	secret, err := a.destinationSecret(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get connection secret: %w", err)
	}

	//
	// Create API destination
	//
	apiDestination, err := a.createAPIDestination(ctx.Integration, ctx.HTTP, ctx.WebhooksBaseURL, region, tags, secret)
	if err != nil {
		return fmt.Errorf("failed to create API destination: %w", err)
	}

	ctx.Logger.Infof("Created API destination %s for region %s", apiDestination.ApiDestinationArn, region)

	metadata.EventBridge = &common.EventBridgeMetadata{
		APIDestinations: map[string]common.APIDestinationMetadata{
			region: *apiDestination,
		},
	}

	return nil
}

/*
 * In order to create and point EventBridge rules to the API destinations,
 * we need a specific IAM role which has the necessary permissions to do so.
 * This role will be used by the SuperPlane triggers created to listen to AWS events.
 */
func (a *AWS) configureRole(ctx core.SyncContext, metadata *common.IntegrationMetadata) error {

	//
	// If the IAM metadata is already configured, do nothing.
	//
	if metadata.IAM != nil {
		return nil
	}

	//
	// Otherwise, create IAM role.
	//
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := iam.NewClient(ctx.HTTP, creds)
	roleName := a.roleName(ctx.Integration)
	roleArn := ""

	trustPolicy, err := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Effect": "Allow",
				"Principal": map[string]any{
					"Service": "events.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal event bridge trust policy: %w", err)
	}

	roleArn, err = client.CreateRole(roleName, string(trustPolicy), metadata.Tags)
	if err != nil {
		if !iam.IsEntityAlreadyExistsErr(err) {
			return fmt.Errorf("failed to create event bridge role: %w", err)
		}

		roleArn, err = client.GetRole(roleName)
		if err != nil {
			return fmt.Errorf("failed to fetch event bridge role: %w", err)
		}
	}

	//
	// Attach policy to the role to allow it to invoke the API destinations.
	//
	policyDocument, err := json.Marshal(map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Effect":   "Allow",
				"Action":   "events:InvokeApiDestination",
				"Resource": fmt.Sprintf("arn:aws:events:*:%s:api-destination/*", metadata.Session.AccountID),
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to marshal event bridge role policy: %w", err)
	}

	err = client.PutRolePolicy(roleName, "invoke-api-destination", string(policyDocument))
	if err != nil {
		return fmt.Errorf("failed to attach event bridge policy: %w", err)
	}

	ctx.Logger.Infof("Created IAM role %s", roleArn)

	metadata.IAM = &common.IAMMetadata{
		TargetDestinationRoleArn: roleArn,
	}

	return nil
}

/*
 * AWS IAM role names must be at most 64 characters long,
 * so we only use the last part of the integration ID.
 */
func (a *AWS) roleName(integration core.IntegrationContext) string {
	idParts := strings.Split(integration.ID().String(), "-")
	return fmt.Sprintf("superplane-destination-invoker-%s", idParts[len(idParts)-1])
}

func roleNameFromArn(arn string) (string, bool) {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "", false
	}

	index := strings.LastIndex(arn, "role/")
	if index == -1 {
		return "", false
	}

	name := strings.TrimSpace(arn[index+len("role/"):])
	if name == "" {
		return "", false
	}

	if lastSlash := strings.LastIndex(name, "/"); lastSlash != -1 {
		name = strings.TrimSpace(name[lastSlash+1:])
		if name == "" {
			return "", false
		}
	}

	return name, true
}

func (a *AWS) createAPIDestination(
	integration core.IntegrationContext,
	http core.HTTPContext,
	baseURL string,
	region string,
	tags []common.Tag,
	secret string,
) (*common.APIDestinationMetadata, error) {
	creds, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := eventbridge.NewClient(http, creds, region)
	name := fmt.Sprintf("superplane-%s", integration.ID().String())
	connectionArn, err := a.ensureConnection(client, name, []byte(secret), tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	apiDestinationArn, err := a.ensureApiDestination(
		client,
		fmt.Sprintf("superplane-%s", integration.ID().String()),
		connectionArn,
		baseURL+"/api/v1/integrations/"+integration.ID().String()+"/events",
		tags,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create API destination: %w", err)
	}

	return &common.APIDestinationMetadata{
		ConnectionArn:     connectionArn,
		ApiDestinationArn: apiDestinationArn,
	}, nil
}

func (a *AWS) ensureConnection(client *eventbridge.Client, name string, secret []byte, tags []common.Tag) (string, error) {
	connectionArn, err := client.CreateConnection(name, APIKeyHeaderName, string(secret), tags)
	if err == nil {
		return connectionArn, nil
	}

	if !common.IsAlreadyExistsErr(err) {
		return "", err
	}

	connectionArn, err = client.DescribeConnection(name)
	if err != nil {
		return "", err
	}

	return connectionArn, nil
}

func (a *AWS) ensureApiDestination(client *eventbridge.Client, name, connectionArn, url string, tags []common.Tag) (string, error) {
	apiDestinationArn, err := client.CreateApiDestination(name, connectionArn, url, tags)
	if err == nil {
		return apiDestinationArn, nil
	}

	if !common.IsAlreadyExistsErr(err) {
		return "", err
	}

	apiDestinationArn, err = client.DescribeApiDestination(name)
	if err != nil {
		return "", err
	}

	return apiDestinationArn, nil
}

func (a *AWS) destinationSecret(integration core.IntegrationContext) (string, error) {
	secrets, err := integration.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == EventBridgeConnectionSecretName {
			return string(secret.Value), nil
		}
	}

	secret, err := crypto.Base64String(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate random string for connection secret: %w", err)
	}

	err = integration.SetSecret(EventBridgeConnectionSecretName, []byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to save connection secret: %w", err)
	}

	return secret, nil
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
		return lambda.ListFunctions(ctx, resourceType)

	case "ecr.repository":
		ctx.Logger.Infof("listing ECR repositories")
		return ecr.ListRepositories(ctx, resourceType)

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (a *AWS) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "provisionDestination",
			Description: "Provision an API destination for EventBridge",
			Parameters: []configuration.Field{
				{
					Name:        "region",
					Label:       "Region",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The region to provision the API destination in",
				},
			},
		},
	}
}

func (a *AWS) HandleAction(ctx core.IntegrationActionContext) error {
	switch ctx.Name {
	case "provisionDestination":
		return a.provisionDestination(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (a *AWS) provisionDestination(ctx core.IntegrationActionContext) error {
	config := common.ProvisionDestinationParameters{}
	if err := mapstructure.Decode(ctx.Parameters, &config); err != nil {
		return fmt.Errorf("failed to decode parameters: %v", err)
	}

	if config.Region == "" {
		return fmt.Errorf("region is required")
	}

	metadata := common.IntegrationMetadata{}
	if err := mapstructure.Decode(ctx.Integration.GetMetadata(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	//
	// If destination already exists, do nothing.
	//
	_, ok := metadata.EventBridge.APIDestinations[config.Region]
	if ok {
		return nil
	}

	secrets, err := ctx.Integration.GetSecrets()
	if err != nil {
		return fmt.Errorf("failed to get integration secrets: %w", err)
	}

	var secret string
	for _, s := range secrets {
		if s.Name == EventBridgeConnectionSecretName {
			secret = string(s.Value)
			break
		}
	}

	if secret == "" {
		return fmt.Errorf("connection secret not found")
	}

	//
	// Create API destination
	//
	apiDestination, err := a.createAPIDestination(ctx.Integration, ctx.HTTP, ctx.WebhooksBaseURL, config.Region, []common.Tag{}, secret)
	if err != nil {
		return fmt.Errorf("failed to create API destination: %w", err)
	}

	ctx.Logger.Infof("Created API destination %s for region %s", apiDestination.ApiDestinationArn, config.Region)

	metadata.EventBridge.APIDestinations[config.Region] = *apiDestination
	ctx.Integration.SetMetadata(metadata)
	return nil
}

/*
 * No additional webhook endpoints are used for AWS triggers.
 * Events from AWS are received through the API destinations configured
 * in the integration itself, using the integration HTTP URL.
 */
func (a *AWS) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (a *AWS) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
