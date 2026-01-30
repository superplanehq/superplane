package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
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
	RoleArn                string `json:"roleArn" mapstructure:"roleArn"`
	Region                 string `json:"region" mapstructure:"region"`
	SessionDurationSeconds int    `json:"sessionDurationSeconds" mapstructure:"sessionDurationSeconds"`
}

type SessionMetadata struct {
	RoleArn   string `json:"roleArn"`
	Region    string `json:"region"`
	ExpiresAt string `json:"expiresAt"`
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
	}
}

func (a *AWS) Components() []core.Component {
	return []core.Component{
		&lambda.RunFunction{},
	}
}

func (a *AWS) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (a *AWS) Sync(ctx core.SyncContext) error {
	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if config.RoleArn == "" {
		return a.showBrowserAction(ctx)
	}

	return a.generateCredentials(ctx, config)
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

func (a *AWS) generateCredentials(ctx core.SyncContext, config Configuration) error {
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

	ctx.Integration.SetMetadata(SessionMetadata{
		RoleArn:   config.RoleArn,
		Region:    strings.TrimSpace(config.Region),
		ExpiresAt: credentials.Expiration.Format(time.RFC3339),
	})

	ctx.Integration.Ready()
	ctx.Integration.RemoveBrowserAction()

	refreshAfter := time.Until(credentials.Expiration) / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	return ctx.Integration.ScheduleResync(refreshAfter)
}

func (a *AWS) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (a *AWS) CompareWebhookConfig(aConfig, bConfig any) (bool, error) {
	return true, nil
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

	default:
		return []core.IntegrationResource{}, nil
	}
}

func (a *AWS) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (a *AWS) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
