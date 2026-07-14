import { describe, expect, it } from "vitest";
import { integrationResourceDisplayLabel } from "./integrationResourceLabel";

describe("integrationResourceDisplayLabel", () => {
  it("returns trimmed strings", () => {
    expect(integrationResourceDisplayLabel("  claude-opus-4-6  ")).toBe("claude-opus-4-6");
  });

  it("prefers name over id for IntegrationResourceRef objects", () => {
    expect(
      integrationResourceDisplayLabel({
        id: "model-id",
        name: "claude-opus-4-6",
        type: "model",
      }),
    ).toBe("claude-opus-4-6");
  });

  it("falls back to id when name is missing", () => {
    expect(integrationResourceDisplayLabel({ id: "repo-123", type: "repository" })).toBe("repo-123");
  });

  it("returns undefined for empty or unrelated values", () => {
    expect(integrationResourceDisplayLabel("")).toBeUndefined();
    expect(integrationResourceDisplayLabel("   ")).toBeUndefined();
    expect(integrationResourceDisplayLabel(null)).toBeUndefined();
    expect(integrationResourceDisplayLabel(undefined)).toBeUndefined();
    expect(integrationResourceDisplayLabel(42)).toBeUndefined();
    expect(integrationResourceDisplayLabel([])).toBeUndefined();
    expect(integrationResourceDisplayLabel({ type: "model" })).toBeUndefined();
  });
});
