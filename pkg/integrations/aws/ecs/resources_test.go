package ecs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__formatTaskResourceName(t *testing.T) {
	t.Run("task definition and status -> friendly label", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn:           "arn:aws:ecs:us-east-1:123456789012:task/demo/ab12cd34ef56gh78ij90klmnop12qr34",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:7",
			LastStatus:        "RUNNING",
		})

		assert.Equal(t, "worker:7 (RUNNING) ab12cd34ef56gh78ij90klmnop12qr34", name)
	})

	t.Run("missing task definition -> fallback to task id", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn: "arn:aws:ecs:us-east-1:123456789012:task/demo/ab12cd34ef56gh78ij90klmnop12qr34",
		})

		assert.Equal(t, "ab12cd34ef56gh78ij90klmnop12qr34", name)
	})

	t.Run("invalid arn -> fallback to task arn", func(t *testing.T) {
		name := formatTaskResourceName(Task{
			TaskArn: "not-an-arn",
		})

		assert.Equal(t, "not-an-arn", name)
	})
}

func Test__batchTaskARNs(t *testing.T) {
	t.Run("splits task arns into 100-sized batches", func(t *testing.T) {
		taskArns := make([]string, 0, 250)
		for i := range 250 {
			taskArns = append(taskArns, fmt.Sprintf("arn:aws:ecs:us-east-1:123:task/demo/%03d", i))
		}

		batches := batchTaskARNs(taskArns, 100)
		require.Len(t, batches, 3)
		assert.Len(t, batches[0], 100)
		assert.Len(t, batches[1], 100)
		assert.Len(t, batches[2], 50)
	})
}

func Test__listTaskResourceNamesByDescribe(t *testing.T) {
	t.Run("calls describe in batches of 100 and maps friendly names", func(t *testing.T) {
		taskArns := make([]string, 0, 250)
		for i := range 250 {
			taskArns = append(taskArns, fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/demo/task-%03d", i))
		}

		callCount := 0
		names := listTaskResourceNamesByDescribe("demo", taskArns, func(cluster string, tasks []string) (*DescribeTasksResponse, error) {
			require.Equal(t, "demo", cluster)
			require.LessOrEqual(t, len(tasks), 100)
			callCount++

			responseTasks := make([]Task, 0, len(tasks))
			for _, taskArn := range tasks {
				responseTasks = append(responseTasks, Task{
					TaskArn:           taskArn,
					TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
					LastStatus:        "RUNNING",
				})
			}

			return &DescribeTasksResponse{
				Tasks: responseTasks,
			}, nil
		})

		assert.Equal(t, 3, callCount)
		assert.Len(t, names, 250)
		assert.Contains(t, names[taskArns[0]], "worker:1")
		assert.Contains(t, names[taskArns[0]], "RUNNING")
	})

	t.Run("describe failure in one batch still preserves successful batches", func(t *testing.T) {
		taskArns := make([]string, 0, 201)
		for i := range 201 {
			taskArns = append(taskArns, fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/demo/task-%03d", i))
		}

		callCount := 0

		names := listTaskResourceNamesByDescribe("demo", taskArns, func(cluster string, tasks []string) (*DescribeTasksResponse, error) {
			callCount++
			if len(tasks) > 0 && tasks[0] == taskArns[100] {
				return nil, fmt.Errorf("upstream error")
			}

			responseTasks := make([]Task, 0, len(tasks))
			for _, taskArn := range tasks {
				responseTasks = append(responseTasks, Task{
					TaskArn:           taskArn,
					TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
					LastStatus:        "RUNNING",
				})
			}

			return &DescribeTasksResponse{
				Tasks: responseTasks,
			}, nil
		})

		assert.Equal(t, 3, callCount)
		assert.Len(t, names, 101)
		_, hasFirstBatch := names[taskArns[0]]
		_, hasFailedBatch := names[taskArns[100]]
		_, hasLastBatch := names[taskArns[200]]
		assert.True(t, hasFirstBatch)
		assert.False(t, hasFailedBatch)
		assert.True(t, hasLastBatch)
	})
}
