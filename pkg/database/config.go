package database

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envStatementTimeout = "DB_STATEMENT_TIMEOUT"
	envIdleInTxTimeout  = "DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT"
)

var (
	DefaultStatementTimeout                = 60 * time.Second
	DefaultIdleInTransactionSessionTimeout = 30 * time.Second
)

// Config holds Postgres client pool session timeouts parsed from the environment.
type Config struct {
	StatementTimeout                time.Duration
	IdleInTransactionSessionTimeout time.Duration
}

// LoadConfig reads DB_STATEMENT_TIMEOUT and DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT
// using Go duration syntax (e.g. 60s, 500ms, 2m). A bare integer is interpreted as
// milliseconds for compatibility with older millisecond-only env values.
func LoadConfig() Config {
	return Config{
		StatementTimeout:                durationFromEnv(envStatementTimeout, DefaultStatementTimeout),
		IdleInTransactionSessionTimeout: durationFromEnv(envIdleInTxTimeout, DefaultIdleInTransactionSessionTimeout),
	}
}

func durationFromEnv(key string, defaultDur time.Duration) time.Duration {
	s := strings.TrimSpace(os.Getenv(key))
	if s == "" {
		return defaultDur
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil && ms >= 0 {
		return time.Duration(ms) * time.Millisecond
	}
	log.Printf("[database] invalid %s=%q, using default %v", key, s, defaultDur)
	return defaultDur
}
