package telemetry

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
)

// InitSentry configures the global Sentry client using environment
// variables that are already used for the frontend:
//
//   - SENTRY_DSN
//   - SENTRY_ENVIRONMENT (falls back to APP_ENV if empty)
//
// If SENTRY_DSN is not set, this is a no-op.
func InitSentry() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return
	}

	environment := os.Getenv("SENTRY_ENVIRONMENT")
	if environment == "" {
		environment = os.Getenv("APP_ENV")
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:           dsn,
		Environment:   environment,
		EnableTracing: false,
	})
	if err != nil {
		log.Warnf("Failed to initialize Sentry: %v", err)
		return
	}

	log.Info("Sentry telemetry initialized")
}
