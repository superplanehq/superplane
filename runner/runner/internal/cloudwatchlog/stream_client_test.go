package cloudwatchlog

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestStreamWriterPutLogEvents(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	var mu sync.Mutex
	var messages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		body, _ := io.ReadAll(r.Body)
		switch r.Header.Get("X-Amz-Target") {
		case "Logs_20140328.CreateLogGroup", "Logs_20140328.CreateLogStream":
			_, _ = w.Write([]byte(`{}`))
		case "Logs_20140328.PutLogEvents":
			var in struct {
				LogEvents []struct {
					Message string `json:"message"`
				} `json:"logEvents"`
			}
			if err := json.Unmarshal(body, &in); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			for _, ev := range in.LogEvents {
				messages = append(messages, ev.Message)
			}
			mu.Unlock()
			_, _ = w.Write([]byte(`{"nextSequenceToken":"2"}`))
		default:
			http.Error(w, "unexpected target "+r.Header.Get("X-Amz-Target"), http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)

	writer, err := NewStreamWriter(context.Background(), StreamConfig{
		LogGroup:   "tasks",
		StreamName: "task-1",
		Region:     "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("hello from runner\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	got := append([]string(nil), messages...)
	mu.Unlock()
	if len(got) != 1 || got[0] != "hello from runner" {
		t.Fatalf("messages = %#v", got)
	}
}
