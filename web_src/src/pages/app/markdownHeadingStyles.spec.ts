import { describe, expect, it } from "vitest";

import { MARKDOWN_HEADING_MARGIN_CLASSES, markdownHeadingClassName } from "./markdownHeadingStyles";

describe("markdownHeadingClassName", () => {
  it("uses shared my-4 vertical margin for all heading levels", () => {
    expect(markdownHeadingClassName("h1")).toContain(MARKDOWN_HEADING_MARGIN_CLASSES);
    expect(markdownHeadingClassName("h2")).toContain("my-4");
    expect(markdownHeadingClassName("h3")).toContain("first:mt-0");
  });
});
