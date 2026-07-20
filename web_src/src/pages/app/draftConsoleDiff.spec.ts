import { describe, expect, it } from "vitest";

import {
  buildDraftConsoleDiffSummary,
  getDraftConsoleDiffCounts,
  hasDraftVersusLiveConsoleDiff,
} from "./draftConsoleDiff";

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

  it("ignores content key ordering between committed and staged serializations", () => {
    // The committed console is serialized by the backend, whose YAML encoder
    // marshals panel `content` map keys alphabetically. The staged/effective
    // console keeps the editor's insertion order. They are semantically
    // identical, so the diff must be false — otherwise the "UNCOMMITTED
    // CHANGES" badge sticks after a commit until a full refresh re-fetches
    // both snapshots from the backend in matching order.
    const committed = {
      panels: [
        { id: "fgfggd", type: "html", content: { body: "aaa", title: "fgfggd" } },
        { id: "aaa", type: "node", content: { node: "start", showRun: false, title: "aaa" } },
      ],
      layout: [
        { i: "fgfggd", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 },
        { i: "aaa", x: 0, y: 6, w: 12, h: 6, minW: 2, minH: 2 },
      ],
    };
    const staged = {
      panels: [
        { id: "fgfggd", type: "html", content: { title: "fgfggd", body: "aaa" } },
        { id: "aaa", type: "node", content: { title: "aaa", node: "start", showRun: false } },
      ],
      layout: [
        { i: "fgfggd", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 },
        { i: "aaa", x: 0, y: 6, w: 12, h: 6, minW: 2, minH: 2 },
      ],
    };

    expect(hasDraftVersusLiveConsoleDiff(committed, staged)).toBe(false);
  });

  it("ignores nested content key ordering (variable sources)", () => {
    const committed = {
      panels: [
        {
          id: "p1",
          type: "markdown",
          content: {
            body: "{{ x }}",
            variables: [{ name: "x", source: { kind: "memory", namespace: "ns" } }],
          },
        },
      ],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };
    const staged = {
      panels: [
        {
          id: "p1",
          type: "markdown",
          content: {
            variables: [{ name: "x", source: { namespace: "ns", kind: "memory" } }],
            body: "{{ x }}",
          },
        },
      ],
      layout: [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    };

    expect(hasDraftVersusLiveConsoleDiff(committed, staged)).toBe(false);
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

describe("buildDraftConsoleDiffSummary", () => {
  it("returns per-panel diff items for added, updated, and removed panels", () => {
    const live = {
      panels: [
        { id: "updated", type: "markdown", content: { title: "Runbook", body: "before" } },
        { id: "removed", type: "markdown", content: { title: "Old", body: "remove me" } },
      ],
      layout: [
        { i: "updated", x: 0, y: 0, w: 4, h: 2 },
        { i: "removed", x: 0, y: 2, w: 4, h: 2 },
      ],
    };
    const draft = {
      panels: [
        { id: "updated", type: "markdown", content: { title: "Runbook", body: "after" } },
        { id: "added", type: "markdown", content: { title: "New", body: "add me" } },
      ],
      layout: [
        { i: "updated", x: 0, y: 0, w: 4, h: 3 },
        { i: "added", x: 0, y: 2, w: 4, h: 2 },
      ],
    };

    const summary = buildDraftConsoleDiffSummary(live, draft);

    expect(summary.addedCount).toBe(1);
    expect(summary.updatedCount).toBe(1);
    expect(summary.removedCount).toBe(1);
    expect(summary.items.map((item) => [item.id, item.changeType, item.title])).toEqual([
      ["added", "added", "New"],
      ["removed", "removed", "Old"],
      ["updated", "updated", "Runbook"],
    ]);
    expect(summary.items.find((item) => item.id === "updated")?.lines).toEqual(
      expect.arrayContaining([
        { prefix: "-", text: "content:" },
        { prefix: "+", text: "content:" },
        { prefix: "-", text: "layout:" },
        { prefix: "+", text: "layout:" },
      ]),
    );
  });

  it("marks layout-only panel changes as updated", () => {
    const live = {
      panels: [{ id: "panel-1", type: "markdown", content: { body: "same" } }],
      layout: [{ i: "panel-1", x: 0, y: 0, w: 4, h: 2 }],
    };
    const draft = {
      panels: [{ id: "panel-1", type: "markdown", content: { body: "same" } }],
      layout: [{ i: "panel-1", x: 6, y: 0, w: 4, h: 2 }],
    };

    const summary = buildDraftConsoleDiffSummary(live, draft);

    expect(summary.items).toHaveLength(1);
    expect(summary.items[0].changeType).toBe("updated");
    expect(summary.items[0].lines).toEqual(
      expect.arrayContaining([
        { prefix: "-", text: "layout:" },
        { prefix: "+", text: "layout:" },
      ]),
    );
  });
});
