import { describe, it, expect } from "vitest";
import { getSuggestions } from "./core";

describe("getSuggestions", () => {
  it("suggests env keys after $ trigger", () => {
    const suggestions = getSuggestions("take($", "take($".length, { foo: 1, bar: 2 });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain('["foo"]');
    expect(suggestions.some((item) => item.insertText === '$["foo"]')).toBe(true);
  });

  it("suggests dot fields based on resolved globals", () => {
    const suggestions = getSuggestions("$.user.", "$.user.".length, { user: { name: "Ana", age: 33 } });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("name");
    expect(labels).toContain("age");
  });

  it("adds a dot for expandable fields but skips empty objects", () => {
    const suggestions = getSuggestions("$.user.", "$.user.".length, {
      user: { filled: { ok: true }, empty: {} },
    });
    const filled = suggestions.find((item) => item.label === "filled");
    const empty = suggestions.find((item) => item.label === "empty");
    expect(filled?.insertText).toBe("filled.");
    expect(empty?.insertText).toBe("empty");
  });

  it("filters out internal metadata keys from dot suggestions", () => {
    const suggestions = getSuggestions("$.user.", "$.user.".length, {
      user: { name: "Ana", __nodeName: "User Node" },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("name");
    expect(labels).not.toContain("__nodeName");
  });

  it("includes built-in functions by prefix", () => {
    const suggestions = getSuggestions("tr", 2, {});
    expect(suggestions.some((item) => item.label === "trim")).toBe(true);
  });

  it("suggests root() payload fields after dot", () => {
    const suggestions = getSuggestions("root().", "root().".length, {
      __root: { github: { ref: "main" }, user: "alice" },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("github");
    expect(labels).toContain("user");
  });

  it("suggests previous() payload fields after dot", () => {
    const suggestions = getSuggestions("previous().", "previous().".length, {
      __previousByDepth: { "1": { image: { version: "1.0.0" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("image");
  });

  it("suggests previous(n) payload fields after dot", () => {
    const suggestions = getSuggestions("previous(2).", "previous(2).".length, {
      __previousByDepth: { "2": { build: { id: "abc" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("build");
  });

  it("suggests nested fields for previous(n).data.", () => {
    const suggestions = getSuggestions("previous(1).data.", "previous(1).data.".length, {
      __previousByDepth: { "1": { data: { image: { tag: "latest" }, sha: "abc" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("image");
    expect(labels).toContain("sha");
  });

  it("suggests root() payload fields inside another function", () => {
    const suggestions = getSuggestions("abs(root().", "abs(root().".length, {
      __root: { value: 42, inner: { ok: true } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("value");
    expect(labels).toContain("inner");
  });

  it("suggests root() payload fields after a complex expression", () => {
    const expression =
      'abs($["node-a"].data.finished_at) && $["node-b"].data.reason || $["node-a"].data.finished_at && root().';
    const suggestions = getSuggestions(expression, expression.length, {
      __root: { github: { ref: "main" }, user: "alice" },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("github");
    expect(labels).toContain("user");
  });

  it("suggests previous() nested fields inside another function", () => {
    const expression = "abs(previous().data.";
    const suggestions = getSuggestions(expression, expression.length, {
      __previousByDepth: { "1": { data: { image: { tag: "latest" }, sha: "abc" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("image");
    expect(labels).toContain("sha");
  });
});

describe("getSuggestions memory namespace", () => {
  it("suggests the memory namespace among top-level completions", () => {
    const suggestions = getSuggestions("m", 1, {});
    const memory = suggestions.find((item) => item.label === "memory");
    expect(memory).toBeDefined();
    expect(memory?.kind).toBe("variable");
    expect(memory?.insertText).toBe("memory.");
  });

  it("suggests memory.find and memory.findFirst after the dot", () => {
    const expression = "memory.";
    const suggestions = getSuggestions(expression, expression.length, {});
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("find");
    expect(labels).toContain("findFirst");
    const findFirst = suggestions.find((item) => item.label === "findFirst");
    expect(findFirst?.kind).toBe("function");
    expect(findFirst?.insertText).toBe('findFirst("${1:namespace}", {${2}})');
  });

  it("filters memory methods by the typed member prefix", () => {
    const expression = "memory.f";
    const suggestions = getSuggestions(expression, expression.length, {});
    const labels = suggestions.map((item) => item.label);
    expect(labels).toEqual(expect.arrayContaining(["find", "findFirst"]));
  });

  it("hides the exact-match memory method from suggestions", () => {
    const expression = "memory.find";
    const suggestions = getSuggestions(expression, expression.length, {});
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("findFirst");
    expect(labels).not.toContain("find");
  });

  it("does not treat an unrelated identifier as a namespace", () => {
    const expression = "notAMemory.";
    const suggestions = getSuggestions(expression, expression.length, {});
    const labels = suggestions.map((item) => item.label);
    expect(labels).not.toContain("find");
    expect(labels).not.toContain("findFirst");
  });
});

describe("getSuggestions config fields", () => {
  it("suggests config fields for $['NodeName'].config.", () => {
    const expression = '$["my-component"].config.';
    const suggestions = getSuggestions(expression, expression.length, {
      "my-component": { status: "ok", config: { url: "https://example.com", timeout: 30 } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("url");
    expect(labels).toContain("timeout");
  });

  it("suggests config as a field for previous()", () => {
    const expression = "previous().";
    const suggestions = getSuggestions(expression, expression.length, {
      __previousByDepth: { "1": { result: "ok", config: { method: "POST" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("result");
    expect(labels).toContain("config");
  });

  it("suggests config nested fields for previous().config.", () => {
    const expression = "previous().config.";
    const suggestions = getSuggestions(expression, expression.length, {
      __previousByDepth: { "1": { result: "ok", config: { method: "POST", endpoint: "/api" } } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("method");
    expect(labels).toContain("endpoint");
  });

  it("suggests config as a field for root()", () => {
    const expression = "root().";
    const suggestions = getSuggestions(expression, expression.length, {
      __root: { user: "alice", config: { source: "github" } },
    });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain("user");
    expect(labels).toContain("config");
  });
});
