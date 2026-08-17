package telemetry

import (
	"context"
	"sync"
)

/*
 * Errors are sanitized before they reach the client, so a 5xx response carries
 * nothing about what actually failed. The HTTP layer that reports the failure
 * only sees a status code, which is not enough to act on.
 *
 * A ServerErrorRecorder travels with the request context so the layer that
 * still holds the original error can hand it to the layer that reports it.
 */
type ServerErrorRecorder struct {
	mu  sync.Mutex
	err error
}

type serverErrorRecorderKey struct{}

// WithServerErrorRecorder attaches an empty recorder to ctx and returns it
// together with the derived context.
func WithServerErrorRecorder(ctx context.Context) (context.Context, *ServerErrorRecorder) {
	recorder := &ServerErrorRecorder{}
	return context.WithValue(ctx, serverErrorRecorderKey{}, recorder), recorder
}

// RecordServerError stores err on the recorder attached to ctx. It is a no-op
// when err is nil or when no recorder is attached, so call sites do not have to
// care whether they run inside an instrumented HTTP request.
func RecordServerError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	recorder := serverErrorRecorderFrom(ctx)
	if recorder == nil {
		return
	}

	recorder.record(err)
}

// Err returns the recorded error, or nil when nothing was recorded.
func (r *ServerErrorRecorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.err
}

// record keeps the first error it is given. That is the one closest to the root
// cause: anything recorded later in the same request is a reaction to it.
func (r *ServerErrorRecorder) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil {
		return
	}

	r.err = err
}

func serverErrorRecorderFrom(ctx context.Context) *ServerErrorRecorder {
	if ctx == nil {
		return nil
	}

	recorder, _ := ctx.Value(serverErrorRecorderKey{}).(*ServerErrorRecorder)
	return recorder
}
