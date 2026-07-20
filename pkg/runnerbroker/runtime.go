package runnerbroker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/runnerbroker/broker"
	"github.com/superplanehq/superplane/pkg/runnerbroker/store"
	"github.com/superplanehq/superplane/pkg/runnerbroker/webhook"
)

const (
	envEnabled        = "RUNNER_BROKER_ENABLED"
	envAuthToken      = "RUNNER_BROKER_AUTH_TOKEN"
	envLegacyAuth     = "TASK_BROKER_AUTH_TOKEN"
	envReapInterval   = "RUNNER_BROKER_REAP_INTERVAL_SEC"
	defaultReapPeriod = 15 * time.Second
)

type Service struct {
	Handler http.Handler
	server  *broker.Server
	store   *store.PostgresStore
	log     *slog.Logger
}

func Enabled() bool {
	return strings.TrimSpace(os.Getenv(envEnabled)) == "yes"
}

func NewFromEnv() (*Service, error) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	auth := strings.TrimSpace(os.Getenv(envAuthToken))
	if auth == "" {
		auth = strings.TrimSpace(os.Getenv(envLegacyAuth))
	}
	if auth == "" {
		return nil, fmt.Errorf("%s or %s is required when %s=yes", envAuthToken, envLegacyAuth, envEnabled)
	}

	st := store.NewPostgresStore(database.Conn())
	ws := webhook.DefaultSender()
	ws.Log = log

	srv := &broker.Server{
		Store:                         st,
		Webhook:                       ws,
		Log:                           log,
		TaskNotify:                    broker.NewWaitHub(),
		RunnerCancel:                  broker.NewRunnerCancelHub(),
		RunnerDrain:                   broker.NewRunnerDrainHub(),
		TaskCloudWatchLogGroup:        strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_GROUP")),
		TaskCloudWatchLogStreamPrefix: strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_LOG_STREAM_PREFIX")),
		TaskCloudWatchRegion:          strings.TrimSpace(os.Getenv("TASK_CLOUDWATCH_REGION")),
	}

	return &Service{
		Handler: broker.NewRouter(srv, broker.RouterOptions{
			AuthToken:           auth,
			LiveLogsCORSOrigins: broker.ParseLiveLogsCORSOrigins(os.Getenv("TASK_BROKER_LIVE_LOGS_CORS_ORIGINS")),
		}),
		server: srv,
		store:  st,
		log:    log,
	}, nil
}

func (s *Service) Start(ctx context.Context) {
	go s.reapExpiredLeases(ctx, reapInterval())
}

func (s *Service) reapExpiredLeases(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			requeuedTasks, canceledTasks, err := s.store.ReapExpiredLeases(context.Background())
			if err != nil {
				s.log.Warn("runner broker reap leases", slog.Any("err", err))
				if len(requeuedTasks) == 0 && len(canceledTasks) == 0 {
					continue
				}
			}
			for _, lease := range requeuedTasks {
				s.server.RecordLeaseReaped(context.Background(), lease.FleetID)
			}
			for _, task := range canceledTasks {
				t := task
				s.server.RecordLeaseReaped(context.Background(), t.FleetID)
				s.server.RecordTaskCompleted(context.Background(), t)
				go s.server.DeliverWebhook(t)
			}
			if len(requeuedTasks) > 0 {
				s.log.Info("runner broker reaped expired task leases", slog.Int("count", len(requeuedTasks)))
			}
			if len(canceledTasks) > 0 {
				s.log.Info("runner broker finalized canceled tasks after lease expiry", slog.Int("count", len(canceledTasks)))
			}
		}
	}
}

func reapInterval() time.Duration {
	raw := strings.TrimSpace(os.Getenv(envReapInterval))
	if raw == "" {
		return defaultReapPeriod
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultReapPeriod
	}
	return time.Duration(n) * time.Second
}
