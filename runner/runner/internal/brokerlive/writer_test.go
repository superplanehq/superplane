package brokerlive

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriterPostsChunks(t *testing.T) {
	var bodies [][]byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/v1/tasks/task-1/live-log-events" {
			t.Errorf("path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Errorf("auth: %s", got)
		}
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	w := NewWriter(ts.Client(), ts.URL, "tok", "task-1")
	if _, err := w.Write([]byte("hello\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if len(bodies) == 0 {
		t.Fatal("expected at least one POST")
	}
	joined := ""
	for _, b := range bodies {
		joined += string(b)
	}
	if joined != "hello\n" {
		t.Fatalf("body: %q", joined)
	}
}
