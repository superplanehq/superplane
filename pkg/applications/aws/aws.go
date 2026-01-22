package aws

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
)

const (
	defaultAudience            = "sts.amazonaws.com"
	defaultSessionDurationSecs = 3600
)

func init() {
	registry.RegisterApplication("aws", &AWS{})
}

type AWS struct{}

type Configuration struct {
	RoleArn                string `json:"roleArn" mapstructure:"roleArn"`
	Region                 string `json:"region" mapstructure:"region"`
	Audience               string `json:"audience" mapstructure:"audience"`
	SessionDurationSeconds int    `json:"sessionDurationSeconds" mapstructure:"sessionDurationSeconds"`
}

type SessionMetadata struct {
	RoleArn   string `json:"roleArn"`
	Region    string `json:"region"`
	Audience  string `json:"audience"`
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
	return "Generate short-lived AWS session tokens using SuperPlane OIDC"
}

func (a *AWS) InstallationInstructions() string {
	return ""
}

func (a *AWS) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "roleArn",
			Label:       "Role ARN",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Role ARN that SuperPlane should assume",
		},
		{
			Name:        "region",
			Label:       "STS Region or Endpoint",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "AWS region for STS (e.g. us-east-1). Use a full URL for nonstandard endpoints.",
		},
		{
			Name:        "audience",
			Label:       "OIDC Audience",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     defaultAudience,
			Description: "OIDC audience to include in the token (usually sts.amazonaws.com)",
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
	}
}

func (a *AWS) Components() []core.Component {
	return []core.Component{
		&RunLambda{},
	}
}

func (a *AWS) Triggers() []core.Trigger {
	return []core.Trigger{}
}

func (a *AWS) Sync(ctx core.SyncContext) error {
	if ctx.OIDCSigner == nil {
		return fmt.Errorf("OIDC signer is not configured")
	}

	config := Configuration{}
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	roleArn := strings.TrimSpace(config.RoleArn)
	if roleArn == "" {
		return fmt.Errorf("roleArn is required")
	}

	issuer := strings.TrimRight(strings.TrimSpace(ctx.BaseURL), "/")
	if issuer == "" {
		return fmt.Errorf("baseURL is required to generate OIDC tokens")
	}

	audience := strings.TrimSpace(config.Audience)
	if audience == "" {
		audience = defaultAudience
	}

	durationSeconds := config.SessionDurationSeconds
	if durationSeconds <= 0 {
		durationSeconds = defaultSessionDurationSecs
	}

	subject := fmt.Sprintf("app-installation:%s", ctx.InstallationID)
	if strings.TrimSpace(ctx.InstallationID) == "" {
		subject = fmt.Sprintf("app-installation:%s", ctx.AppInstallation.ID())
	}

	oidcToken, err := ctx.OIDCSigner.GenerateWithClaims(subject, 5*time.Minute, issuer, audience, nil)
	if err != nil {
		return fmt.Errorf("failed to generate OIDC token: %w", err)
	}

	sessionName := fmt.Sprintf("SuperPlane-%s", ctx.AppInstallation.ID())
	credentials, err := assumeRoleWithWebIdentity(ctx.HTTP, config.Region, roleArn, sessionName, oidcToken, durationSeconds)
	if err != nil {
		return err
	}

	if err := ctx.AppInstallation.SetSecret("accessKeyId", []byte(credentials.AccessKeyID)); err != nil {
		return err
	}
	if err := ctx.AppInstallation.SetSecret("secretAccessKey", []byte(credentials.SecretAccessKey)); err != nil {
		return err
	}
	if err := ctx.AppInstallation.SetSecret("sessionToken", []byte(credentials.SessionToken)); err != nil {
		return err
	}

	ctx.AppInstallation.SetMetadata(SessionMetadata{
		RoleArn:   roleArn,
		Region:    strings.TrimSpace(config.Region),
		Audience:  audience,
		ExpiresAt: credentials.Expiration.Format(time.RFC3339),
	})

	ctx.AppInstallation.SetState("ready", "")

	refreshAfter := time.Until(credentials.Expiration) / 2
	if refreshAfter < time.Minute {
		refreshAfter = time.Minute
	}

	return ctx.AppInstallation.ScheduleResync(refreshAfter)
}

func (a *AWS) HandleRequest(ctx core.HTTPRequestContext) {
	// no-op
}

func (a *AWS) CompareWebhookConfig(aConfig, bConfig any) (bool, error) {
	return true, nil
}

func (a *AWS) ListResources(resourceType string, ctx core.ListResourcesContext) ([]core.ApplicationResource, error) {
	return []core.ApplicationResource{}, nil
}

func (a *AWS) SetupWebhook(ctx core.SetupWebhookContext) (any, error) {
	return nil, nil
}

func (a *AWS) CleanupWebhook(ctx core.CleanupWebhookContext) error {
	return nil
}
