import { describe, it, expect } from "vitest";
import { getSuggestions } from "./core";

describe("getSuggestions", () => {
  it("suggests env keys after $ trigger", () => {
    const suggestions = getSuggestions("take($", "take($".length, { foo: 1, bar: 2 });
    const labels = suggestions.map((item) => item.label);
    expect(labels).toContain('["foo"]');
    expect(suggestions.some((item) => item.insertText === '$["foo"]')).toBe(true);
  });

  it("suggests nested run-node keys after a variable dollar trigger", () => {
    const runNodes = { Deploy: { outputs: { url: "https://example.com" } } };
    const globals = { run: { $: runNodes, __runNodes__: runNodes }, sibling: {} };

    const suggestions = getSuggestions("run.$", "run.$".length, globals);
    const labels = suggestions.map((item) => item.label);

    expect(labels).toContain('["Deploy"]');
    expect(labels).not.toContain('["sibling"]');
  });

  it("filters nested run-node keys inside a dollar bracket", () => {
    const runNodes = { Deploy: {}, Test: {} };
    const globals = { run: { $: runNodes, __runNodes__: runNodes }, sibling: {} };

    const suggestions = getSuggestions('run.$["De', 'run.$["De'.length, globals);

    expect(suggestions.map((item) => item.label)).toEqual(['["Deploy"]']);
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

  it("suggests the memory namespace by prefix", () => {
    const suggestions = getSuggestions("mem", "mem".length, {});
    const memorySuggestion = suggestions.find((item) => item.label === "memory");
    expect(memorySuggestion).toBeDefined();
    expect(memorySuggestion?.kind).toBe("variable");
    expect(memorySuggestion?.insertText).toBe("memory.");
  });

  it("suggests memory namespace methods after dot", () => {
    const suggestions = getSuggestions("memory.", "memory.".length, {});
    const findSuggestion = suggestions.find((item) => item.label === "find");
    const findFirstSuggestion = suggestions.find((item) => item.label === "findFirst");

    expect(findSuggestion).toBeDefined();
    expect(findSuggestion?.kind).toBe("function");
    expect(findSuggestion?.example).toBe('memory.find("machines", {"sandbox_id": "12121"})');
    expect(findFirstSuggestion).toBeDefined();
    expect(findFirstSuggestion?.kind).toBe("function");
    expect(findFirstSuggestion?.example).toBe('memory.findFirst("machines", {"creator": "igor"}).sandbox_id');
  });

  it("does not suggest another memory method when the prefix exactly matches one", () => {
    const suggestions = getSuggestions("memory.find", "memory.find".length, {});
    expect(suggestions.map((item) => item.label)).not.toContain("findFirst");
  });

  it("suggests findFirst when the memory method prefix is partial", () => {
    const suggestions = getSuggestions("memory.findF", "memory.findF".length, {});
    expect(suggestions.map((item) => item.label)).toContain("findFirst");
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

describe("getSuggestions top-level globals", () => {
  it("suggests top-level globals by name when enabled", () => {
    const suggestions = getSuggestions(
      "par",
      "par".length,
      { parameters: { message: "hello" } },
      {
        includeTopLevelGlobals: true,
      },
    );
    const parametersSuggestion = suggestions.find((item) => item.label === "parameters");
    expect(parametersSuggestion).toBeDefined();
    expect(parametersSuggestion?.insertText).toBe("parameters.");
  });
});
