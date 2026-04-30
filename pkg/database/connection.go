package database

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	postgres "gorm.io/driver/postgres"
	gorm "gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type DSNConfig struct {
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

func buildPostgresDSN(c DSNConfig, statementTimeout, idleInTxTimeout time.Duration) string {
	stmtMs := strconv.FormatInt(statementTimeout.Milliseconds(), 10)
	idleMs := strconv.FormatInt(idleInTxTimeout.Milliseconds(), 10)
	u := url.URL{
		Scheme: "postgres",
		Host:   net.JoinHostPort(c.Host, c.Port),
		Path:   "/" + c.Name,
	}
	u.User = url.UserPassword(c.User, c.Pass)

	q := url.Values{}
	q.Set("sslmode", c.Ssl)
	if c.ApplicationName != "" {
		q.Set("application_name", c.ApplicationName)
	}

	options := fmt.Sprintf(
		"-c statement_timeout=%s -c idle_in_transaction_session_timeout=%s",
		stmtMs,
		idleMs,
	)
	q.Set("options", options)
	u.RawQuery = q.Encode()
	return u.String()
}

func connect() *gorm.DB {
	postgresDbSSL := os.Getenv("POSTGRES_DB_SSL")
	sslMode := "disable"
	if postgresDbSSL == "true" {
		sslMode = "require"
	}

	c := DSNConfig{
		Host:            os.Getenv("DB_HOST"),
		Port:            os.Getenv("DB_PORT"),
		Name:            os.Getenv("DB_NAME"),
		Pass:            os.Getenv("DB_PASSWORD"),
		User:            os.Getenv("DB_USERNAME"),
		Ssl:             sslMode,
		ApplicationName: os.Getenv("APPLICATION_NAME"),
	}

	cfg := LoadConfig()
	dsn := buildPostgresDSN(c, cfg.StatementTimeout, cfg.IdleInTransactionSessionTimeout)

	baseLogger := gormLogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormLogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormLogger.Warn,
		Colorful:                  true,
		IgnoreRecordNotFoundError: true,
	})
	logger := newGormTimeoutLogger(baseLogger)

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

	log.Printf(
		"[database] enforced timeouts: max_open=%d DB_STATEMENT_TIMEOUT=%s DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT=%s host=%s dbname=%s",
		dbPoolSize(),
		cfg.StatementTimeout,
		cfg.IdleInTransactionSessionTimeout,
		c.Host,
		c.Name,
	)

	return db
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
