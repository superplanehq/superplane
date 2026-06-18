package core

import (
	"os"
	"strconv"
	"sync"
)

const (
	defaultMaxForEachItems = 100
	maxForEachItemsEnvVar  = "SUPERPLANE_FOREACH_MAX_ITEMS"

	// MaxEmitCount is the maximum number of events a single execution may emit at once.
	// Components that fan out (For Each, Read Memory "One By One", and similar) must stay
	// within this limit so we do not create unbounded downstream runs, DB rows, or queue load.
	MaxEmitCount = 500
)

var (
	maxForEachItemsOnce sync.Once
	maxForEachItems     int
)

// MaxForEachItems returns the maximum number of array items the For Each component may
// emit per execution. Defaults to 100. Override with SUPERPLANE_FOREACH_MAX_ITEMS.
func MaxForEachItems() int {
	maxForEachItemsOnce.Do(func() {
		if v := os.Getenv(maxForEachItemsEnvVar); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				maxForEachItems = min(n, MaxEmitCount)
				return
			}
		}
		maxForEachItems = defaultMaxForEachItems
	})
	return maxForEachItems
}

// ResetMaxForEachItemsForTests clears the cached For Each item limit.
func ResetMaxForEachItemsForTests() {
	maxForEachItemsOnce = sync.Once{}
	maxForEachItems = 0
}
