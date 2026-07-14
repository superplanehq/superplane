import { describe, expect, it } from "vitest";
import { resourceLabel } from "./utils";

describe("resourceLabel", () => {
  it("returns strings unchanged", () => {
    expect(resourceLabel("claude-opus-4")).toBe("claude-opus-4");
  });

  it("stringifies numbers and booleans", () => {
    expect(resourceLabel(42)).toBe("42");
    expect(resourceLabel(true)).toBe("true");
  });

  it("returns undefined for empty-ish values", () => {
    expect(resourceLabel(undefined)).toBeUndefined();
    expect(resourceLabel(null)).toBeUndefined();
    expect(resourceLabel("")).toBeUndefined();
  });

  it("resolves an integration-resource reference to its name", () => {
    expect(resourceLabel({ id: "m_1", name: "claude-opus-4", type: "model" })).toBe("claude-opus-4");
  });

  it("falls back to the id when there is no name", () => {
    expect(resourceLabel({ id: "m_1", type: "model" })).toBe("m_1");
  });

  it("returns undefined for objects without name or id", () => {
    expect(resourceLabel({ type: "model" })).toBeUndefined();
  });
});
