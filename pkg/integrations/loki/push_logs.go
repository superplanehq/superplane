package loki

import (
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

type PushLogsNodeMetadata struct {
	Labels string `json:"labels"`
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

func (c *PushLogs) Documentation() string {
	return `The Push Logs component sends log entries to a Loki instance via the push API (` + "`POST /loki/api/v1/push`" + `).

## Configuration

- **Labels**: Required comma-separated key=value pairs used as Loki stream labels (e.g. ` + "`job=superplane,env=prod`" + `)
- **Message**: Required log message content to push

## Output

Emits one ` + "`loki.pushLogs`" + ` payload confirming the pushed labels and message.`
}

func (c *PushLogs) Icon() string {
	return "loki"
}

func (c *PushLogs) Color() string {
	return "gray"
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
			Placeholder: "job=superplane,env=prod",
			Description: "Comma-separated key=value label pairs for the log stream",
		},
		{
			Name:        "message",
			Label:       "Message",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "Log message to push",
		},
	}
}

func (c *PushLogs) Setup(ctx core.SetupContext) error {
	spec := PushLogsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec = sanitizePushLogsSpec(spec)

	if spec.Labels == "" {
		return fmt.Errorf("labels is required")
	}

	if spec.Message == "" {
		return fmt.Errorf("message is required")
	}

	if _, err := parseLabels(spec.Labels); err != nil {
		return fmt.Errorf("invalid labels: %w", err)
	}

	return nil
}

func (c *PushLogs) Execute(ctx core.ExecutionContext) error {
	spec := PushLogsSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec = sanitizePushLogsSpec(spec)

	labels, err := parseLabels(spec.Labels)
	if err != nil {
		return fmt.Errorf("invalid labels: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create Loki client: %w", err)
	}

	ts := strconv.FormatInt(time.Now().UnixNano(), 10)

	streams := []Stream{
		{
			Stream: labels,
			Values: [][]string{{ts, spec.Message}},
		},
	}

	if err := client.Push(streams); err != nil {
		return fmt.Errorf("failed to push logs: %w", err)
	}

	ctx.Metadata.Set(PushLogsNodeMetadata{Labels: spec.Labels})

	payload := map[string]any{
		"labels":  labels,
		"message": spec.Message,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"loki.pushLogs",
		[]any{payload},
	)
}

func (c *PushLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *PushLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *PushLogs) Actions() []core.Action {
	return []core.Action{}
}

func (c *PushLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *PushLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *PushLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *PushLogs) ExampleOutput() map[string]any {
	return exampleOutputPushLogs()
}

func parseLabels(labelsStr string) (map[string]string, error) {
	labels := make(map[string]string)

	for _, pair := range strings.Split(labelsStr, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format %q, expected key=value", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("label key cannot be empty")
		}

		labels[key] = value
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("at least one label is required")
	}

	return labels, nil
}

func sanitizePushLogsSpec(spec PushLogsSpec) PushLogsSpec {
	spec.Labels = strings.TrimSpace(spec.Labels)
	spec.Message = strings.TrimSpace(spec.Message)
	return spec
}
