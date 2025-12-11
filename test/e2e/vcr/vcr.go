package helpers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

var (
	// globalRecorder is the recorder used for the global VCR.
	globalRecorder *recorder.Recorder

	// Keep the original http transport to restore it back to normal after tests.
	originalTransport = http.DefaultTransport
)

// WithVCR is the main entry point for using the global VCR in tests.
//
// It starts the global VCR with the given cassette name, runs the provided test function,
// and ensures the VCR is stopped afterwards.
//
// It fails the test if starting or stopping the VCR fails.
//
// Usage:
//
//	helpers.WithVCR(t, "my-test", func(t *testing.T) {
//	    // Your test code here
//	})
func Run(t *testing.T, testName string, testFunc func(t *testing.T)) {
	cassetteName := testNameToCassetteName(testName)

	t.Run(testName, func(t *testing.T) {
		err := startGlobalVCR(cassetteName)
		require.NoError(t, err)

		defer func() {
			require.NoError(t, stopGlobalVCR())
		}()

		testFunc(t)
	})
}

func startGlobalVCR(cassetteName string) error {
	r, err := recorder.NewAsMode(cassetteName, recorder.ModeReplayingOrRecording, nil)
	if err != nil {
		return err
	}

	r.AddPassthrough(localTraficPassthrough)

	globalRecorder = r
	http.DefaultTransport = r

	return nil
}

func stopGlobalVCR() error {
	http.DefaultTransport = originalTransport

	if globalRecorder != nil {
		err := globalRecorder.Stop()
		globalRecorder = nil
		return err
	}

	return nil
}

// HTTP trafic between the browser and the application server should not be recorded by VCR.
// This function identifies such trafic and allows it to passthrough the VCR recorder
// without being recorded or replayed.
func localTraficPassthrough(req *http.Request) bool {
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
}

// testNameToCassetteName converts a test name to a valid cassette file name.
// e.g. "Test My Feature/Subfeature" -> "Test_My_Feature_Subfeature"
func testNameToCassetteName(testName string) string {
	// Replace spaces and slashes with underscores to form a valid file name.
	cassetteName := strings.ReplaceAll(testName, " ", "_")
	cassetteName = strings.ReplaceAll(cassetteName, "/", "_")
	return cassetteName
}
