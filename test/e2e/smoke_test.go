package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/superplanehq/superplane/pkg/server"
)

func TestSmoke(t *testing.T) {
	go server.Start()

	client := &http.Client{Timeout: 1 * time.Second}
	deadline := time.Now().Add(20 * time.Second)

	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://127.0.0.1:8000/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return
			}
		} else {
			lastErr = err
		}

		time.Sleep(200 * time.Millisecond)
	}

	t.Fatalf("server did not come online in time: %v", lastErr)
}
