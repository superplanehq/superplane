package runner

import (
	"reflect"
	"testing"
)

func TestNormalizeLines(t *testing.T) {
	t.Parallel()
	got := normalizeLines("echo a\n\n  echo b  \n")
	want := []string{"echo a", "echo b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeLines: got %#v want %#v", got, want)
	}
	if len(normalizeLines("")) != 0 {
		t.Fatal("empty input should yield empty slice")
	}
	if len(normalizeLines("\n \n")) != 0 {
		t.Fatal("blank-only lines should yield empty slice")
	}
}
