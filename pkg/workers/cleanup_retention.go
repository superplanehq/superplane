package workers

import "time"

const deletedResourceGracePeriodDays = 30

func deletedResourceWithinGracePeriod(deletedAt time.Time, now time.Time) bool {
	cutoff := now.UTC().AddDate(0, 0, -deletedResourceGracePeriodDays)
	return deletedAt.UTC().After(cutoff)
}
