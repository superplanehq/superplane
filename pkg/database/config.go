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

type Config struct {
	StatementTimeout                time.Duration
	IdleInTransactionSessionTimeout time.Duration
}

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
