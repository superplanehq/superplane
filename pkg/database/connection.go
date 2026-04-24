package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	postgres "gorm.io/driver/postgres"
	gorm "gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type Config struct {
	Host            string
	Port            string
	Name            string
	User            string
	Pass            string
	Ssl             string
	ApplicationName string
}

var dbInstance *gorm.DB

func Conn() *gorm.DB {
	if dbInstance == nil {
		dbInstance = connect()
	}

	return dbInstance.Session(&gorm.Session{})
}

func dbPoolSize() int {
	poolSize := os.Getenv("DB_POOL_SIZE")

	size, err := strconv.Atoi(poolSize)
	if err != nil {
		return 5
	}

	return size
}

func connect() *gorm.DB {
	postgresDbSSL := os.Getenv("POSTGRES_DB_SSL")
	sslMode := "disable"
	if postgresDbSSL == "true" {
		sslMode = "require"
	}

	c := Config{
		Host:            os.Getenv("DB_HOST"),
		Port:            os.Getenv("DB_PORT"),
		Name:            os.Getenv("DB_NAME"),
		Pass:            os.Getenv("DB_PASSWORD"),
		User:            os.Getenv("DB_USERNAME"),
		Ssl:             sslMode,
		ApplicationName: os.Getenv("APPLICATION_NAME"),
	}

	dsnTemplate := "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s application_name=%s"
	dsn := fmt.Sprintf(dsnTemplate, c.Host, c.Port, c.User, c.Pass, c.Name, c.Ssl, c.ApplicationName)

	logger := gormLogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormLogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormLogger.Warn,
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
	})

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger})
	if err != nil {
		panic(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}

	sqlDB.SetMaxOpenConns(dbPoolSize())
	sqlDB.SetMaxIdleConns(dbPoolSize())
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	configureTimeouts(db)

	return db
}

const (
	DefaultStatementTimeout               = "60s"
	DefaultIdleInTransactionTimeout       = "5min"
	DefaultTransactionTimeout             = 30 * time.Second
	DefaultWorkerTransactionTimeout       = 2 * time.Minute
	DefaultReadOnlyTransactionTimeout     = 15 * time.Second
	DefaultCanvasMutationTimeout          = 45 * time.Second
	DefaultEventProcessingTimeout         = 60 * time.Second
	DefaultCleanupTransactionTimeout      = 2 * time.Minute
	DefaultAuthTransactionTimeout         = 15 * time.Second
	DefaultOrganizationMutationTimeout    = 30 * time.Second
	DefaultIntegrationOperationTimeout    = 30 * time.Second
	DefaultAccountOperationTimeout        = 15 * time.Second
	DefaultServiceAccountOperationTimeout = 15 * time.Second
)

func configureTimeouts(db *gorm.DB) {
	statementTimeout := envOrDefault("DB_STATEMENT_TIMEOUT", DefaultStatementTimeout)
	idleInTxTimeout := envOrDefault("DB_IDLE_IN_TRANSACTION_TIMEOUT", DefaultIdleInTransactionTimeout)

	if err := db.Exec(fmt.Sprintf("SET statement_timeout = '%s'", statementTimeout)).Error; err != nil {
		log.Printf("[database] Warning: failed to set statement_timeout: %v", err)
	}

	if err := db.Exec(fmt.Sprintf("SET idle_in_transaction_session_timeout = '%s'", idleInTxTimeout)).Error; err != nil {
		log.Printf("[database] Warning: failed to set idle_in_transaction_session_timeout: %v", err)
	}
}

func envOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return defaultValue
}

func TransactionWithContext(ctx context.Context, timeout time.Duration, operationName string, fn func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	err := Conn().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})

	elapsed := time.Since(start)
	if err != nil && ctx.Err() != nil {
		log.Printf(
			"[database] Transaction timed out or cancelled: operation=%s elapsed=%s timeout=%s error=%v",
			operationName, elapsed, timeout, err,
		)
	}

	return err
}

func TruncateTables() error {
	return Conn().Exec(`
		truncate table
			secrets,
			account_magic_codes,
			account_password_auth,
			accounts,
			account_providers,
			users,
			organizations,
			organization_invitations,
			organization_invite_links,
			app_installations,
			app_installation_secrets,
			app_installation_requests,
			app_installation_subscriptions,
			casbin_rule,
			role_metadata,
			group_metadata,
			installation_metadata,
			blueprints,
			workflows,
			workflow_nodes,
			workflow_events,
			workflow_node_execution_kvs,
			workflow_node_executions,
			workflow_node_queue_items,
			workflow_node_requests,
			webhooks
		restart identity cascade;
	`).Error
}
