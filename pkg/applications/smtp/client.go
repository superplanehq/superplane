package smtp

import (
	"fmt"
	"strconv"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/wneessen/go-mail"
)

func NewClient(ctx core.AppInstallationContext) (*mail.Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("no app installation context")
	}

	host, err := ctx.GetConfig("host")
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP host: %w", err)
	}

	port, err := ctx.GetConfig("port")
	if err != nil {
		return nil, fmt.Errorf("failed to get SMTP port: %w", err)
	}

	authMethod, err := ctx.GetConfig("authMethod")
	if err != nil {
		return nil, fmt.Errorf("failed to get auth method: %w", err)
	}

	portInt, err := strconv.Atoi(string(port))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP port: %w", err)
	}

	// Handle password authentication
	if string(authMethod) == "password" {
		user, err := ctx.GetConfig("user")
		if err != nil {
			return nil, fmt.Errorf("failed to get SMTP user: %w", err)
		}

		password, err := ctx.GetConfig("password")
		if err != nil {
			return nil, fmt.Errorf("failed to get SMTP password: %w", err)
		}

		client, err := mail.NewClient(
			string(host),
			mail.WithPort(portInt),
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(string(user)),
			mail.WithPassword(string(password)),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}

		return client, nil
	}

	// Handle OAuth 2.0 authentication
	if string(authMethod) == "oauth2" {
		user, err := ctx.GetConfig("user")
		if err != nil {
			// For OAuth, user might not be set, try to extract from token
			user = []byte("")
		}

		accessToken, err := findSecret(ctx, SMTPAccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to get access token: %w", err)
		}

		client, err := mail.NewClient(
			string(host),
			mail.WithPort(portInt),
			mail.WithSMTPAuth(mail.SMTPAuthXOAUTH2),
			mail.WithUsername(string(user)),
			mail.WithPassword(accessToken),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create SMTP client: %w", err)
		}

		return client, nil
	}

	return nil, fmt.Errorf("unsupported auth method: %s", string(authMethod))
}

func findSecret(ctx core.AppInstallationContext, secretName string) (string, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Name == secretName {
			return string(secret.Value), nil
		}
	}

	return "", fmt.Errorf("secret %s not found", secretName)
}
