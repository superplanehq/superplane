import { describe, expect, it } from "vitest";

import {
  buildLatestArtifactDataById,
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  extractArtifactUrl,
  extractPrArtifactState,
  formatPrArtifactLabel,
  overlayLiveArtifactData,
} from "./workOrderArtifact";

describe("formatPrArtifactLabel", () => {
  it("returns #<number> when data.number is set (backend convention)", () => {
    expect(formatPrArtifactLabel({ number: 1234 })).toBe("#1234");
    expect(formatPrArtifactLabel({ number: "1234" })).toBe("#1234");
  });

  it("falls back to data.prNumber when data.number is absent", () => {
    expect(formatPrArtifactLabel({ prNumber: 42 })).toBe("#42");
  });

  it("prefers data.number over data.prNumber when both are set", () => {
    expect(formatPrArtifactLabel({ number: 1, prNumber: 2 })).toBe("#1");
  });

  it("normalizes a leading '#' so callers don't render '##'", () => {
    expect(formatPrArtifactLabel({ number: "#42" })).toBe("#42");
    expect(formatPrArtifactLabel({ prNumber: "#7" })).toBe("#7");
  });

  it("returns undefined when no known key carries a usable value", () => {
    expect(formatPrArtifactLabel(undefined)).toBeUndefined();
    expect(formatPrArtifactLabel({})).toBeUndefined();
    expect(formatPrArtifactLabel({ number: "" })).toBeUndefined();
    expect(formatPrArtifactLabel({ number: "   " })).toBeUndefined();
    expect(formatPrArtifactLabel({ number: null })).toBeUndefined();
    expect(formatPrArtifactLabel({ prNumber: "" })).toBeUndefined();
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

describe("extractArtifactUrl", () => {
  it("returns data.url when present", () => {
    expect(extractArtifactUrl({ url: "https://example.com/pr/1" })).toBe("https://example.com/pr/1");
  });

  it("returns undefined for missing / blank / non-string urls", () => {
    expect(extractArtifactUrl(undefined)).toBeUndefined();
    expect(extractArtifactUrl({})).toBeUndefined();
    expect(extractArtifactUrl({ url: "" })).toBeUndefined();
    expect(extractArtifactUrl({ url: "   " })).toBeUndefined();
    expect(extractArtifactUrl({ url: null })).toBeUndefined();
  });
});

describe("extractArtifactTitle", () => {
  it("returns data.title when present", () => {
    expect(extractArtifactTitle({ title: "Draft PR" })).toBe("Draft PR");
  });

  it("returns undefined for missing / blank titles", () => {
    expect(extractArtifactTitle(undefined)).toBeUndefined();
    expect(extractArtifactTitle({})).toBeUndefined();
    expect(extractArtifactTitle({ title: "" })).toBeUndefined();
    expect(extractArtifactTitle({ title: "  " })).toBeUndefined();
  });
});

describe("extractArtifactName", () => {
  it("returns data.name when present", () => {
    expect(extractArtifactName({ name: "feature/refund-retry" })).toBe("feature/refund-retry");
  });

  it("returns undefined for missing / blank names", () => {
    expect(extractArtifactName(undefined)).toBeUndefined();
    expect(extractArtifactName({})).toBeUndefined();
    expect(extractArtifactName({ name: "" })).toBeUndefined();
    expect(extractArtifactName({ name: "  " })).toBeUndefined();
  });
});

describe("extractPrArtifactState", () => {
  it("returns each known state as-is", () => {
    expect(extractPrArtifactState({ state: "open" })).toBe("open");
    expect(extractPrArtifactState({ state: "draft" })).toBe("draft");
    expect(extractPrArtifactState({ state: "closed" })).toBe("closed");
    expect(extractPrArtifactState({ state: "merged" })).toBe("merged");
  });

  it("normalizes case", () => {
    expect(extractPrArtifactState({ state: "MERGED" })).toBe("merged");
    expect(extractPrArtifactState({ state: "Draft" })).toBe("draft");
  });

  it("returns undefined for missing, blank, or unrecognized values (back-compat default)", () => {
    expect(extractPrArtifactState(undefined)).toBeUndefined();
    expect(extractPrArtifactState({})).toBeUndefined();
    expect(extractPrArtifactState({ state: "" })).toBeUndefined();
    expect(extractPrArtifactState({ state: "in_review" })).toBeUndefined();
  });
});

describe("buildLatestArtifactDataById", () => {
  it("indexes artifacts that have both an id and data", () => {
    const byId = buildLatestArtifactDataById([
      { id: "art-1", data: { state: "merged" } },
      { id: "art-2" },
      { data: { state: "open" } },
    ]);

    expect(byId.get("art-1")).toEqual({ state: "merged" });
    expect(byId.has("art-2")).toBe(false);
  });
});

describe("overlayLiveArtifactData", () => {
  it("replaces snapshot data when a matching live row exists", () => {
    const snapshot = { id: "art-1", type: "pr", data: { state: "open" } };
    const live = overlayLiveArtifactData(snapshot, new Map([["art-1", { state: "merged", title: "Done" }]]));

    expect(live).toEqual({ id: "art-1", type: "pr", data: { state: "merged", title: "Done" } });
  });

  it("keeps the snapshot when the artifact is missing from the live list", () => {
    const snapshot = { id: "art-1", data: { state: "open" } };
    expect(overlayLiveArtifactData(snapshot, new Map())).toBe(snapshot);
  });
});
