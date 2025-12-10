package helpers

import (
	"fmt"
	"net/http"
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
	err := StartGlobalVCR(fmt.Sprintf("testdata/cassettes/%s.yaml", cassetteName))
	require.NoError(t, err)

	defer func() {
		require.NoError(t, StopGlobalVCR())
	}()

	testFunc(t)
}
