package blob

import "time"

const (
	UIDownloadTTL          = time.Hour
	minDispatchTimeoutSecs = 3600
	dispatchTTLPadding     = 30 * time.Minute
)

func DispatchDownloadTTL(executionTimeoutSeconds int) time.Duration {
	timeout := executionTimeoutSeconds
	if timeout < minDispatchTimeoutSecs {
		timeout = minDispatchTimeoutSecs
	}
	return time.Duration(timeout)*time.Second + dispatchTTLPadding
}
