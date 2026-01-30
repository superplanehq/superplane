package aws

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
)

const (
	APIKeyHeaderName                = "X-Superplane-Secret"
	EventBridgeConnectionSecretName = "eventbridge.connection.secret"
)

func CreateAPIDestination(ctx core.SyncContext, tags []common.Tag) (*common.APIDestinationMetadata, error) {
	creds, err := common.CredentialsFromInstallation(ctx.Integration)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(common.RegionFromInstallation(ctx.Integration))
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	client := eventbridge.NewClient(ctx.HTTP, creds, region)

	secret, err := crypto.Base64String(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random string for connection secret: %w", err)
	}

	err = ctx.Integration.SetSecret(EventBridgeConnectionSecretName, []byte(secret))
	if err != nil {
		return nil, fmt.Errorf("failed to save connection secret: %w", err)
	}

	name := fmt.Sprintf("superplane-%s", ctx.Integration.ID().String())
	connectionArn, err := ensureConnection(client, name, []byte(secret), tags)
	if err != nil {
		return nil, err
	}

	apiDestinationArn, err := ensureApiDestination(
		client,
		fmt.Sprintf("superplane-%s", ctx.Integration.ID().String()),
		connectionArn,
		ctx.WebhooksBaseURL+"/api/v1/integrations/"+ctx.Integration.ID().String()+"/events",
		tags,
	)

	if err != nil {
		return nil, err
	}

	return &common.APIDestinationMetadata{
		ConnectionArn:     connectionArn,
		ApiDestinationArn: apiDestinationArn,
	}, nil
}

func ensureConnection(client *eventbridge.Client, name string, secret []byte, tags []common.Tag) (string, error) {
	connectionArn, err := client.CreateConnection(name, APIKeyHeaderName, string(secret), tags)
	if err == nil {
		if err := client.TagResource(connectionArn, tags); err != nil {
			return "", err
		}
		return connectionArn, nil
	}

	if !common.IsAlreadyExistsErr(err) {
		return "", err
	}

	connectionArn, err = client.DescribeConnection(name)
	if err != nil {
		return "", err
	}

	if err := client.TagResource(connectionArn, tags); err != nil {
		return "", err
	}

	return connectionArn, nil
}

func ensureApiDestination(client *eventbridge.Client, name, connectionArn, url string, tags []common.Tag) (string, error) {
	apiDestinationArn, err := client.CreateApiDestination(name, connectionArn, url, tags)
	if err == nil {
		if err := client.TagResource(apiDestinationArn, tags); err != nil {
			return "", err
		}
		return apiDestinationArn, nil
	}

	if !common.IsAlreadyExistsErr(err) {
		return "", err
	}

	apiDestinationArn, err = client.DescribeApiDestination(name)
	if err != nil {
		return "", err
	}

	if err := client.TagResource(apiDestinationArn, tags); err != nil {
		return "", err
	}

	return apiDestinationArn, nil
}
