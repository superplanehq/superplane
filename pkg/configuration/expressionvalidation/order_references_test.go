package expressionvalidation

import "testing"

func TestExpressionUsesOrderRepository(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "whole order", raw: `order()`, want: true},
		{name: "whole task", raw: `task()`, want: true},
		{name: "repository", raw: `order().repository`, want: true},
		{name: "repository URL", raw: `task().repository_url`, want: true},
		{name: "default branch", raw: `order()["default_branch"]`, want: true},
		{name: "aliased payload", raw: `let payload = task(); payload.repository`, want: true},
		{name: "dynamic property", raw: `order()[root().field]`, want: true},
		{name: "unrelated property", raw: `task().id`, want: false},
		{name: "nested unrelated property", raw: `order().source.repository`, want: false},
		{name: "multiple unrelated accesses", raw: `order().id + task().title`, want: false},
		{name: "mixed accesses", raw: `order().id + task().repository`, want: true},
		{name: "whole payload and unrelated access", raw: `[order().id, task()]`, want: true},
		{name: "unrelated function", raw: `root().repository`, want: false},
		{name: "no calls", raw: `"order().repository"`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderRepository(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("invalid expression", func(t *testing.T) {
		_, err := ExpressionUsesOrderRepository(`order(`)
		if err == nil {
			t.Fatal("expected a parse error")
		}
	})
}

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
		{name: "task alias", raw: `task().artifacts`, want: true},
		{name: "task alias none", raw: `none(task().artifacts, {#.type == "pr"})`, want: true},
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
		{name: "task alias", raw: `task().url`, want: true},
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

func TestExpressionUsesOrderKey(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "dot key", raw: `order().key`, want: true},
		{name: "bracket key", raw: `order()["key"]`, want: true},
		{name: "concatenated", raw: `"[" + order().key + "](" + order().url + ")"`, want: true},
		{name: "task alias", raw: `task().key`, want: true},
		{name: "unrelated", raw: `app().key`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderKey(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpressionUsesOrderSpec(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "order artifacts", raw: `order().artifacts`, want: false},
		{name: "dot spec", raw: `order().spec`, want: true},
		{name: "bracket spec", raw: `order()["spec"]`, want: true},
		{name: "ternary", raw: `task().spec != "" ? task().spec : ""`, want: true},
		{name: "task alias", raw: `task().spec`, want: true},
		{name: "let alias spec", raw: `let taskData = task(); taskData.spec`, want: true},
		{name: "let alias bracket spec", raw: `let taskData = task(); taskData["spec"]`, want: true},
		{name: "let alias order spec", raw: `let orderData = order(); orderData.spec`, want: true},
		{name: "let unrelated spec", raw: `let taskData = root(); taskData.spec`, want: false},
		{name: "let shadowed alias", raw: `let taskData = task(); let taskData = root(); taskData.spec`, want: false},
		{name: "let rebound alias", raw: `let taskData = root(); let taskData = task(); taskData.spec`, want: true},
		{name: "unrelated", raw: `root().data.spec`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderSpec(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExpressionUsesOrderAssignees(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "order only", raw: `order()`, want: false},
		{name: "order id", raw: `order().id`, want: false},
		{name: "order comments", raw: `order().comments`, want: false},
		{name: "dot assignees", raw: `order().assignees`, want: true},
		{name: "bracket assignees", raw: `order()["assignees"]`, want: true},
		{name: "indexed", raw: `order().assignees[0]`, want: true},
		{name: "nested email", raw: `order().assignees[0].email`, want: true},
		{name: "mixed bracket", raw: `order()["assignees"][0]["name"]`, want: true},
		{name: "len", raw: `len(order().assignees)`, want: true},
		{name: "filter", raw: `filter(order().assignees, {#.email != ""})`, want: true},
		{name: "task alias", raw: `task().assignees`, want: true},
		{name: "unrelated", raw: `root().data.work_order`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpressionUsesOrderAssignees(tc.raw)
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
		{name: "task alias", raw: `task().comments`, want: true},
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
