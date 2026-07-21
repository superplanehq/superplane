package models

import (
	"encoding/json"
	"testing"
)

func TestCommandListUnmarshalStringsAndObjects(t *testing.T) {
	t.Parallel()

	var list CommandList
	err := json.Unmarshal([]byte(`[
		"echo hi",
		{"name":"Clone","command":"git clone repo"},
		{"command":"echo only"},
		"  ",
		{"name":"Skip","command":"  "}
	]`), &list)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("len=%d want 3: %#v", len(list), list)
	}
	if list[0].Command != "echo hi" || list[0].Name != "" {
		t.Fatalf("list[0]=%#v", list[0])
	}
	if list[1].Name != "Clone" || list[1].Command != "git clone repo" {
		t.Fatalf("list[1]=%#v", list[1])
	}
	if list[2].Command != "echo only" {
		t.Fatalf("list[2]=%#v", list[2])
	}
	if list[1].DisplayText() != "Clone" {
		t.Fatalf("display=%q", list[1].DisplayText())
	}
	if list[0].DisplayText() != "echo hi" {
		t.Fatalf("display=%q", list[0].DisplayText())
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
