package livelogs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamCloudWatchLogPagesEvents(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Header.Get("X-Amz-Target") != "Logs_20140328.GetLogEvents" {
			http.Error(w, "unexpected target "+r.Header.Get("X-Amz-Target"), http.StatusBadRequest)
			return
		}
		calls++
		if calls == 1 {
			_, _ = w.Write([]byte(`{"events":[{"message":"paged line","timestamp":1}],"nextForwardToken":"page-2","nextBackwardToken":"start"}`))
			return
		}
		_, _ = w.Write([]byte(`{"events":[],"nextForwardToken":"page-2","nextBackwardToken":"page-2"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var buf bytes.Buffer
	err := StreamCloudWatchLogToNDJSON(ctx, cancelAfter{&buf, cancel}, nil, "tasks", "task-1", "us-east-1")
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"text":"paged line"`) {
		t.Fatalf("output = %s", buf.String())
	}
	if calls < 1 {
		t.Fatal("expected GetLogEvents")
	}
}

type cancelAfter struct {
	buf    *bytes.Buffer
	cancel context.CancelFunc
}

func (c cancelAfter) Write(p []byte) (int, error) {
	n, err := c.buf.Write(p)
	c.cancel()
	return n, err
}
