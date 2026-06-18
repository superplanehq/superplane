package config

import (
	"os"
	"strconv"
)

func intFromEnv(envVar string, defaultValue int) int {
	if v := os.Getenv(envVar); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}

	return defaultValue
}
