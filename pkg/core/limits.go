package core

import (
	"os"
	"strconv"
)

const (
	defaultMaxEmitCount = 100
	maxEmitCountEnvVar  = "SUPERPLANE_MAX_EMIT_COUNT"
)

// MaxEmitCount returns the maximum number of events a single execution may emit at once.
// Components that fan out (For Each, Read Memory "One By One", and similar) must stay
// within this limit so we do not create unbounded downstream runs, DB rows, or queue load.
// Defaults to 100. Override with SUPERPLANE_MAX_EMIT_COUNT.
func MaxEmitCount() int {
	if v := os.Getenv(maxEmitCountEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultMaxEmitCount
}
