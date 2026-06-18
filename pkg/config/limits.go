package config

import (
	"os"
	"strconv"
)

const (
	defaultMaxEmitCount   = 100
	maxEmitCountEnvVar    = "SUPERPLANE_MAX_EMIT_COUNT"
	defaultMaxPayloadSize = 64 * 1024
	maxPayloadSizeEnvVar  = "SUPERPLANE_MAX_PAYLOAD_SIZE"
)

// MaxEmitCount returns the maximum number of events a single execution may emit at once.
// Defaults to 100. Override with SUPERPLANE_MAX_EMIT_COUNT.
func MaxEmitCount() int {
	if v := os.Getenv(maxEmitCountEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultMaxEmitCount
}

// MaxPayloadSize returns the maximum serialized event payload size in bytes.
// Defaults to 64 KiB. Override with SUPERPLANE_MAX_PAYLOAD_SIZE.
func MaxPayloadSize() int {
	if v := os.Getenv(maxPayloadSizeEnvVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultMaxPayloadSize
}
