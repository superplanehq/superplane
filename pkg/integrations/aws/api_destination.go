package aws

import (
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
	"github.com/superplanehq/superplane/pkg/integrations/aws/eventbridge"
)

const (
	APIKeyHeaderName                = "X-Superplane-Secret"
	EventBridgeConnectionSecretName = "eventbridge.connection.secret"
)

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
