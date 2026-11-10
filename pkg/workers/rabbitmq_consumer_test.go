package workers

import (
	"context"
	"testing"
	"time"
)

func TestRabbitMQConsumersStopWhenContextIsCanceled(t *testing.T) {
	for _, test := range []struct {
		name  string
		start func(context.Context)
	}{
		{name: "event router", start: NewEventRouter("").StartRabbitMQConsumer},
		{name: "node executor", start: NewNodeExecutor(nil, nil, nil, "", "", "", nil).StartRabbitMQConsumer},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			done := make(chan struct{})
			go func() {
				test.start(ctx)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("consumer did not stop after cancellation")
			}
		})
	}
}
