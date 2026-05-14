package runner

import (
	"errors"
	"strings"
)

func normalizeCommands(commands string) []string {
	lines := strings.Split(commands, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func validateCommands(commands string) error {
	lines := normalizeCommands(commands)
	if len(lines) == 0 {
		return errors.New("at least one command is required")
	}
	return nil
}
