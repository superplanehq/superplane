import { describe, expect, it } from "vitest";

import { formatLinePhaseDescription, humanizeLineName } from "./humanizeLineName";

describe("humanizeLineName", () => {
  it("turns kebab-case into title case and keeps connectors lowercase", () => {
    expect(humanizeLineName("plan-and-implement")).toBe("Plan and Implement");
  });

  it("handles underscores and mixed casing", () => {
    expect(humanizeLineName("HOTFIX_VERIFY")).toBe("Hotfix Verify");
  });

  it("falls back when the name is empty", () => {
    expect(humanizeLineName("")).toBe("Unnamed line");
    expect(humanizeLineName(undefined)).toBe("Unnamed line");
  });
});

describe("formatLinePhaseDescription", () => {
  it("joins humanized phase names as a flow", () => {
    expect(formatLinePhaseDescription([{ name: "plan" }, { name: "implement" }, { name: "verify" }])).toBe(
      "Plan → Implement → Verify",
    );
  });

  it("returns undefined when there are no named phases", () => {
    expect(formatLinePhaseDescription([])).toBeUndefined();
    expect(formatLinePhaseDescription(undefined)).toBeUndefined();
  });
});
