import { describe, expect, it } from "vitest";

import { getDraftConsoleDiffCounts, hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";

describe("hasDraftVersusLiveConsoleDiff", () => {
  it("returns false when both consoles are empty", () => {
    expect(hasDraftVersusLiveConsoleDiff(undefined, undefined)).toBe(false);
    expect(hasDraftVersusLiveConsoleDiff({ panels: [], layout: [] }, { panels: [], layout: [] })).toBe(false);
  });

  it("returns true when draft adds a panel", () => {
    const live = { panels: [], layout: [] };
    const draft = {
      panels: [{ id: "p1", type: "markdown", content: { body: "hello" } }],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("returns true when panel content changes", () => {
    const live = {
      panels: [{ id: "p1", type: "markdown", content: { body: "before" } }],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };
    const draft = {
      panels: [{ id: "p1", type: "markdown", content: { body: "after" } }],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("returns false when consoles match", () => {
    const console = {
      panels: [{ id: "p1", type: "markdown", content: { body: "same" } }],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };

    expect(hasDraftVersusLiveConsoleDiff(console, console)).toBe(false);
  });
});

describe("getDraftConsoleDiffCounts", () => {
  it("counts added, updated, and removed console items", () => {
    const live = {
      panels: [
        { id: "updated", type: "markdown", content: { body: "before" } },
        { id: "removed", type: "markdown", content: { body: "remove me" } },
      ],
      layout: [
        { i: "updated", x: 0, y: 0, w: 4, h: 2 },
        { i: "removed", x: 0, y: 2, w: 4, h: 2 },
      ],
    };
    const draft = {
      panels: [
        { id: "updated", type: "markdown", content: { body: "after" } },
        { id: "added", type: "markdown", content: { body: "add me" } },
      ],
      layout: [
        { i: "updated", x: 0, y: 0, w: 4, h: 3 },
        { i: "added", x: 0, y: 2, w: 4, h: 2 },
      ],
    };

    expect(getDraftConsoleDiffCounts(live, draft)).toEqual({ added: 1, updated: 1, removed: 1 });
  });
});
