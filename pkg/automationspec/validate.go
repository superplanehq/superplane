package automationspec

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/triggers/schedule"
)

func (d *Document) validateSemantics() error {
	ids := make(map[string]bool)
	webhooks := make(map[string]bool)
	for index, trigger := range d.Triggers {
		triggerPath := itemPath("triggers", index)
		if trigger.ID != "" {
			if ids[trigger.ID] {
				return d.Diagnostic(triggerPath+".id", "Give each trigger a unique ID.")
			}
			ids[trigger.ID] = true
		}
		if trigger.Use == Webhook {
			var id uuid.UUID
			var err error
			if trigger.Config != nil {
				id, err = uuid.Parse(trigger.Config.WebhookID)
			}
			if trigger.Config == nil || err != nil {
				return d.Diagnostic(triggerPath+".config.webhook_id", "Enter the ID of a webhook from the workspace settings.")
			}
			if webhooks[id.String()] {
				return d.Diagnostic(triggerPath+".config.webhook_id", "Use each webhook once.")
			}
			if trigger.Config.TaskMatch != nil {
				match := trigger.Config.TaskMatch
				if match.Kind != "task" && match.Kind != "pull_request" && match.Kind != "commit" {
					return d.Diagnostic(triggerPath+".config.task_match.kind", "Choose task, pull_request, or commit.")
				}
				if strings.TrimSpace(match.FieldPath) == "" {
					return d.Diagnostic(triggerPath+".config.task_match.field_path", "Enter the request body field path.")
				}
			}
			webhooks[id.String()] = true
			continue
		}
		if trigger.Config != nil {
			cron := strings.TrimSpace(trigger.Config.Cron)
			offset := strconv.FormatFloat(trigger.Config.UTCOffset, 'f', -1, 64)
			config := schedule.Configuration{Type: schedule.TypeCron, CronExpression: &cron, Timezone: &offset}
			if _, err := schedule.NextTrigger(config, time.Now()); err != nil {
				return d.Diagnostic(triggerPath+".config.cron", err.Error())
			}
		}
	}

	return nil
}
