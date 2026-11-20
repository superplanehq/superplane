package automationspec

import (
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/triggers/schedule"
)

func (d *Document) validateSemantics() error {
	ids := make(map[string]bool)
	for index, trigger := range d.Triggers {
		triggerPath := itemPath("triggers", index)
		if trigger.ID != "" {
			if ids[trigger.ID] {
				return d.Diagnostic(triggerPath+".id", "Give each trigger a unique ID.")
			}
			ids[trigger.ID] = true
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
