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
  it("joins automation names as a flow", () => {
    expect(
      formatLinePhaseDescription(
        [{ app: { app: "app-plan" } }, { app: { app: "app-implement" } }, { app: { app: "app-verify" } }],
        [
          { id: "app-plan", name: "Refund Planner" },
          { id: "app-implement", name: "Refund Implementer" },
          { id: "app-verify", name: "Refund Verifier" },
        ],
      ),
    ).toBe("Refund Planner → Refund Implementer → Refund Verifier");
  });

  it("returns undefined when there are no named phases", () => {
    expect(formatLinePhaseDescription([], [])).toBeUndefined();
    expect(formatLinePhaseDescription(undefined, undefined)).toBeUndefined();
    expect(formatLinePhaseDescription([{ app: { app: "missing" } }], [])).toBeUndefined();
  });
});
