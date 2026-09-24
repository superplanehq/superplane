package livelogs

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestHubAppendAndWait(t *testing.T) {
	hub := NewHub()
	hub.Append("task-1", []byte("hello\n"))
	hub.Append("task-1", []byte(`{"type":"cmd_start","index":0,"text":"echo hello","started_at":1}`+"\n"))
	hub.Close("task-1")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	recs, next, closed, err := hub.Wait(ctx, "task-1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatalf("expected closed after Close")
	}
	if next != 2 || len(recs) != 2 {
		t.Fatalf("records: n=%d next=%d", len(recs), next)
	}

	var first map[string]any
	if err := json.Unmarshal(recs[0], &first); err != nil {
		t.Fatal(err)
	}
	if first["type"] != "line" || first["text"] != "hello" {
		t.Fatalf("first record: %#v", first)
	}

	var second map[string]any
	if err := json.Unmarshal(recs[1], &second); err != nil {
		t.Fatal(err)
	}
	if second["type"] != "cmd_start" || second["index"] != float64(0) {
		t.Fatalf("second record: %#v", second)
	}

	more, _, closed, err := hub.Wait(ctx, "task-1", next)
	if err != nil {
		t.Fatal(err)
	}
	if !closed || len(more) != 0 {
		t.Fatalf("tail: closed=%v n=%d", closed, len(more))
	}
}

func TestHubWaitUnblocksOnClose(t *testing.T) {
	hub := NewHub()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(20 * time.Millisecond)
		hub.Close("task-wait")
	}()

	_, _, closed, err := hub.Wait(ctx, "task-wait", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatalf("expected closed")
	}
	<-done
}

func TestRecordFromLogMessage(t *testing.T) {
	line := RecordFromLogMessage("stdout")
	if line["type"] != "line" || line["text"] != "stdout" {
		t.Fatalf("line: %#v", line)
	}
	start := RecordFromLogMessage(`{"type":"cmd_start","index":1,"text":"ls"}`)
	if start["type"] != "cmd_start" || start["index"] != 1 {
		t.Fatalf("cmd_start: %#v", start)
	}
}
