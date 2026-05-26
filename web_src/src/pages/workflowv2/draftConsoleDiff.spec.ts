import { describe, expect, it } from "vitest";

import { hasDraftVersusLiveConsoleDiff } from "./draftConsoleDiff";

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
