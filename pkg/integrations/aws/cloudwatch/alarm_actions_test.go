package cloudwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__EC2AutomationARN(t *testing.T) {
	assert.Equal(t, "arn:aws:automate:us-east-1:ec2:recover", EC2AutomationARN("us-east-1", "recover"))
	assert.Equal(t, "arn:aws-cn:automate:cn-north-1:ec2:reboot", EC2AutomationARN("cn-north-1", "reboot"))
	assert.Equal(t, "arn:aws-us-gov:automate:us-gov-west-1:ec2:stop", EC2AutomationARN("us-gov-west-1", "stop"))
}

func Test__AlarmActionClassification(t *testing.T) {
	t.Run("recognises every partition", func(t *testing.T) {
		for _, arn := range []string{
			"arn:aws:sns:us-east-1:123456789012:ops",
			"arn:aws-cn:sns:cn-north-1:123456789012:ops",
			"arn:aws-us-gov:sns:us-gov-west-1:123456789012:ops",
		} {
			assert.True(t, isSNSTopicARN(arn), arn)
			assert.False(t, isEC2AutomationARN(arn), arn)
		}

		for _, arn := range []string{
			"arn:aws:automate:us-east-1:ec2:recover",
			"arn:aws-cn:automate:cn-north-1:ec2:recover",
		} {
			assert.True(t, isEC2AutomationARN(arn), arn)
			assert.False(t, isSNSTopicARN(arn), arn)
		}
	})

	t.Run("other services are neither", func(t *testing.T) {
		for _, arn := range []string{
			"arn:aws:lambda:us-east-1:123456789012:function:remediate",
			"arn:aws:autoscaling:us-east-1:123456789012:scalingPolicy:abc",
			"",
			"not-an-arn",
		} {
			assert.False(t, isSNSTopicARN(arn), arn)
			assert.False(t, isEC2AutomationARN(arn), arn)
		}
	})
}

func Test__MergeAlarmActions__ChinaPartition(t *testing.T) {
	// Before partition-aware classification these topics counted as "other" and
	// survived alongside the replacements, silently accumulating.
	existing := []string{
		"arn:aws-cn:sns:cn-north-1:123456789012:old-topic",
		"arn:aws-cn:automate:cn-north-1:ec2:recover",
	}

	merged := mergeAlarmActions(
		existing,
		[]string{"arn:aws-cn:sns:cn-north-1:123456789012:new-topic"},
		"",
		true,
		false,
	)

	assert.Equal(t, []string{
		"arn:aws-cn:automate:cn-north-1:ec2:recover",
		"arn:aws-cn:sns:cn-north-1:123456789012:new-topic",
	}, merged)
}
