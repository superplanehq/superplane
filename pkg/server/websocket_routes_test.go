package server

import (
	"testing"

	runnerlogs "github.com/superplanehq/superplane/pkg/runners/logs"
)

func TestShouldRegisterWebSocketRoutes(t *testing.T) {
	tests := []struct {
		name            string
		websocketServer string
		webServer       string
		want            bool
	}{
		{name: "explicit yes", websocketServer: "yes", webServer: "no", want: true},
		{name: "explicit no", websocketServer: "no", webServer: "yes", want: false},
		{name: "unset follows web yes", websocketServer: "", webServer: "yes", want: true},
		{name: "unset follows web no", websocketServer: "", webServer: "no", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("START_WEBSOCKET_SERVER", tt.websocketServer)
			t.Setenv("START_WEB_SERVER", tt.webServer)
			if got := shouldRegisterWebSocketRoutes(); got != tt.want {
				t.Fatalf("shouldRegisterWebSocketRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldRegisterGRPCGateway(t *testing.T) {
	t.Run("unset defaults on", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "")
		if !shouldRegisterGRPCGateway() {
			t.Fatal("expected true when unset")
		}
	})
	t.Run("yes", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "yes")
		if !shouldRegisterGRPCGateway() {
			t.Fatal("expected true")
		}
	})
	t.Run("no", func(t *testing.T) {
		t.Setenv("START_GRPC_GATEWAY", "no")
		if shouldRegisterGRPCGateway() {
			t.Fatal("expected false")
		}
	})
}

func TestGetRunnerAPIBaseURL(t *testing.T) {
	t.Run("uses configured runner API URL", func(t *testing.T) {
		t.Setenv("RUNNER_API_BASE_URL", "https://runner.example")

		got := getRunnerAPIBaseURL("https://app.example")

		if got != "https://runner.example" {
			t.Fatalf("getRunnerAPIBaseURL() = %q, want %q", got, "https://runner.example")
		}
	})

	t.Run("falls back to application base URL", func(t *testing.T) {
		t.Setenv("RUNNER_API_BASE_URL", "")

		got := getRunnerAPIBaseURL("https://app.example")

		if got != "https://app.example" {
			t.Fatalf("getRunnerAPIBaseURL() = %q, want %q", got, "https://app.example")
		}
	})
}

func TestStartWorkersDoesNotRequireRegularWorkerDependenciesWhenDisabled(t *testing.T) {
	t.Setenv("START_REGULAR_WORKERS", "no")
	t.Setenv("START_RUNNER_LOG_COMPACTOR", "")
	t.Setenv("START_RUNNER_CLEANUP_WORKER", "")
	t.Setenv("RABBITMQ_URL", "")

	startWorkers(nil, nil, nil, "", nil, nil)
}

func TestNewRunnerActiveLogStoreUsesConfiguredPath(t *testing.T) {
	t.Setenv("RUNNER_ACTIVE_LOG_FS_PATH", t.TempDir())
	t.Setenv("START_RUNNER_API", "")
	t.Setenv("START_RUNNER_LOG_COMPACTOR", "")
	t.Setenv("START_RUNNER_CLEANUP_WORKER", "")

	store, err := newRunnerActiveLogStore()
	if err != nil {
		t.Fatalf("newRunnerActiveLogStore() error = %v", err)
	}
	if store == nil {
		t.Fatal("newRunnerActiveLogStore() = nil, want configured store")
	}
	if store.Name() != runnerlogs.StoreFS {
		t.Fatalf("newRunnerActiveLogStore().Name() = %q, want %q", store.Name(), runnerlogs.StoreFS)
	}
}

func TestNewRunnerActiveLogStoreDoesNotRequirePathForCleanupWorker(t *testing.T) {
	t.Setenv("RUNNER_ACTIVE_LOG_FS_PATH", "")
	t.Setenv("START_RUNNER_CLEANUP_WORKER", "yes")

	store, err := newRunnerActiveLogStore()
	if err != nil {
		t.Fatalf("newRunnerActiveLogStore() error = %v", err)
	}
	if store != nil {
		t.Fatalf("newRunnerActiveLogStore() = %T, want nil", store)
	}
}
