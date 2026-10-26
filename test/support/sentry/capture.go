package sentrytest

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
)

// Capture records events from a test Sentry client without external delivery.
type Capture struct {
	mu     sync.Mutex
	events []*sentry.Event
}

// BindHub captures the current hub's events and restores its client after the test.
func BindHub(t *testing.T) *Capture {
	t.Helper()
	transport := &Capture{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@localhost/1",
		Transport: transport,
	})
	require.NoError(t, err)
	hub := sentry.CurrentHub()
	previous := hub.Client()
	hub.BindClient(client)
	t.Cleanup(func() { hub.BindClient(previous) })
	return transport
}

func (*Capture) Configure(sentry.ClientOptions) {}

func (*Capture) Flush(time.Duration) bool { return true }

func (c *Capture) SendEvent(event *sentry.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	clone := *event
	c.events = append(c.events, &clone)
}

// Events returns the events that the client sent to this capture.
func (c *Capture) Events() []*sentry.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*sentry.Event(nil), c.events...)
}

// ExceptionValues returns every exception message in the event.
func ExceptionValues(event *sentry.Event) []string {
	values := make([]string, 0, len(event.Exception))
	for _, exception := range event.Exception {
		values = append(values, exception.Value)
	}
	return values
}

// ExceptionText joins the exception messages for assertions on an error chain.
func ExceptionText(event *sentry.Event) string {
	return strings.Join(ExceptionValues(event), "\n")
}
