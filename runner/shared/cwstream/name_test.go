package cwstream

import "testing"

func TestTaskLogStream(t *testing.T) {
	if got := TaskLogStream("", "abc"); got != "abc" {
		t.Fatalf("empty prefix: got %q", got)
	}
	if got := TaskLogStream("tasks", "abc"); got != "tasks/abc" {
		t.Fatalf("prefix: got %q", got)
	}
	if got := TaskLogStream("tasks/", "abc"); got != "tasks/abc" {
		t.Fatalf("prefix slash: got %q", got)
	}
}
