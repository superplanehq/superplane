package database

import (
	"testing"
	"time"
)

func TestLoadConfig_defaults(t *testing.T) {
	t.Setenv(envStatementTimeout, "")
	t.Setenv(envIdleInTxTimeout, "")
	cfg := LoadConfig()
	if cfg.StatementTimeout != DefaultStatementTimeout {
		t.Fatalf("StatementTimeout: %v", cfg.StatementTimeout)
	}
	if cfg.IdleInTransactionSessionTimeout != DefaultIdleInTransactionSessionTimeout {
		t.Fatalf("IdleInTransactionSessionTimeout: %v", cfg.IdleInTransactionSessionTimeout)
	}
}

func TestLoadConfig_durationSyntax(t *testing.T) {
	t.Setenv(envStatementTimeout, "2m")
	t.Setenv(envIdleInTxTimeout, "45s")
	cfg := LoadConfig()
	if cfg.StatementTimeout != 2*time.Minute || cfg.IdleInTransactionSessionTimeout != 45*time.Second {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadConfig_legacyBareMilliseconds(t *testing.T) {
	t.Setenv(envStatementTimeout, "120000")
	t.Setenv(envIdleInTxTimeout, "60000")
	cfg := LoadConfig()
	if cfg.StatementTimeout != 120*time.Second || cfg.IdleInTransactionSessionTimeout != 60*time.Second {
		t.Fatalf("got %+v", cfg)
	}
}

func TestLoadConfig_invalidFallsBack(t *testing.T) {
	t.Setenv(envStatementTimeout, "not-a-duration")
	t.Setenv(envIdleInTxTimeout, "30s")
	cfg := LoadConfig()
	if cfg.StatementTimeout != DefaultStatementTimeout {
		t.Fatalf("expected default statement timeout, got %v", cfg.StatementTimeout)
	}
	if cfg.IdleInTransactionSessionTimeout != 30*time.Second {
		t.Fatalf("IdleInTransactionSessionTimeout: %v", cfg.IdleInTransactionSessionTimeout)
	}
}
