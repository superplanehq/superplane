package core

import (
	"os"
	"strconv"
)

const (
	defaultMaxForEachItems = 100
	maxForEachItemsEnvVar  = "SUPERPLANE_FOREACH_MAX_ITEMS"

	// MaxEmitCount is the maximum number of events a single execution may emit at once.
	// Components that fan out (For Each, Read Memory "One By One", and similar) must stay
	// within this limit so we do not create unbounded downstream runs, DB rows, or queue load.
	MaxEmitCount = 500
)

// MaxForEachItems returns the maximum number of array items the For Each component may
// emit per execution. Defaults to 100. Override with SUPERPLANE_FOREACH_MAX_ITEMS.
func MaxForEachItems() int {
	if v := os.Getenv(maxForEachItemsEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return min(n, MaxEmitCount)
		}
	}

	return defaultMaxForEachItems
}
