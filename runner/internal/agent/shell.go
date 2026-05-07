package agent

import (
	"errors"
	"strings"
)

func normalizeDirectiveLines(directives []string) []string {
	var out []string
	for _, s := range directives {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func errEmptyCommands() error {
	return errors.New("empty commands")
}
