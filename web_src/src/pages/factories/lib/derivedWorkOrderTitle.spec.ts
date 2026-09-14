import { describe, expect, it } from "bun:test";

import { derivedWorkOrderTitle } from "./derivedWorkOrderTitle";

describe("derivedWorkOrderTitle", () => {
  it("uses the first non-empty line", () => {
    expect(derivedWorkOrderTitle("Refunds fail on retry.\n\nMore context.")).toBe("Refunds fail on retry.");
  });

  it("skips image-only lines", () => {
    expect(derivedWorkOrderTitle("![shot](sp-file://abc)\nFix the checkout retry.")).toBe("Fix the checkout retry.");
  });

  it("strips a leading markdown heading", () => {
    expect(derivedWorkOrderTitle("## Refunds fail on retry.")).toBe("Refunds fail on retry.");
  });

  it("returns an empty string when there is no text", () => {
    expect(derivedWorkOrderTitle("")).toBe("");
    expect(derivedWorkOrderTitle("![shot](sp-file://abc)")).toBe("");
  });
});
