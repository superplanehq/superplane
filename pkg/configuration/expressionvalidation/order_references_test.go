package expressionvalidation

import "testing"

func TestExpressionUsesOrderArtifacts(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "order source", raw: `order().source.issue`, want: false},
		{name: "dot artifacts", raw: `order().artifacts`, want: true},
		{name: "bracket artifacts", raw: `order()["artifacts"]`, want: true},
		{name: "indexed", raw: `order().artifacts[0]`, want: true},
		{name: "nested type", raw: `order().artifacts[0].type`, want: true},
		{name: "mixed bracket", raw: `order()["artifacts"][0]["type"]`, want: true},
		{name: "len", raw: `len(order().artifacts)`, want: true},
		{name: "none predicate", raw: `none(order().artifacts, {#.type == "pr"})`, want: true},
		{name: "any predicate", raw: `any(order().artifacts, {#.type == "pr"})`, want: true},
		{name: "unrelated", raw: `root().data.work_order`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderArtifacts(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpressionUsesOrderURL(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "artifact url", raw: `order().artifacts[0].data.url`, want: false},
		{name: "dot url", raw: `order().url`, want: true},
		{name: "bracket url", raw: `order()["url"]`, want: true},
		{name: "concatenated", raw: `"[Work Order](" + order().url + ")"`, want: true},
		{name: "unrelated", raw: `app().url`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderURL(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpressionUsesOrderComments(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "order artifacts", raw: `order().artifacts`, want: false},
		{name: "dot comments", raw: `order().comments`, want: true},
		{name: "bracket comments", raw: `order()["comments"]`, want: true},
		{name: "indexed", raw: `order().comments[0]`, want: true},
		{name: "nested body", raw: `order().comments[0].body`, want: true},
		{name: "mixed bracket", raw: `order()["comments"][0]["body"]`, want: true},
		{name: "len", raw: `len(order().comments)`, want: true},
		{name: "none predicate", raw: `none(order().comments, {#.author.kind == "automation"})`, want: true},
		{name: "any predicate", raw: `any(order().comments, {#.author.kind == "automation"})`, want: true},
		{name: "unrelated", raw: `root().data.work_order`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderComments(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
