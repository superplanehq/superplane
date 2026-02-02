package eventbridge

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/aws/common"
)

type RuleMetadata struct {
	RuleArn  string `json:"ruleArn" mapstructure:"ruleArn"`
	TargetID string `json:"targetId" mapstructure:"targetId"`
}

func CreateRule(
	integration core.IntegrationContext,
	http core.HTTPContext,
	targetRoleArn string,
	region string,
	destinationArn string,
	tags []common.Tag,
	eventPattern map[string]any,
) (*RuleMetadata, error) {
	creds, err := common.CredentialsFromInstallation(integration)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	client := NewClient(http, creds, region)
	pattern, err := json.Marshal(eventPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event pattern: %w", err)
	}

	ruleName := fmt.Sprintf("superplane-%s", uuid.NewString())
	ruleArn, err := client.PutRule(ruleName, string(pattern), tags)
	if err != nil {
		return nil, fmt.Errorf("error creating EventBridge rule: %v", err)
	}

	if targetRoleArn == "" {
		return nil, fmt.Errorf("event bridge target role is missing")
	}

	targetID := uuid.NewString()
	err = client.PutTargets(ruleName, []Target{
		{
			ID:      targetID,
			Arn:     destinationArn,
			RoleArn: targetRoleArn,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error creating EventBridge target: %v", err)
	}

	return &RuleMetadata{
		RuleArn:  ruleArn,
		TargetID: targetID,
	}, nil
}
