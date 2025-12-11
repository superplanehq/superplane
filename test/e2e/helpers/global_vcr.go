package helpers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

var (
	globalRecorder    *recorder.Recorder
	originalTransport = http.DefaultTransport
)

// StartGlobalVCR replaces the default HTTP transport with a VCR recorder
// so that all outbound HTTP calls are recorded/replayed.
func StartGlobalVCR(cassetteName string) error {
	r, err := recorder.NewAsMode(cassetteName, recorder.ModeReplayingOrRecording, nil)
	if err != nil {
		return err
	}

	// Let local/dev HTTP traffic bypass VCR so we only
	// record external calls (e.g. GitHub API).
	r.AddPassthrough(func(req *http.Request) bool {
		host := req.URL.Host
		if host == "" {
			return false
		}

		// Strip port if present.
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		switch host {
		case "localhost", "127.0.0.1":
			return true
		default:
			return false
		}
	})

	globalRecorder = r
	http.DefaultTransport = r

	return nil
}

// StopGlobalVCR restores the original HTTP transport and stops the recorder.
func StopGlobalVCR() error {
	http.DefaultTransport = originalTransport

	if globalRecorder != nil {
		err := globalRecorder.Stop()
		globalRecorder = nil
		return err
	}

	return nil
}

// WithVCR is a test helper that wraps a test in a global VCR cassette.
func WithVCR(t *testing.T, cassetteName string, testFunc func(t *testing.T)) {
	err := StartGlobalVCR(cassetteName)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, StopGlobalVCR())
	}()

	testFunc(t)
}
