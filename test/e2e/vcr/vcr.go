package helpers

import (
	"net/http"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v2/cassette"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

var (
	// globalRecorder is the recorder used for the global VCR.
	globalRecorder *recorder.Recorder

	// Keep the original http transport to restore it back to normal after tests.
	originalTransport = http.DefaultTransport
)

func customMatcher(r *http.Request, i cassette.Request) bool {
	log.Printf("VCR: Trying to match request %s %s", r.Method, r.URL.String())
	log.Printf("VCR: Against cassette entry %s %s", i.Method, i.URL)

	matches := r.Method == i.Method && r.URL.String() == i.URL
	log.Printf("VCR: Match result: %v", matches)

	return matches
}

func Run(t *testing.T, testName string, testFunc func(t *testing.T)) {
	t.Run(testName, func(t *testing.T) {
		withVCR(t, testName, recorder.ModeReplaying, testFunc)
	})
}

func Record(t *testing.T, testName string, testFunc func(t *testing.T)) {
	if os.Getenv("CI") == "true" {
		t.Fatalf("Recording tests are not allowed to run in CI environment")
	}

	t.Run(testName, func(t *testing.T) {
		withVCR(t, testName, recorder.ModeRecording, testFunc)
	})
}

func withVCR(t *testing.T, testName string, mode recorder.Mode, testFunc func(t *testing.T)) {
	cassetteName := testNameToCassetteName(t, testName)

	err := startGlobalVCR(cassetteName, mode)
	require.NoError(t, err)

	defer func() {
		require.NoError(t, stopGlobalVCR())
	}()

	testFunc(t)
}

func startGlobalVCR(cassetteName string, mode recorder.Mode) error {
	r, err := recorder.NewAsMode(cassetteName, mode, nil)
	if err != nil {
		return err
	}

	r.AddPassthrough(localTraficPassthrough)
	r.SetMatcher(customMatcher)

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

func testNameToCassetteName(t *testing.T, testName string) string {
	cassetteName := t.Name() + "_" + testName
	cassetteName = strings.ReplaceAll(cassetteName, " ", "_")
	cassetteName = strings.ReplaceAll(cassetteName, "/", "_")

	return "vcr/cassettes/" + cassetteName
}
