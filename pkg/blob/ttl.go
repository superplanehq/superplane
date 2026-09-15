package blob

import "time"

const (
	UIDownloadTTL          = time.Hour
	minDispatchTimeoutSecs = 3600
	dispatchTTLPadding     = 30 * time.Minute
	signedURLExpiryBucket  = 5 * time.Minute
)

// StableExpiry rounds the signed-URL end time up to a bucket so remints
// in the same window return the same URL. That keeps <img> src stable
// while a chat poll refreshes the work order.
func StableExpiry(ttl time.Duration) time.Time {
	if ttl <= 0 {
		ttl = time.Hour
	}
	bucket := signedURLExpiryBucket
	if ttl < 2*bucket {
		bucket = time.Minute
	}
	target := time.Now().Add(ttl).UTC()
	rounded := target.Truncate(bucket)
	if rounded.Before(target) {
		rounded = rounded.Add(bucket)
	}
	return rounded
}

func DispatchDownloadTTL(executionTimeoutSeconds int) time.Duration {
	timeout := executionTimeoutSeconds
	if timeout < minDispatchTimeoutSecs {
		timeout = minDispatchTimeoutSecs
	}
	return time.Duration(timeout)*time.Second + dispatchTTLPadding
}
