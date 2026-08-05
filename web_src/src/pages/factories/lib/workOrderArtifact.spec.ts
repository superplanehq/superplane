import { describe, expect, it } from "vitest";

import { extractArtifactMarkdownBody, formatPrArtifactLabel } from "./workOrderArtifact";

describe("formatPrArtifactLabel", () => {
  it("returns #<number> when data.prNumber is a numeric value", () => {
    expect(formatPrArtifactLabel({ prNumber: 1234 })).toBe("#1234");
  });

  it("returns #<number> when data.prNumber is a numeric string", () => {
    expect(formatPrArtifactLabel({ prNumber: "1234" })).toBe("#1234");
  });

  it("normalizes a leading '#' so callers don't render '##'", () => {
    expect(formatPrArtifactLabel({ prNumber: "#42" })).toBe("#42");
  });

  it("returns undefined when data.prNumber is missing or empty", () => {
    expect(formatPrArtifactLabel(undefined)).toBeUndefined();
    expect(formatPrArtifactLabel({})).toBeUndefined();
    expect(formatPrArtifactLabel({ prNumber: "" })).toBeUndefined();
    expect(formatPrArtifactLabel({ prNumber: "   " })).toBeUndefined();
    expect(formatPrArtifactLabel({ prNumber: null })).toBeUndefined();
  });
});

describe("extractArtifactMarkdownBody", () => {
  it("returns data.body when present", () => {
    expect(extractArtifactMarkdownBody({ body: "note content" })).toBe("note content");
  });

  it("returns undefined for missing / blank / non-string bodies", () => {
    expect(extractArtifactMarkdownBody(undefined)).toBeUndefined();
    expect(extractArtifactMarkdownBody({})).toBeUndefined();
    expect(extractArtifactMarkdownBody({ body: "" })).toBeUndefined();
    expect(extractArtifactMarkdownBody({ body: 123 })).toBe("123");
  });
});
