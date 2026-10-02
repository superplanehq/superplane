package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/superplanehq/superplane/pkg/runners/agent"
	"github.com/superplanehq/superplane/pkg/runners/protocol"
)

var Version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("runner stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	defaultWorkDirectory, err := agent.ResolveTaskWorkDir()
	if err != nil {
		return err
	}

	apiURL := flag.String(
		"url",
		env("RUNNER_API_URL", ""),
		"SuperPlane runner API base URL",
	)
	registrationToken := flag.String(
		"registration-token",
		env("RUNNER_REGISTRATION_TOKEN", ""),
		"single-use generic or task-specific registration token",
	)
	spoolDirectory := flag.String(
		"spool-directory",
		env(
			"RUNNER_LOG_SPOOL_DIRECTORY",
			os.TempDir()+"/superplane-runner-logs",
		),
		"directory for unacknowledged log chunks",
	)
	taskWorkDirectory := flag.String(
		"task-work-directory",
		env("RUNNER_TASK_WORK_DIRECTORY", defaultWorkDirectory),
		"host task working directory",
	)
	resetTaskHome := flag.Bool(
		"reset-task-home",
		envBool("RUNNER_RESET_TASK_HOME", false),
		"create a fresh HOME for the task",
	)
	maxExecutionSeconds := flag.Int(
		"max-execution-seconds",
		envInt("RUNNER_MAX_EXECUTION_SECONDS", 0),
		"maximum execution time in seconds; zero uses the task limit",
	)
	logChunkBytes := flag.Int64(
		"log-chunk-bytes",
		envInt64("RUNNER_LOG_CHUNK_BYTES", 64*1024),
		"target size for one log upload",
	)
	logSpoolMaxBytes := flag.Int64(
		"log-spool-max-bytes",
		envInt64("RUNNER_LOG_SPOOL_MAX_BYTES", 10*1024*1024),
		"maximum unacknowledged log bytes on disk",
	)
	flag.Parse()

	if strings.TrimSpace(*apiURL) == "" {
		return fmt.Errorf("--url or RUNNER_API_URL is required")
	}
	if strings.TrimSpace(*registrationToken) == "" {
		return fmt.Errorf(
			"--registration-token or RUNNER_REGISTRATION_TOKEN is required",
		)
	}

	startupContext, stopStartup := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stopStartup()

	httpClient := &http.Client{Timeout: 30 * time.Second}
	registrationContext, cancelRegistration := context.WithTimeout(startupContext, 30*time.Second)
	registration, err := protocol.Register(
		registrationContext,
		httpClient,
		*apiURL,
		*registrationToken,
		Version,
	)
	cancelRegistration()
	if err != nil {
		return err
	}
	stopStartup()

	shutdown := make(chan struct{}, 2)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go forwardShutdownSignals(signals, shutdown, slog.Default())

	config := agent.DefaultConfig()
	config.BaseURL = *apiURL
	config.Registration = registration
	config.Version = Version
	config.TaskWorkDir = *taskWorkDirectory
	config.ResetTaskHome = *resetTaskHome
	config.MaxExecutionSeconds = *maxExecutionSeconds
	config.LogSpoolDirectory = *spoolDirectory
	config.LogChunkBytes = *logChunkBytes
	config.LogSpoolMaxBytes = *logSpoolMaxBytes
	config.Shutdown = shutdown
	config.Log = slog.Default()

	slog.Info(
		"runner registered",
		slog.String("runner_id", registration.RunnerID),
		slog.String("fleet_id", registration.FleetID),
		slog.String("version", Version),
	)
	runner := &agent.Agent{HTTP: httpClient, Config: config}
	return runner.Run(context.Background())
}

func forwardShutdownSignals(
	signals <-chan os.Signal,
	shutdown chan<- struct{},
	log *slog.Logger,
) {
	for received := range signals {
		log.Info(
			"runner received shutdown signal",
			slog.String("signal", received.String()),
		)
		shutdown <- struct{}{}
	}
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt64(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
