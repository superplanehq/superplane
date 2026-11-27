package automationspec

import (
	"fmt"
	"time"
)

// ParseDuration accepts whole seconds and the runner's zero/default value.
func ParseDuration(value string) (int, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 || duration > 24*time.Hour || duration%time.Second != 0 {
		return 0, fmt.Errorf("use a duration from 0s through 24h in whole seconds, such as 2h or 30s")
	}
	return int(duration / time.Second), nil
}

func FormatDuration(seconds int) string {
	if seconds%3600 == 0 && seconds != 0 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	if seconds%60 == 0 && seconds != 0 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	return fmt.Sprintf("%ds", seconds)
}
