package database

import (
	"database/sql"
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

func dsnConfigFromEnv() DSNConfig {
	postgresDbSSL := os.Getenv("POSTGRES_DB_SSL")
	sslMode := "disable"
	if postgresDbSSL == "true" {
		sslMode = "require"
	}

	return DSNConfig{
		Host:            os.Getenv("DB_HOST"),
		Port:            os.Getenv("DB_PORT"),
		Name:            os.Getenv("DB_NAME"),
		Pass:            os.Getenv("DB_PASSWORD"),
		User:            os.Getenv("DB_USERNAME"),
		Ssl:             sslMode,
		ApplicationName: os.Getenv("APPLICATION_NAME"),
	}
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

func OpenDedicatedSQLDB(applicationName string, maxOpenConns int) (*sql.DB, error) {
	c := dsnConfigFromEnv()
	if applicationName != "" {
		c.ApplicationName = applicationName
	}
	cfg := LoadConfig()
	dsn := buildPostgresDSN(c, cfg.StatementTimeout, cfg.IdleInTransactionSessionTimeout)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if maxOpenConns <= 0 {
		maxOpenConns = 1
	}
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxOpenConns)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	return sqlDB, nil
}

func connect() *gorm.DB {
	c := dsnConfigFromEnv()
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
		DO $$
		DECLARE
			tables text;
		BEGIN
			SELECT string_agg(format('%I', tablename), ', ' ORDER BY tablename)
			INTO tables
			FROM pg_tables
			WHERE schemaname = 'public'
			  AND tablename != 'schema_migrations';

			IF tables IS NOT NULL THEN
				EXECUTE 'TRUNCATE TABLE ' || tables || ' RESTART IDENTITY CASCADE';
			END IF;
		END$$;
	`).Error
}
