package aws

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/aws/cloudwatch"
	"github.com/superplanehq/superplane/pkg/integrations/aws/codeartifact"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ec2"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecr"
	"github.com/superplanehq/superplane/pkg/integrations/aws/ecs"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
	"github.com/superplanehq/superplane/pkg/integrations/aws/iam"
	"github.com/superplanehq/superplane/pkg/integrations/aws/lambda"
	"github.com/superplanehq/superplane/pkg/integrations/aws/route53"
	"github.com/superplanehq/superplane/pkg/integrations/aws/sns"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	defaultSessionDurationSecs      = 3600
	APIKeyHeaderName                = "X-Superplane-Secret"
	EventBridgeConnectionSecretName = "eventbridge.connection.secret"
)

func init() {
	registry.RegisterIntegrationWithWebhookHandler("aws", &AWS{}, &WebhookHandler{})
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
		&codeartifact.CopyPackageVersions{},
		&codeartifact.CreateRepository{},
		&codeartifact.DeletePackageVersions{},
		&codeartifact.DeleteRepository{},
		&codeartifact.DisposePackageVersions{},
		&codeartifact.GetPackageVersion{},
		&codeartifact.UpdatePackageVersionsStatus{},
		&ecs.DescribeService{},
		&ecs.RunTask{},
		&ecs.StopTask{},
		&ec2.CreateImage{},
		&ec2.GetImage{},
		&sns.GetTopic{},
		&sns.GetSubscription{},
		&sns.CreateTopic{},
		&sns.DeleteTopic{},
		&sns.PublishMessage{},
		&ecr.GetImage{},
		&ecr.GetImageScanFindings{},
		&ecr.ScanImage{},
		&lambda.RunFunction{},
		&route53.CreateRecord{},
		&route53.UpsertRecord{},
		&route53.DeleteRecord{},
	}
}

func (a *AWS) Triggers() []core.Trigger {
	return []core.Trigger{
		&cloudwatch.OnAlarm{},
		&codeartifact.OnPackageVersion{},
		&ec2.OnImage{},
		&ecr.OnImageScan{},
		&ecr.OnImagePush{},
		&sns.OnTopicMessage{},
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

	credentials, err := a.generateCredentials(ctx, config, accountID, &metadata)
	if err != nil {
		return fmt.Errorf("failed to generate credentials: %v", err)
	}

	err = a.configureRole(ctx, &metadata, credentials)
	if err != nil {
		return fmt.Errorf("failed to configure IAM role: %w", err)
	}

	err = a.configureEventBridge(ctx, config, &metadata, credentials)
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

	//
	// If we don't have session metadata, we don't need to cleanup anything.
	//
	if metadata.Session == nil {
		return nil
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	var errs error
	if metadata.EventBridge != nil {
		err := a.cleanupEventBridge(ctx, &metadata, credentials)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to cleanup event bridge: %w", err))
		}
	}

	if metadata.IAM != nil {
		err := a.cleanupIAM(ctx, &metadata, credentials)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to cleanup IAM: %w", err))
		}
	}

	return errs
}

func (a *AWS) cleanupEventBridge(ctx core.IntegrationCleanupContext, metadata *common.IntegrationMetadata, credentials *aws.Credentials) error {
	var err error

	//
	// Remove the EventBridge rules and targets.
	//
	for _, rule := range metadata.EventBridge.Rules {
		client := eventbridge.NewClient(ctx.HTTP, credentials, rule.Region)
		err := client.RemoveTargets(rule.Name, []string{"api-destination"})
		if err != nil && !common.IsNotFoundErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to remove targets for rule %s in region %s: %w", rule.Name, rule.Region, err))
		}

		err = client.DeleteRule(rule.Name)
		if err != nil && !common.IsNotFoundErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to delete rule %s in region %s: %w", rule.Name, rule.Region, err))
		}
	}

	//
	// Remove the EventBridge API destinations and connections.
	//
	for region, destination := range metadata.EventBridge.APIDestinations {
		client := eventbridge.NewClient(ctx.HTTP, credentials, region)

		err := client.DeleteAPIDestination(destination.Name)
		if err != nil && !common.IsNotFoundErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to delete API destination in region %s: %w", region, err))
		}

		err = client.DeleteConnection(destination.Name)
		if err != nil && !common.IsNotFoundErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to delete connection in region %s: %w", region, err))
		}
	}

	return err
}

