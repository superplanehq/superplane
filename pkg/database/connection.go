package database

import (
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

	return db
}

func TruncateTables() error {
	return Conn().Exec(`
		truncate table canvases, event_rejections, events, event_sources, stages,
		stage_events, stage_event_approvals,
		connections, stage_executions,
		connection_groups, connection_group_field_sets, connection_group_field_set_events,
		integrations, resources,
		execution_resources,
		secrets, accounts, account_providers, users, organizations, organization_invitations,
		casbin_rule, role_metadata, group_metadata, alerts;
	`).Error
}
