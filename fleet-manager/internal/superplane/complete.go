package superplane

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// IsBridgeWebhook reports whether the task was created from SuperPlane sync.
func IsBridgeWebhook(url string) bool {
	return strings.TrimSpace(url) == BridgeWebhookURL
}

// CompleteTask reports a terminal local task to SuperPlane.
func (c *Client) CompleteTask(ctx context.Context, task *models.Task, taskLog *api.TaskLogSink) error {
	if c == nil || task == nil || !IsBridgeWebhook(task.WebhookURL) {
		return nil
	}

	exit := 0
	if task.ExitCode != nil {
		exit = *task.ExitCode
	}
	req := CompleteRequest{
		ExitCode: exit,
		Output:   task.Output,
		Error:    task.ErrorMessage,
		Canceled: task.Status == models.StatusCanceled,
	}
	if strings.TrimSpace(task.ResultJSON) != "" {
		req.Result = json.RawMessage(task.ResultJSON)
	}
	if taskLog != nil {
		req.TaskLog = apiTaskLogToSink(taskLog)
	}
	return c.Complete(ctx, task.ID, req)
}

func apiTaskLogToSink(t *api.TaskLogSink) *TaskLogSink {
	if t == nil || strings.TrimSpace(t.Type) == "" {
		return nil
	}
	out := &TaskLogSink{Type: t.Type}
	if t.CloudWatch != nil {
		out.CloudWatch = &TaskLogSinkCloudWatch{
			LogGroupName:  t.CloudWatch.LogGroupName,
			LogStreamName: t.CloudWatch.LogStreamName,
			Region:        t.CloudWatch.Region,
		}
	}
	return out
}