func (a *AWS) cleanupIAM(ctx core.IntegrationCleanupContext, metadata *common.IntegrationMetadata, credentials *aws.Credentials) error {
	client := iam.NewClient(ctx.HTTP, credentials)

	var err error
	if metadata.IAM.TargetDestinationRole != nil {
		err := client.DeleteRolePolicy(metadata.IAM.TargetDestinationRole.RoleName, "invoke-api-destination")
		if err != nil && !iam.IsNoSuchEntityErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to delete IAM role policy for %s: %w", metadata.IAM.TargetDestinationRole.RoleName, err))
		}

		err = client.DeleteRole(metadata.IAM.TargetDestinationRole.RoleName)
		if err != nil && !iam.IsNoSuchEntityErr(err) {
			err = errors.Join(err, fmt.Errorf("failed to delete IAM role %s: %w", metadata.IAM.TargetDestinationRole.RoleName, err))
		}
	}

	return err
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
- Depending on the SuperPlane actions and triggers you will use, different permissions will be needed. Include the ones you need.
- Give it a name and description, and create it

**3. Complete the installation setup**

- Copy the ARN of the IAM role created in step 2
- Paste it into the "Role ARN" field in the installation configuration
`, ctx.BaseURL, ctx.Integration.ID().String()),
	})

	return nil
}

func (a *AWS) generateCredentials(ctx core.SyncContext, config Configuration, accountID string, metadata *common.IntegrationMetadata) (*aws.Credentials, error) {
	durationSeconds := config.SessionDurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultSessionDurationSecs
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.Integration.ID())
	oidcToken, err := ctx.OIDC.Sign(subject, 5*time.Minute, ctx.Integration.ID().String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OIDC token: %w", err)
	}

	sessionName := fmt.Sprintf("SuperPlane-%s", ctx.Integration.ID())
	stsCredentials, err := assumeRoleWithWebIdentity(ctx.HTTP, config.Region, config.RoleArn, sessionName, oidcToken, durationSeconds)
	if err != nil {
		return nil, fmt.Errorf("failed to assume role: %w", err)
	}

	if err := ctx.Integration.SetSecret("accessKeyId", []byte(stsCredentials.AccessKeyID)); err != nil {
		return nil, fmt.Errorf("failed to set access key ID secret: %w", err)
	}
	if err := ctx.Integration.SetSecret("secretAccessKey", []byte(stsCredentials.SecretAccessKey)); err != nil {
		return nil, fmt.Errorf("failed to set secret access key secret: %w", err)
	}
	if err := ctx.Integration.SetSecret("sessionToken", []byte(stsCredentials.SessionToken)); err != nil {
		return nil, fmt.Errorf("failed to set session token secret: %w", err)
	}

	refreshAfter := time.Until(stsCredentials.Expiration) / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	metadata.Session = &common.SessionMetadata{
		RoleArn:   config.RoleArn,
		AccountID: accountID,
		Region:    strings.TrimSpace(config.Region),
		ExpiresAt: stsCredentials.Expiration.Format(time.RFC3339),
	}

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	return credentials, ctx.Integration.ScheduleResync(refreshAfter)
}

func (a *AWS) configureEventBridge(ctx core.SyncContext, config Configuration, metadata *common.IntegrationMetadata, credentials *aws.Credentials) error {
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
	apiDestination, err := a.createAPIDestination(credentials, ctx.Integration, ctx.HTTP, ctx.WebhooksBaseURL, region, tags, secret)
	if err != nil {
		return fmt.Errorf("failed to create API destination: %w", err)
	}

	ctx.Logger.Infof("Created API destination %s for region %s", apiDestination.APIDestinationArn, region)

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
func (a *AWS) configureRole(ctx core.SyncContext, metadata *common.IntegrationMetadata, credentials *aws.Credentials) error {

	//
	// If the IAM metadata is already configured, do nothing.
	//
	if metadata.IAM != nil {
		return nil
	}

	//
	// Otherwise, create IAM role.
	//
	client := iam.NewClient(ctx.HTTP, credentials)
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
		TargetDestinationRole: &common.IAMRoleMetadata{
			RoleArn:  roleArn,
			RoleName: roleName,
		},
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

func (a *AWS) ruleName(integration core.IntegrationContext, source string) (string, error) {
	sourceParts := strings.Split(source, ".")
	if len(sourceParts) != 2 {
		return "", fmt.Errorf("invalid source: %s", source)
	}

	service := sourceParts[1]
	return fmt.Sprintf("superplane-%s-%s", integration.ID().String(), service), nil
}

func (a *AWS) createAPIDestination(
	credentials *aws.Credentials,
	integration core.IntegrationContext,
	http core.HTTPContext,
	baseURL string,
	region string,
	tags []common.Tag,
	secret string,
) (*common.APIDestinationMetadata, error) {
	client := eventbridge.NewClient(http, credentials, region)
	name := fmt.Sprintf("superplane-%s", integration.ID().String())
	connectionArn, err := a.ensureConnection(client, name, []byte(secret), tags)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	apiDestinationArn, err := a.ensureAPIDestination(
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
		Name:              name,
		Region:            region,
		ConnectionArn:     connectionArn,
		APIDestinationArn: apiDestinationArn,
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

func (a *AWS) ensureAPIDestination(client *eventbridge.Client, name, connectionArn, url string, tags []common.Tag) (string, error) {
	apiDestinationArn, err := client.CreateAPIDestination(name, connectionArn, url, tags)
	if err == nil {
		return apiDestinationArn, nil
	}

	if !common.IsAlreadyExistsErr(err) {
		return "", err
	}

	apiDestinationArn, err = client.DescribeAPIDestination(name)
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

	if configuration.Region != event.Region {
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

func (a *AWS) Actions() []core.Action {
	return []core.Action{
		{
			Name:        "provisionRule",
			Description: "Provision an EventBridge rule",
			Parameters: []configuration.Field{
				{
					Name:        "region",
					Label:       "Region",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The region to provision the API destination in",
				},
				{
					Name:        "source",
					Label:       "Source",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The source to provision the rule for",
				},
				{
					Name:        "detailType",
					Label:       "Detail Type",
					Type:        configuration.FieldTypeString,
					Required:    true,
					Description: "The detail type to provision the rule for",
				},
			},
		},
	}
}

func (a *AWS) HandleAction(ctx core.IntegrationActionContext) error {
	switch ctx.Name {
	case "provisionRule":
		return a.handleProvisionRule(ctx)

	default:
		return fmt.Errorf("unknown action: %s", ctx.Name)
	}
}

func (a *AWS) handleProvisionRule(ctx core.IntegrationActionContext) error {
	config := common.ProvisionRuleParameters{}
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

	credentials, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	//
	// If destination already exists, do nothing.
	//
	destination, err := a.provisionDestination(credentials, ctx.Logger, ctx.Integration, ctx.HTTP, ctx.WebhooksBaseURL, &metadata, config.Region)
	if err != nil {
		return fmt.Errorf("failed to provision destination: %w", err)
	}

	err = a.provisionRule(credentials, ctx.Logger, ctx.Integration, ctx.HTTP, &metadata, destination, config.Source, config.DetailType)
	if err != nil {
		return fmt.Errorf("failed to provision rule: %w", err)
	}

	ctx.Integration.SetMetadata(metadata)
	return nil
}

func (a *AWS) provisionDestination(credentials *aws.Credentials, logger *logrus.Entry, ctx core.IntegrationContext, http core.HTTPContext, webhooksBaseURL string, metadata *common.IntegrationMetadata, region string) (*common.APIDestinationMetadata, error) {
	v, ok := metadata.EventBridge.APIDestinations[region]
	if ok {
		return &v, nil
	}

	secrets, err := ctx.GetSecrets()
	if err != nil {
		return nil, fmt.Errorf("failed to get integration secrets: %w", err)
	}

	var secret string
	for _, s := range secrets {
		if s.Name == EventBridgeConnectionSecretName {
			secret = string(s.Value)
			break
		}
	}

	if secret == "" {
		return nil, fmt.Errorf("connection secret not found")
	}

	//
	// Create API destination
	//
	newDestination, err := a.createAPIDestination(credentials, ctx, http, webhooksBaseURL, region, metadata.Tags, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create API destination: %w", err)
	}

	logger.Infof("Created API destination %s for region %s", newDestination.APIDestinationArn, region)
	metadata.EventBridge.APIDestinations[region] = *newDestination
	return newDestination, nil
}

func (a *AWS) provisionRule(credentials *aws.Credentials, logger *logrus.Entry, integration core.IntegrationContext, http core.HTTPContext, metadata *common.IntegrationMetadata, destination *common.APIDestinationMetadata, source string, detailType string) error {
	ruleKey := common.EventBridgeRuleKey(source, destination.Region)

	//
	// If the rule does not exist yet, we create it.
	//
	rule, ok := metadata.EventBridge.Rules[ruleKey]
	if !ok {
		return a.createRule(credentials, logger, integration, http, metadata, destination, source, []string{detailType})
	}

	//
	// If rule already exists, and already has the detail type we are interested in, do nothing.
	//
	if slices.Contains(rule.DetailTypes, detailType) {
		return nil
	}

	//
	// Otherwise, update the detail types for the rule.
	//
	newDetailTypes := append(rule.DetailTypes, detailType)
	return a.updateRule(credentials, logger, http, metadata, &rule, newDetailTypes)
}

func (a *AWS) updateRule(credentials *aws.Credentials, logger *logrus.Entry, http core.HTTPContext, metadata *common.IntegrationMetadata, rule *common.EventBridgeRuleMetadata, detailTypes []string) error {
	client := eventbridge.NewClient(http, credentials, rule.Region)
	pattern, err := json.Marshal(map[string]any{
		"source":      []string{rule.Source},
		"detail-type": detailTypes,
	})

	if err != nil {
		return fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	_, err = client.PutRule(rule.Name, string(pattern), metadata.Tags)
	if err != nil {
		return fmt.Errorf("error updating EventBridge rule %s: %v", rule.RuleArn, err)
	}

	ruleKey := common.EventBridgeRuleKey(rule.Source, rule.Region)
	metadata.EventBridge.Rules[ruleKey] = common.EventBridgeRuleMetadata{
		Name:        rule.Name,
		Source:      rule.Source,
		Region:      rule.Region,
		RuleArn:     rule.RuleArn,
		DetailTypes: detailTypes,
	}

	logger.Infof("Updated EventBridge rule %s: %v", rule.RuleArn, detailTypes)

	return nil
}

func (a *AWS) createRule(
	credentials *aws.Credentials,
	logger *logrus.Entry,
	integration core.IntegrationContext,
	http core.HTTPContext,
	metadata *common.IntegrationMetadata,
	destination *common.APIDestinationMetadata,
	source string,
	detailTypes []string,
) error {
	client := eventbridge.NewClient(http, credentials, destination.Region)
	pattern, err := json.Marshal(map[string]any{
		"source":      []string{source},
		"detail-type": detailTypes,
	})

	if err != nil {
		return fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	ruleName, err := a.ruleName(integration, source)
	if err != nil {
		return fmt.Errorf("failed to get rule name: %w", err)
	}

	ruleArn, err := client.PutRule(ruleName, string(pattern), metadata.Tags)
	if err != nil {
		return fmt.Errorf("error creating EventBridge rule for %s: %v", source, err)
	}

	err = client.PutTargets(ruleName, []eventbridge.Target{
		{
			ID:      "api-destination",
			Arn:     destination.APIDestinationArn,
			RoleArn: metadata.IAM.TargetDestinationRole.RoleArn,
		},
	})

	if err != nil {
		return fmt.Errorf("error creating EventBridge target: %v", err)
	}

	if metadata.EventBridge.Rules == nil {
		metadata.EventBridge.Rules = make(map[string]common.EventBridgeRuleMetadata)
	}

	ruleKey := common.EventBridgeRuleKey(source, destination.Region)
	metadata.EventBridge.Rules[ruleKey] = common.EventBridgeRuleMetadata{
		Name:        ruleName,
		Source:      source,
		Region:      destination.Region,
		RuleArn:     ruleArn,
		DetailTypes: detailTypes,
	}

	logger.Infof("Created EventBridge rule %s: %v", ruleArn, detailTypes)

	return nil
}
