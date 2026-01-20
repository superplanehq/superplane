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
});
