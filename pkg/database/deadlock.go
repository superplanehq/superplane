package database

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const deadlockSQLState = "40P01"

func isDeadlock(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == deadlockSQLState {
		return true
	}

	message := err.Error()
	return strings.Contains(message, "deadlock detected") || strings.Contains(message, deadlockSQLState)
}

func RetryOnDeadlock(attempts int, operation func() error, wait func(attempt int)) error {
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		err = operation()
		if err == nil || !isDeadlock(err) || attempt == attempts {
			return err
		}
		if wait != nil {
			wait(attempt)
		}
	}

	return err
}
