package ecs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__formatTaskResourceName(t *testing.T) {
	t.Run("task definition and status -> friendly label", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn:           "arn:aws:ecs:us-east-1:123456789012:task/demo/ab12cd34ef56gh78ij90klmnop12qr34",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:7",
			LastStatus:        "RUNNING",
		})

		assert.Equal(t, "worker:7 (RUNNING) ab12cd34", name)
	})

	t.Run("missing task definition -> fallback to short task id", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn: "arn:aws:ecs:us-east-1:123456789012:task/demo/ab12cd34ef56gh78ij90klmnop12qr34",
		})

		assert.Equal(t, "ab12cd34", name)
	})

	t.Run("invalid arn -> fallback to task arn", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn: "not-an-arn",
		})

		assert.Equal(t, "not-an-arn", name)
	})
}
