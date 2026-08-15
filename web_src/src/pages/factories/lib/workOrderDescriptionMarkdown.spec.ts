import { describe, expect, it } from "vitest";

import { looksLikeMarkdown } from "./workOrderDescriptionMarkdown";

describe("looksLikeMarkdown", () => {
  it("does not treat glob stars as markdown", () => {
    expect(looksLikeMarkdown("**/*.ts")).toBe(false);
  });

  it("does not treat dunder names as markdown", () => {
    expect(looksLikeMarkdown("__init__.py")).toBe(false);
    expect(looksLikeMarkdown("print(__name__)")).toBe(false);
  });

  it("does not treat increment operators as underline markdown", () => {
    expect(looksLikeMarkdown("i++ then j++")).toBe(false);
    expect(looksLikeMarkdown("Use C++ for this")).toBe(false);
  });

  it("treats paired emphasis and headings as markdown", () => {
    expect(looksLikeMarkdown("this is **bold** text")).toBe(true);
    expect(looksLikeMarkdown("this is __important__ now")).toBe(true);
    expect(looksLikeMarkdown("this is <u>underlined</u> text")).toBe(true);
    expect(looksLikeMarkdown("## Title")).toBe(true);
  });

  it("does not treat ++ pairs as underline markdown", () => {
    expect(looksLikeMarkdown("this is ++underlined++ text")).toBe(false);
    expect(looksLikeMarkdown("++word++.")).toBe(false);
  });
});
