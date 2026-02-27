package loki

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type PushLogs struct{}

type PushLogsSpec struct {
	Labels  string `json:"labels"`
	Message string `json:"message"`
}

func (c *PushLogs) Name() string {
	return "loki.pushLogs"
}

func (c *PushLogs) Label() string {
	return "Push Logs"
}

func (c *PushLogs) Description() string {
	return "Push log entries to Loki"
}

func (c *PushLogs) Icon() string {
	return "file-text"
}

func (c *PushLogs) Color() string {
	return "gray"
}

func (c *PushLogs) Documentation() string {
	return `The Push Logs component sends log entries to a Grafana Loki instance.

## Use Cases

- **Workflow audit logging**: Record workflow execution events as structured logs in Loki
- **Deployment tracking**: Push deployment logs to Loki for centralized observability
- **Event forwarding**: Forward events from other systems to Loki for analysis

## Outputs

The component emits an event containing:
- ` + "`labels`" + `: The labels attached to the log stream
- ` + "`message`" + `: The log message that was pushed
- ` + "`timestamp`" + `: The Unix nanosecond timestamp of the log entry
`
}

func (c *PushLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *PushLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "labels",
			Label:       "Labels",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Comma-separated label pairs for the log stream (e.g., job=superplane,env=prod)",
			Placeholder: "job=superplane,env=prod",
		},
		{
			Name:        "message",
			Label:       "Log Message",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The log message to push to Loki",
		},
	}
}

func (c *PushLogs) Setup(ctx core.SetupContext) error {
	spec := PushLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	if spec.Labels == "" {
		return errors.New("labels is required")
	}

	if spec.Message == "" {
		return errors.New("message is required")
	}

	return nil
}

func (c *PushLogs) Execute(ctx core.ExecutionContext) error {
	spec := PushLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("error decoding configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	labels := parseLabels(spec.Labels)
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)

	req := PushLogRequest{
		Streams: []PushLogStream{
			{
				Stream: labels,
				Values: [][]string{
					{timestamp, spec.Message},
				},
			},
		},
	}

	err = client.PushLogs(req)
	if err != nil {
		return fmt.Errorf("failed to push logs: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"loki.logEntry",
		[]any{pushResultToMap(labels, spec.Message, timestamp)},
	)
}

func (c *PushLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PushLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PushLogs) Actions() []core.Action {
	return []core.Action{}
}

func (c *PushLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PushLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PushLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

func pushResultToMap(labels map[string]string, message, timestamp string) map[string]any {
	return map[string]any{
		"labels":    labels,
		"message":   message,
		"timestamp": timestamp,
	}
}

func parseLabels(input string) map[string]string {
	labels := make(map[string]string)
	for _, pair := range strings.Split(input, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				labels[key] = value
			}
		}
	}

	return labels
}
