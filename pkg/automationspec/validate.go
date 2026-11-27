package automationspec

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/triggers/schedule"
)

func (d *Document) validateSemantics() error {
	if d.Timeout != "" {
		seconds, err := ParseDuration(d.Timeout)
		if err != nil {
			return d.Diagnostic("timeout", err.Error())
		}
		d.TimeoutSeconds = seconds
		d.Timeout = FormatDuration(seconds)
	}
	if d.Agent != nil && d.Agent.Timeout != "" {
		seconds, err := ParseDuration(d.Agent.Timeout)
		if err != nil {
			return d.Diagnostic("agent.timeout", err.Error())
		}
		if d.Timeout != "" && seconds != d.TimeoutSeconds {
			return d.Diagnostic("agent.timeout", "Use one timeout value for this automation.")
		}
		d.TimeoutSeconds = seconds
		d.Agent.Timeout = FormatDuration(seconds)
	}
	taskEvents, hasSchedule, hasWebhook, hasManual := false, false, false, false
	for _, trigger := range d.Triggers {
		switch trigger.Uses {
		case TaskEntersStage, TaskLeavesStage, PullRequestUpdated:
			taskEvents = true
		case Scheduled:
			hasSchedule = true
		case Webhook:
			hasWebhook = true
		case Manual:
			hasManual = true
		}
	}
	if taskEvents && hasSchedule {
		return d.Diagnostic("triggers", "Task events cannot share an automation with schedule triggers.")
	}
	if d.RunContext == "task" && (hasSchedule || hasWebhook) {
		return d.Diagnostic("run_context", "Schedule and webhook triggers require workspace context.")
	}
	if d.RunContext == "workspace" && taskEvents {
		return d.Diagnostic("run_context", "Task event triggers require task context.")
	}
	if d.RunContext != "" {
		onTask := d.RunContext == "task"
		d.RunsOnTask = &onTask
	} else if hasSchedule || (hasWebhook && !taskEvents && !hasManual) {
		onTask := false
		d.RunsOnTask = &onTask
	}
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
		if trigger.Uses == Webhook {
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
			webhooks[id.String()] = true
			continue
		}
		if trigger.Config != nil {
			cron := strings.TrimSpace(trigger.Config.Cron)
			if trigger.Config.Timezone == "" {
				trigger.Config.Timezone = "0"
			}
			offset := trigger.Config.Timezone
			hours, err := strconv.ParseFloat(offset, 64)
			if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) || hours < -12 || hours > 14 {
				return d.Diagnostic(triggerPath+".config.timezone", "Use a UTC offset from -12 through 14, such as \"5.5\".")
			}
			config := schedule.Configuration{Type: schedule.TypeCron, CronExpression: &cron, Timezone: &offset}
			if _, err := schedule.NextTrigger(config, time.Now()); err != nil {
				return d.Diagnostic(triggerPath+".config.cron", err.Error())
			}
		}
	}

	return nil
}

func (d *Document) rejectReplacedFields() error {
	replacements := map[string]string{
		"timeout_seconds": "timeout", "execution_timeout_seconds": "timeout", "runs_on_task": "run_context",
		"skip_setup_commands": "agent.skip_setup_commands", "skip_repository_checkout": "agent.skip_repository_checkout",
	}
	for index := 0; index < len(d.root.Content)-1; index += 2 {
		key := d.root.Content[index]
		if replacement := replacements[key.Value]; replacement != "" {
			return at(key.Value, key, "Use "+replacement+" instead of "+key.Value+".")
		}
	}
	if agent := mappingProperty(d.root, "agent").value; agent != nil {
		for old, replacement := range map[string]string{"instructions": "additional_instructions", "timeout_seconds": "timeout", "execution_timeout_seconds": "timeout"} {
			if field := mappingProperty(agent, old); field.key != nil {
				return at("agent."+old, field.key, "Use "+replacement+" instead of "+old+".")
			}
		}
	}
	triggers := mappingProperty(d.root, "triggers").value
	if triggers == nil {
		return nil
	}
	for index, trigger := range triggers.Content {
		prefix := itemPath("triggers", index)
		if field := mappingProperty(trigger, "use"); field.key != nil {
			return at(prefix+".use", field.key, "Use uses instead of use.")
		}
		if config := mappingProperty(trigger, "config").value; config != nil {
			if field := mappingProperty(config, "utc_offset"); field.key != nil {
				return at(prefix+".config.utc_offset", field.key, "Use timezone instead of utc_offset.")
			}
		}
	}
	return nil
}
