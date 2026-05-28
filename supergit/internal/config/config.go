package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	DefaultRoot           = "/var/lib/supergit/repos"
	DefaultPort           = "8080"
	DefaultBranch         = "main"
	DefaultMaxFileBytes   = 10 * 1024 * 1024
	DefaultMaxCommitBytes = 25 * 1024 * 1024
)

type Config struct {
	Root           string
	Port           string
	DefaultBranch  string
	MaxFileBytes   int64
	MaxCommitBytes int64
}

func Load() Config {
	return Config{
		Root:           loadString("SUPERGIT_ROOT", DefaultRoot),
		Port:           loadString("SUPERGIT_PORT", DefaultPort),
		DefaultBranch:  loadString("SUPERGIT_DEFAULT_BRANCH", DefaultBranch),
		MaxFileBytes:   loadInt64("SUPERGIT_MAX_FILE_BYTES", DefaultMaxFileBytes),
		MaxCommitBytes: loadInt64("SUPERGIT_MAX_COMMIT_BYTES", DefaultMaxCommitBytes),
	}
}

func loadString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func loadInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
