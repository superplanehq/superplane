package runner

import (
	"reflect"
	"testing"
)

func TestNormalizeCommands(t *testing.T) {
	t.Parallel()
	got := normalizeCommands("echo a\n\n  echo b  \n")
	want := []string{"echo a", "echo b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeCommands: got %#v want %#v", got, want)
	}
	if len(normalizeCommands("")) != 0 {
		t.Fatal("empty input should yield empty slice")
	}
	if len(normalizeCommands("\n \n")) != 0 {
		t.Fatal("blank-only lines should yield empty slice")
	}
}
