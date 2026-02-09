package sns

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

// validateTopic verifies the topic exists and returns normalized topic metadata.
func validateTopic(httpCtx core.HTTPContext, integration core.IntegrationContext, region string, topicArn string) (*Topic, error) {
	credentials, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		return nil, fmt.Errorf("validate SNS topic: failed to load AWS credentials from integration: %w", err)
	}

	client := NewClient(httpCtx, credentials, region)
	topic, err := client.GetTopic(topicArn)
	if err != nil {
		return nil, fmt.Errorf("validate SNS topic: failed to get topic %q in region %q: %w", topicArn, region, err)
	}

	return topic, nil
}
