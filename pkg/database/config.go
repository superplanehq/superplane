package database

import (
	"os"
)

const (
	DefaultDBStatementTimeoutMS                = "60000"
	DefaultDBIdleInTransactionSessionTimeoutMS = "30000"
)

type Config struct {
	Database Database
}

type Database struct {
	StatementTimeoutMS                string
	IdleInTransactionSessionTimeoutMS string
}

func Load() Config {
	return Config{
		Database: DatabaseFromEnv(),
	}
}

func DatabaseFromEnv() Database {
	return Database{
		StatementTimeoutMS: getenvOrDefault("DB_STATEMENT_TIMEOUT_MS", DefaultDBStatementTimeoutMS),
		IdleInTransactionSessionTimeoutMS: getenvOrDefault(
			"DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT_MS",
			DefaultDBIdleInTransactionSessionTimeoutMS,
		),
	}
}

func getenvOrDefault(key, defaultValue string) string {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	return v
}
