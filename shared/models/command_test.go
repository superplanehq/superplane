package models

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCommandListUnmarshalObjects(t *testing.T) {
	t.Parallel()

	var list CommandList
	err := json.Unmarshal([]byte(`[
		{"name":"Clone","command":"git clone repo"},
		{"command":"echo only"},
		{"name":"Skip","command":"  "}
	]`), &list)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len=%d want 2: %#v", len(list), list)
	}
	if list[0].Name != "Clone" || list[0].Command != "git clone repo" {
		t.Fatalf("list[0]=%#v", list[0])
	}
	if list[1].Command != "echo only" || list[1].Name != "" {
		t.Fatalf("list[1]=%#v", list[1])
	}
	if list[0].DisplayText() != "Clone" {
		t.Fatalf("display=%q", list[0].DisplayText())
	}
	if list[1].DisplayText() != "echo only" {
		t.Fatalf("display=%q", list[1].DisplayText())
	}
}

func TestCommandListUnmarshalRejectsPlainStrings(t *testing.T) {
	t.Parallel()

	var list CommandList
	err := json.Unmarshal([]byte(`["echo hi"]`), &list)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `must be an object with "command"`) {
		t.Fatalf("err=%v", err)
	}
}

func TestCommandListUnmarshalRequiresCommandOnObject(t *testing.T) {
	t.Parallel()

	var list CommandList
	err := json.Unmarshal([]byte(`[{"name":"Nope"}]`), &list)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCommandListMarshalRoundTrip(t *testing.T) {
	t.Parallel()

	in := CommandList{
		{Command: "echo hi"},
		{Name: "Clone", Command: "git clone"},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out CommandList
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[1].Name != "Clone" || out[0].Command != "echo hi" {
		t.Fatalf("out=%#v", out)
	}
}
