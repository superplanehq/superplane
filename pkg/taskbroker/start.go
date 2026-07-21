package taskbroker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/taskbroker/broker"
	brokermetrics "github.com/superplanehq/superplane/pkg/taskbroker/metrics"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/webhook"
	"github.com/superplanehq/superplane/pkg/taskbroker/store"
	"go.opentelemetry.io/otel"
)

// Start runs the task-broker HTTP API and background loops until ctx is cancelled.
// It uses SuperPlane's shared Postgres connection and listens on TASK_BROKER_LISTEN_ADDR
// (default :8081). Auth is TASK_BROKER_AUTH_TOKEN.
func Start(ctx context.Context) error {
	auth := strings.TrimSpace(os.Getenv("TASK_BROKER_AUTH_TOKEN"))
	if auth == "" {
		return fmt.Errorf("TASK_BROKER_AUTH_TOKEN is required when START_TASK_BROKER=yes")
	}

	slogLog := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	st := store.NewPostgresStore(database.Conn())

	var brokerMetrics *brokermetrics.BrokerMetrics
	if os.Getenv("OTEL_ENABLED") == "yes" {
		bm, err := brokermetrics.New(otel.Meter("task-broker"))
		if err != nil {
			return fmt.Errorf("init broker metrics: %w", err)
		}
		brokerMetrics = bm
	}

	ws := webhook.DefaultSender()
	ws.Log = slogLog
	hub := broker.NewWaitHub()
	cancelHub := broker.NewRunnerCancelHub()
	drainHub := broker.NewRunnerDrainHub()
	srv := &broker.Server{
		Store:                         st,
		Webhook:                       ws,
		Log:                           slogLog,
		Metrics:                       brokerMetrics,
		TaskNotify:                    hub,
		RunnerCancel:                  cancelHub,
		RunnerDrain:                   drainHub,
		TaskCloudWatchLogGroup:        strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_GROUP")),
		TaskCloudWatchLogStreamPrefix: strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_STREAM_PREFIX")),
		TaskCloudWatchRegion:          strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_REGION")),
	}

	liveLogsCORSOrigins := broker.ParseLiveLogsCORSOrigins(os.Getenv("TASK_BROKER_LIVE_LOGS_CORS_ORIGINS"))
	handler := broker.NewRouter(srv, broker.RouterOptions{
		AuthToken:           auth,
		LiveLogsCORSOrigins: liveLogsCORSOrigins,
	})

	addr := getenv("TASK_BROKER_LISTEN_ADDR", ":8081")
	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	reapInterval := 15 * time.Second
	if v := getenv("REAP_INTERVAL_SEC", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			reapInterval = time.Duration(n) * time.Second
		}
	}
	go runLeaseReaper(ctx, slogLog, st, srv, reapInterval)

	if brokerMetrics != nil {
		sampleInterval := 30 * time.Second
		if v := getenv("METRICS_SAMPLE_INTERVAL_SEC", ""); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				sampleInterval = time.Duration(n) * time.Second
			}
		}
		go runMetricsSampler(ctx, slogLog, st, brokerMetrics, sampleInterval)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Infof("Task broker listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		log.Info("Task broker shutdown complete")
		return nil
	case err := <-errCh:
		return fmt.Errorf("task broker http server: %w", err)
	}
}

func runLeaseReaper(ctx context.Context, slogLog *slog.Logger, st store.Store, srv *broker.Server, reapInterval time.Duration) {
	t := time.NewTicker(reapInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			requeuedTasks, canceledTasks, err := st.ReapExpiredLeases(context.Background())
			if err != nil {
				slogLog.Warn("reap leases", slog.Any("err", err))
				if len(requeuedTasks) == 0 && len(canceledTasks) == 0 {
					continue
				}
			}
			reapCtx := context.Background()
			for _, lease := range requeuedTasks {
				srv.RecordLeaseReaped(reapCtx, lease.FleetID)
			}
			for _, task := range canceledTasks {
				canceled := task
				srv.RecordLeaseReaped(reapCtx, canceled.FleetID)
				srv.RecordTaskCompleted(reapCtx, canceled)
				go srv.DeliverWebhook(canceled)
			}
			if len(requeuedTasks) > 0 {
				slogLog.Info("reaped expired task leases", slog.Int("count", len(requeuedTasks)))
			}
			if len(canceledTasks) > 0 {
				slogLog.Info("finalized canceled tasks after lease expiry", slog.Int("count", len(canceledTasks)))
			}
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
