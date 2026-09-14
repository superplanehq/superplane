package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const testProcessIsolateEnv = "TEST_PROCESS_ISOLATE"

var processTestDatabaseNamePattern = regexp.MustCompile(`^superplane_[0-9]+_test$`)

var (
	isolatedName     string
	isolatedNameErr  error
	isolatedNameOnce sync.Once
)

func processTestDatabaseName(pid int) string {
	return "superplane_" + strconv.Itoa(pid) + "_test"
}

func isProcessTestDatabaseName(name string) bool {
	return processTestDatabaseNamePattern.MatchString(name)
}

func shouldIsolateTestDatabase(name string) bool {
	if os.Getenv(testProcessIsolateEnv) == "0" {
		return false
	}
	if !isTestDatabaseName(name) {
		return false
	}
	return !isProcessTestDatabaseName(name)
}

func isolateTestDatabaseName(c DSNConfig) (string, error) {
	if !shouldIsolateTestDatabase(c.Name) {
		return c.Name, nil
	}

	isolatedNameOnce.Do(func() {
		isolatedName, isolatedNameErr = cloneTestDatabase(c)
	})
	return isolatedName, isolatedNameErr
}

func cloneTestDatabase(c DSNConfig) (string, error) {
	clone := processTestDatabaseName(os.Getpid())
	if clone == c.Name {
		return clone, nil
	}

	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		lastErr = createDatabaseFromTemplate(c, clone)
		if lastErr == nil {
			log.Printf("[database] isolated test database %s from template %s", clone, c.Name)
			return clone, nil
		}
		if !isRetryableCloneError(lastErr) {
			return "", fmt.Errorf("clone test database %s from %s: %w", clone, c.Name, lastErr)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return "", fmt.Errorf("clone test database %s from %s: %w", clone, c.Name, lastErr)
}

func createDatabaseFromTemplate(c DSNConfig, clone string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, adminPostgresDSN(c))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	drop := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdent(clone))
	if _, err := conn.Exec(ctx, drop); err != nil {
		return err
	}

	create := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", quoteIdent(clone), quoteIdent(c.Name))
	_, err = conn.Exec(ctx, create)
	return err
}

func adminPostgresDSN(c DSNConfig) string {
	admin := c
	admin.Name = "postgres"
	admin.ApplicationName = "superplane-test-isolate"
	return buildPostgresDSN(admin, 30*time.Second, 30*time.Second)
}

func isRetryableCloneError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		// 55006 object_in_use: the template has another session, or a sibling
		// process is already cloning it.
		return pgErr.Code == "55006"
	}
	return strings.Contains(strings.ToLower(err.Error()), "being accessed by other users")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
