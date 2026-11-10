package workers

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
)

func TestRabbitMQConsumersStopDuringRetry(t *testing.T) {
	for _, name := range []string{"event router", "node executor"} {
		t.Run(name, func(t *testing.T) {
			entry, connectionFailed := loggerWaitingFor("Error consuming messages")
			start := rabbitMQConsumer(name, "://invalid", entry)
			ctx, cancel := context.WithCancel(t.Context())
			done := startConsumer(ctx, start)

			requireSignal(t, connectionFailed, "consumer did not enter retry")
			cancel()
			requireSignal(t, done, "consumer did not stop during retry")
		})
	}
}

func TestRabbitMQConsumersStopWhileListening(t *testing.T) {
	rabbitMQURL, err := config.RabbitMQURL()
	if err != nil {
		t.Skip("RabbitMQ is not configured")
	}

	for _, name := range []string{"event router", "node executor"} {
		t.Run(name, func(t *testing.T) {
			entry, listening := loggerWaitingFor("started listening for messages")
			start := rabbitMQConsumer(name, rabbitMQURL, entry)
			ctx, cancel := context.WithCancel(t.Context())
			done := startConsumer(ctx, start)

			requireSignal(t, listening, "consumer did not start listening")
			cancel()
			requireSignal(t, done, "consumer did not stop while listening")
		})
	}
}

func rabbitMQConsumer(name, rabbitMQURL string, logger *log.Entry) func(context.Context) {
	if name == "event router" {
		worker := NewEventRouter(rabbitMQURL)
		worker.logger = logger
		return worker.StartRabbitMQConsumer
	}
	worker := NewNodeExecutor(nil, nil, nil, "", "", rabbitMQURL, nil)
	worker.logger = logger
	return worker.StartRabbitMQConsumer
}

func startConsumer(ctx context.Context, start func(context.Context)) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		start(ctx)
		close(done)
	}()
	return done
}

func requireSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

type messageHook struct {
	match   string
	reached chan struct{}
	once    sync.Once
}

func loggerWaitingFor(match string) (*log.Entry, <-chan struct{}) {
	hook := &messageHook{match: match, reached: make(chan struct{})}
	logger := log.New()
	logger.AddHook(hook)
	return log.NewEntry(logger), hook.reached
}

func (h *messageHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *messageHook) Fire(entry *log.Entry) error {
	if strings.Contains(entry.Message, h.match) {
		h.once.Do(func() { close(h.reached) })
	}
	return nil
}
