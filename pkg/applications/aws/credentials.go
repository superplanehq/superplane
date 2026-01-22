package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	accessKeyIDSecret     = "accessKeyId"
	secretAccessKeySecret = "secretAccessKey"
	sessionTokenSecret    = "sessionToken"
)

func getSessionCredentials(ctx core.AppInstallationContext) (aws.Credentials, error) {
	secrets, err := ctx.GetSecrets()
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to get AWS session secrets: %w", err)
	}

	var accessKeyID string
	var secretAccessKey string
	var sessionToken string

	for _, secret := range secrets {
		switch secret.Name {
		case accessKeyIDSecret:
			accessKeyID = string(secret.Value)
		case secretAccessKeySecret:
			secretAccessKey = string(secret.Value)
		case sessionTokenSecret:
			sessionToken = string(secret.Value)
		}
	}

	if strings.TrimSpace(accessKeyID) == "" || strings.TrimSpace(secretAccessKey) == "" || strings.TrimSpace(sessionToken) == "" {
		return aws.Credentials{}, fmt.Errorf("AWS session credentials are missing")
	}

	return aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
		Source:          "superplane",
	}, nil
}

func getRegionFromInstallation(ctx core.AppInstallationContext) string {
	regionBytes, err := ctx.GetConfig("region")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(regionBytes))
}
