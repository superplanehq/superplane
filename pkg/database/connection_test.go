package database

import (
	"testing"
	"time"
)

func TestEnvOrDefault(t *testing.T) {
	result := envOrDefault("NON_EXISTENT_ENV_VAR_FOR_TEST", "default_value")
	if result != "default_value" {
		t.Errorf("expected 'default_value', got '%s'", result)
	}

	t.Setenv("TEST_ENV_VAR_FOR_DB_TIMEOUT", "custom_value")
	result = envOrDefault("TEST_ENV_VAR_FOR_DB_TIMEOUT", "default_value")
	if result != "custom_value" {
		t.Errorf("expected 'custom_value', got '%s'", result)
	}
}

func TestTimeoutConstants(t *testing.T) {
	if DefaultTransactionTimeout != 30*time.Second {
		t.Errorf("expected DefaultTransactionTimeout to be 30s, got %s", DefaultTransactionTimeout)
	}

	if DefaultWorkerTransactionTimeout != 2*time.Minute {
		t.Errorf("expected DefaultWorkerTransactionTimeout to be 2m, got %s", DefaultWorkerTransactionTimeout)
	}

	if DefaultReadOnlyTransactionTimeout != 15*time.Second {
		t.Errorf("expected DefaultReadOnlyTransactionTimeout to be 15s, got %s", DefaultReadOnlyTransactionTimeout)
	}

	if DefaultCanvasMutationTimeout != 45*time.Second {
		t.Errorf("expected DefaultCanvasMutationTimeout to be 45s, got %s", DefaultCanvasMutationTimeout)
	}

	if DefaultEventProcessingTimeout != 60*time.Second {
		t.Errorf("expected DefaultEventProcessingTimeout to be 60s, got %s", DefaultEventProcessingTimeout)
	}

	if DefaultCleanupTransactionTimeout != 2*time.Minute {
		t.Errorf("expected DefaultCleanupTransactionTimeout to be 2m, got %s", DefaultCleanupTransactionTimeout)
	}

	if DefaultAuthTransactionTimeout != 15*time.Second {
		t.Errorf("expected DefaultAuthTransactionTimeout to be 15s, got %s", DefaultAuthTransactionTimeout)
	}
}
