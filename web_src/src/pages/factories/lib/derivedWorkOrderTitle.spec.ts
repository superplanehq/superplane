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

  it("strips paired bold marks from the first line", () => {
    expect(derivedWorkOrderTitle("**Something**\n\nMore context.")).toBe("Something");
  });

  it("strips paired emphasis inside mixed text", () => {
    expect(derivedWorkOrderTitle("**Something** happens")).toBe("Something happens");
  });

  it("strips paired underscore bold and italic marks", () => {
    expect(derivedWorkOrderTitle("__bold__")).toBe("bold");
    expect(derivedWorkOrderTitle("*italic*")).toBe("italic");
  });

  it("unwraps nested emphasis wrappers", () => {
    expect(derivedWorkOrderTitle("***nested***")).toBe("nested");
  });

  it("leaves unmatched emphasis marks in place", () => {
    expect(derivedWorkOrderTitle("**Something")).toBe("**Something");
  });

  it("keeps spaced asterisks that are not emphasis", () => {
    expect(derivedWorkOrderTitle("5 * 3 * 2")).toBe("5 * 3 * 2");
  });

  it("keeps emphasis marks inside inline code", () => {
    expect(derivedWorkOrderTitle("`retry*now*`")).toBe("`retry*now*`");
  });

  it("keeps escaped emphasis marks", () => {
    expect(derivedWorkOrderTitle("\\*not italic\\*")).toBe("\\*not italic\\*");
  });

  it("strips emphasis that wraps inline code", () => {
    expect(derivedWorkOrderTitle("**use `retry*now*` please**")).toBe("use `retry*now*` please");
  });

  it("keeps underscores between word characters", () => {
    expect(derivedWorkOrderTitle("foo_bar")).toBe("foo_bar");
  });

  it("strips a heading and then paired emphasis", () => {
    expect(derivedWorkOrderTitle("## **Refunds fail on retry.**")).toBe("Refunds fail on retry.");
  });
});
