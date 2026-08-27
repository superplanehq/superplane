import { describe, expect, it } from "vitest";

import { humanizeLineName } from "./humanizeLineName";

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
