package cloudwatch

import (
	"fmt"
	"strings"
)

// CloudWatch keeps every action for a state transition in one flat list of ARNs.
// The components expose SNS topics and the EC2 automation action as separate
// controls, so these helpers split that list back apart and reassemble it.

const alarmNamespaceEC2 = "AWS/EC2"

const dimensionInstanceID = "InstanceId"

// PartitionForRegion returns the ARN partition a region belongs to. China and
// GovCloud regions do not use the default "aws" partition.
func PartitionForRegion(region string) string {
	region = strings.TrimSpace(region)
	switch {
	case strings.HasPrefix(region, "cn-"):
		return "aws-cn"
	case strings.HasPrefix(region, "us-gov-"):
		return "aws-us-gov"
	default:
		return "aws"
	}
}

// EC2AutomationARN builds the ARN for an EC2 automation action.
func EC2AutomationARN(region, action string) string {
	region = strings.TrimSpace(region)
	return fmt.Sprintf("arn:%s:automate:%s:ec2:%s", PartitionForRegion(region), region, strings.TrimSpace(action))
}

// arnService reads the service segment of arn:<partition>:<service>:...,
// so classification holds in every partition.
func arnService(arn string) string {
	segments := strings.SplitN(strings.TrimSpace(arn), ":", 4)
	if len(segments) < 3 || segments[0] != "arn" {
		return ""
	}

	return segments[2]
}

func isEC2AutomationARN(arn string) bool {
	return arnService(arn) == "automate"
}

func isSNSTopicARN(arn string) bool {
	return arnService(arn) == "sns"
}

// mergeAlarmActions rebuilds the action list for a transition. Actions of a kind
// the user did not enable are kept as they are, and ARNs of any other kind
// (Lambda, Auto Scaling, SSM, ...) always survive.
func mergeAlarmActions(existing []string, snsTopics []string, ec2ActionARN string, replaceSNS, replaceEC2 bool) []string {
	var actions []string
	for _, arn := range existing {
		switch {
		case isSNSTopicARN(arn):
			if !replaceSNS {
				actions = append(actions, arn)
			}
		case isEC2AutomationARN(arn):
			if !replaceEC2 {
				actions = append(actions, arn)
			}
		default:
			actions = append(actions, arn)
		}
	}

	if replaceSNS {
		actions = append(actions, snsTopics...)
	}

	if replaceEC2 {
		if arn := strings.TrimSpace(ec2ActionARN); arn != "" {
			actions = append(actions, arn)
		}
	}

	return actions
}

// requireEC2ActionTarget checks the alarm can actually carry an EC2 automation
// action: CloudWatch resolves the target instance from the InstanceId dimension,
// so an alarm without one would accept the action and never run it.
func requireEC2ActionTarget(namespace string, dimensions []Dimension) error {
	if strings.TrimSpace(namespace) != alarmNamespaceEC2 {
		return fmt.Errorf("EC2 actions require an %s alarm, but the namespace is %q", alarmNamespaceEC2, namespace)
	}

	for _, dimension := range dimensions {
		if dimension.Name == dimensionInstanceID && strings.TrimSpace(dimension.Value) != "" {
			return nil
		}
	}

	return fmt.Errorf("EC2 actions require the alarm to have an %s dimension", dimensionInstanceID)
}
