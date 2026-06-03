package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/superplane/runner/shared/telemetry"
	"github.com/superplane/runner/shared/webhook"
	"github.com/superplane/runner/task-broker/internal/broker"
	brokermetrics "github.com/superplane/runner/task-broker/internal/metrics"
	"github.com/superplane/runner/task-broker/internal/store"
	"go.opentelemetry.io/otel"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	databaseURL := getenv("DATABASE_URL", "")
	if databaseURL == "" {
		log.Error("DATABASE_URL is required — postgres connection string, e.g. postgres://user:pass@host:5432/broker?sslmode=disable")
		os.Exit(1)
	}
	st, err := store.OpenPostgres(databaseURL)
	if err != nil {
		log.Error("open database", slog.Any("err", err))
		os.Exit(1)
	}
	defer st.Close()

	telemetryShutdown, metricsEnabled, err := telemetry.Init(context.Background())
	if err != nil {
		log.Error("init telemetry", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() {
		_ = telemetryShutdown(context.Background())
	}()
	if metricsEnabled {
		log.Info("metrics export enabled")
	}

	var brokerMetrics *brokermetrics.BrokerMetrics
	if metricsEnabled {
		bm, err := brokermetrics.New(otel.Meter("task-broker"))
		if err != nil {
			log.Error("init broker metrics", slog.Any("err", err))
			os.Exit(1)
		}
		brokerMetrics = bm
	}

	ws := webhook.DefaultSender()
	ws.Log = log
	hub := broker.NewWaitHub()
	cancelHub := broker.NewRunnerCancelHub()
	srv := &broker.Server{
		Store:                         st,
		Webhook:                       ws,
		Log:                           log,
		Metrics:                       brokerMetrics,
		TaskNotify:                    hub,
		RunnerCancel:                  cancelHub,
		TaskCloudWatchLogGroup:        strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_GROUP")),
		TaskCloudWatchLogStreamPrefix: strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_STREAM_PREFIX")),
		TaskCloudWatchRegion:          strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_REGION")),
	}

	auth := strings.TrimSpace(os.Getenv("AUTH_TOKEN"))
	if auth == "" {
		log.Error("AUTH_TOKEN is required — set a non-empty secret; clients send Authorization: Bearer <token> for /v1")
		os.Exit(1)
	}
	liveLogsCORSOrigins := broker.ParseLiveLogsCORSOrigins(os.Getenv("TASK_BROKER_LIVE_LOGS_CORS_ORIGINS"))
	handler := broker.NewRouter(srv, broker.RouterOptions{
		AuthToken:           auth,
		LiveLogsCORSOrigins: liveLogsCORSOrigins,
	})

	addr := getenv("LISTEN_ADDR", ":8081")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	reapInterval := 15 * time.Second
	if v := getenv("REAP_INTERVAL_SEC", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			reapInterval = time.Duration(n) * time.Second
		}
	}
	go func() {
		t := time.NewTicker(reapInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				requeuedTasks, canceledTasks, err := st.ReapExpiredLeases(context.Background())
				if err != nil {
					log.Warn("reap leases", slog.Any("err", err))
					continue
				}
				reapCtx := context.Background()
				for _, lease := range requeuedTasks {
					srv.RecordLeaseReaped(reapCtx, lease.FleetID)
				}
				for _, task := range canceledTasks {
					t := task
					srv.RecordLeaseReaped(reapCtx, t.FleetID)
					srv.RecordTaskCompleted(reapCtx, t)
					go srv.DeliverWebhook(t)
				}
				if len(requeuedTasks) > 0 {
					log.Info("reaped expired task leases", slog.Int("count", len(requeuedTasks)))
				}
				if len(canceledTasks) > 0 {
					log.Info("finalized canceled tasks after lease expiry", slog.Int("count", len(canceledTasks)))
				}
			}
		}
	}()

	if brokerMetrics != nil {
		sampleInterval := 30 * time.Second
		if v := getenv("METRICS_SAMPLE_INTERVAL_SEC", ""); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				sampleInterval = time.Duration(n) * time.Second
			}
		}
		go runMetricsSampler(ctx, log, st, brokerMetrics, sampleInterval)
	}

	go func() {
		log.Info("task-broker listening", slog.String("addr", addr))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server", slog.Any("err", err))
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
