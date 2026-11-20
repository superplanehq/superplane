package automationspec

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"
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
			if err := validateCron(trigger.Config.Cron); err != nil {
				return d.Diagnostic(triggerPath+".config.cron", err.Error())
			}
		}
	}

	return nil
}

func validateCron(expression string) error {
	options := cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow
	switch len(strings.Fields(expression)) {
	case 5:
	case 6:
		options |= cron.Second
	default:
		return fmt.Errorf("enter a cron expression with five fields, or six with seconds first")
	}
	for _, char := range strings.TrimSpace(expression) {
		if !strings.ContainsRune("0123456789*,-/ abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", char) {
			return fmt.Errorf("remove %q from the cron expression", char)
		}
	}
	if _, err := cron.NewParser(options).Parse(expression); err != nil {
		return fmt.Errorf("correct the cron expression (%s)", err)
	}
	return nil
}
