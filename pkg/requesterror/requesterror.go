// Package requesterror carries the server-side error that caused an HTTP
// response, from the handler that sanitizes the error to the middleware that
// reports it.
//
// Responses are sanitized before they reach the client, so the cause is lost
// after the handler returns. Without this package the error report contains
// the status code and the path only.
package requesterror

import (
	"context"
	"sync"
)

type contextKey struct{}

// Recorder holds the first server-side error of one HTTP request.
type Recorder struct {
	mu  sync.Mutex
	err error
}

// NewContext returns a context that carries a new Recorder, and the Recorder.
func NewContext(ctx context.Context) (context.Context, *Recorder) {
	recorder := &Recorder{}
	return context.WithValue(ctx, contextKey{}, recorder), recorder
}

// Record stores err in the Recorder of ctx. The first error wins, because it
// is the error that is nearest to the cause. Record does nothing when err is
// nil, or when ctx has no Recorder.
func Record(ctx context.Context, err error) {
	if err == nil {
		return
	}

	recorder, ok := ctx.Value(contextKey{}).(*Recorder)
	if !ok {
		return
	}

	recorder.record(err)
}

// Err returns the recorded error, or nil if the request caused no error.
func (r *Recorder) Err() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r *Recorder) record(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil {
		return
	}

	r.err = err
}
