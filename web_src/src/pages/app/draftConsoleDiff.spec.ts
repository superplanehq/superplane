import { describe, expect, it } from "vitest";

import type { ConsoleLayoutItem, ConsolePage, ConsolePanel } from "@/hooks/useCanvasData";

import {
  buildDraftConsoleDiffSummary,
  getDraftConsoleDiffCounts,
  hasDraftVersusLiveConsoleDiff,
} from "./draftConsoleDiff";

// Every console the diff engine sees is normalized into the page shape.
// These specs wrap flat panels/layout into a single implicit `main` page
// so the assertions stay focused on panel-level behavior.
function singlePage(panels: ConsolePanel[], layout: ConsoleLayoutItem[]): { pages: ConsolePage[] } {
  return { pages: [{ id: "main", panels, layout }] };
}

describe("hasDraftVersusLiveConsoleDiff", () => {
  it("returns false when both consoles are empty", () => {
    expect(hasDraftVersusLiveConsoleDiff(undefined, undefined)).toBe(false);
    expect(hasDraftVersusLiveConsoleDiff(singlePage([], []), singlePage([], []))).toBe(false);
  });

  it("returns true when draft adds a panel", () => {
    const live = singlePage([], []);
    const draft = singlePage(
      [{ id: "p1", type: "markdown", content: { body: "hello" } }],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("returns true when panel content changes", () => {
    const live = singlePage(
      [{ id: "p1", type: "markdown", content: { body: "before" } }],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );
    const draft = singlePage(
      [{ id: "p1", type: "markdown", content: { body: "after" } }],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("detects the addition of an empty page (page metadata only)", () => {
    // Adding an empty second page does not touch any panel or layout,
    // so a panel-only comparison would return false. The comparison
    // must include page-level metadata (id, name, order) so page-only
    // edits still register — otherwise a staged `console.yaml` write
    // is silently discarded.
    const live: { pages: ConsolePage[] } = {
      pages: [{ id: "main", name: "Main", panels: [], layout: [] }],
    };
    const draft: { pages: ConsolePage[] } = {
      pages: [
        { id: "main", name: "Main", panels: [], layout: [] },
        { id: "details", name: "Details", panels: [], layout: [] },
      ],
    };

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("detects a page rename that leaves panels untouched", () => {
    const live: { pages: ConsolePage[] } = {
      pages: [
        { id: "main", name: "Main", panels: [], layout: [] },
        { id: "details", name: "Details", panels: [], layout: [] },
      ],
    };
    const draft: { pages: ConsolePage[] } = {
      pages: [
        { id: "main", name: "Main", panels: [], layout: [] },
        { id: "details", name: "Renamed", panels: [], layout: [] },
      ],
    };

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("detects a page reorder that keeps every panel intact", () => {
    const live: { pages: ConsolePage[] } = {
      pages: [
        { id: "main", name: "Main", panels: [], layout: [] },
        { id: "details", name: "Details", panels: [], layout: [] },
      ],
    };
    const draft: { pages: ConsolePage[] } = {
      pages: [
        { id: "details", name: "Details", panels: [], layout: [] },
        { id: "main", name: "Main", panels: [], layout: [] },
      ],
    };

    expect(hasDraftVersusLiveConsoleDiff(live, draft)).toBe(true);
  });

  it("treats a sole empty default page as equivalent to zero pages", () => {
    // Committed legacy YAML with no panels parses back to zero pages.
    // The local editor keeps a `main`/`Main` page in state so the grid
    // has somewhere to render into. The two must compare equal so the
    // header does not show a phantom "UNCOMMITTED CHANGES" badge and
    // autosave does not fire against a byte-identical export.
    const committed: { pages: ConsolePage[] } = { pages: [] };
    const local: { pages: ConsolePage[] } = {
      pages: [{ id: "main", name: "Main", panels: [], layout: [] }],
    };

    expect(hasDraftVersusLiveConsoleDiff(committed, local)).toBe(false);
  });

  it("still detects a truly non-default single empty page", () => {
    // A renamed sole page IS a real edit — it changes the on-disk YAML.
    const committed: { pages: ConsolePage[] } = { pages: [] };
    const local: { pages: ConsolePage[] } = {
      pages: [{ id: "main", name: "Renamed", panels: [], layout: [] }],
    };

    expect(hasDraftVersusLiveConsoleDiff(committed, local)).toBe(true);
  });

  it("returns false when consoles match", () => {
    const c = singlePage(
      [{ id: "p1", type: "markdown", content: { body: "same" } }],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );

    expect(hasDraftVersusLiveConsoleDiff(c, c)).toBe(false);
  });

  it("ignores content key ordering between committed and staged serializations", () => {
    // The committed console is serialized by the backend, whose YAML encoder
    // marshals panel `content` map keys alphabetically. The staged/effective
    // console keeps the editor's insertion order. They are semantically
    // identical, so the diff must be false — otherwise the "UNCOMMITTED
    // CHANGES" badge sticks after a commit until a full refresh re-fetches
    // both snapshots from the backend in matching order.
    const committed = singlePage(
      [
        { id: "fgfggd", type: "html", content: { body: "aaa", title: "fgfggd" } },
        { id: "aaa", type: "node", content: { node: "start", showRun: false, title: "aaa" } },
      ],
      [
        { i: "fgfggd", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 },
        { i: "aaa", x: 0, y: 6, w: 12, h: 6, minW: 2, minH: 2 },
      ],
    );
    const staged = singlePage(
      [
        { id: "fgfggd", type: "html", content: { title: "fgfggd", body: "aaa" } },
        { id: "aaa", type: "node", content: { title: "aaa", node: "start", showRun: false } },
      ],
      [
        { i: "fgfggd", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 },
        { i: "aaa", x: 0, y: 6, w: 12, h: 6, minW: 2, minH: 2 },
      ],
    );

    expect(hasDraftVersusLiveConsoleDiff(committed, staged)).toBe(false);
  });

  it("ignores nested content key ordering (variable sources)", () => {
    const committed = singlePage(
      [
        {
          id: "p1",
          type: "markdown",
          content: {
            body: "{{ x }}",
            variables: [{ name: "x", source: { kind: "memory", namespace: "ns" } }],
          },
        },
      ],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );
    const staged = singlePage(
      [
        {
          id: "p1",
          type: "markdown",
          content: {
            variables: [{ name: "x", source: { namespace: "ns", kind: "memory" } }],
            body: "{{ x }}",
          },
        },
      ],
      [{ i: "p1", x: 0, y: 0, w: 4, h: 2 }],
    );

    expect(hasDraftVersusLiveConsoleDiff(committed, staged)).toBe(false);
  });
});

describe("getDraftConsoleDiffCounts", () => {
  it("counts added, updated, and removed console items", () => {
    const live = singlePage(
      [
        { id: "updated", type: "markdown", content: { body: "before" } },
        { id: "removed", type: "markdown", content: { body: "remove me" } },
      ],
      [
        { i: "updated", x: 0, y: 0, w: 4, h: 2 },
        { i: "removed", x: 0, y: 2, w: 4, h: 2 },
      ],
    );
    const draft = singlePage(
      [
        { id: "updated", type: "markdown", content: { body: "after" } },
        { id: "added", type: "markdown", content: { body: "add me" } },
      ],
      [
        { i: "updated", x: 0, y: 0, w: 4, h: 3 },
        { i: "added", x: 0, y: 2, w: 4, h: 2 },
      ],
    );

    expect(getDraftConsoleDiffCounts(live, draft)).toEqual({ added: 1, updated: 1, removed: 1 });
  });
});

describe("buildDraftConsoleDiffSummary", () => {
  it("returns per-panel diff items for added, updated, and removed panels", () => {
    const live = singlePage(
      [
        { id: "updated", type: "markdown", content: { title: "Runbook", body: "before" } },
        { id: "removed", type: "markdown", content: { title: "Old", body: "remove me" } },
      ],
      [
        { i: "updated", x: 0, y: 0, w: 4, h: 2 },
        { i: "removed", x: 0, y: 2, w: 4, h: 2 },
      ],
    );
    const draft = singlePage(
      [
        { id: "updated", type: "markdown", content: { title: "Runbook", body: "after" } },
        { id: "added", type: "markdown", content: { title: "New", body: "add me" } },
      ],
      [
        { i: "updated", x: 0, y: 0, w: 4, h: 3 },
        { i: "added", x: 0, y: 2, w: 4, h: 2 },
      ],
    );

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

  it("does not collide when two pages have panels with the same id", () => {
    // Two different pages each hold a panel called `overview`. The diff
    // engine must treat them as separate panels — a change on the
    // Overview page must not silently overwrite / merge with the panel
    // on the Details page.
    const live: { pages: ConsolePage[] } = {
      pages: [
        {
          id: "overview",
          panels: [{ id: "shared", type: "markdown", content: { body: "overview text" } }],
          layout: [{ i: "shared", x: 0, y: 0, w: 4, h: 2 }],
        },
        {
          id: "details",
          panels: [{ id: "shared", type: "markdown", content: { body: "details text" } }],
          layout: [{ i: "shared", x: 0, y: 0, w: 4, h: 2 }],
        },
      ],
    };
    const draft: { pages: ConsolePage[] } = {
      pages: [
        {
          id: "overview",
          panels: [{ id: "shared", type: "markdown", content: { body: "overview text CHANGED" } }],
          layout: [{ i: "shared", x: 0, y: 0, w: 4, h: 2 }],
        },
        {
          id: "details",
          panels: [{ id: "shared", type: "markdown", content: { body: "details text" } }],
          layout: [{ i: "shared", x: 0, y: 0, w: 4, h: 2 }],
        },
      ],
    };

    const summary = buildDraftConsoleDiffSummary(live, draft);
    expect(summary.items).toHaveLength(1);
    expect(summary.items[0].changeType).toBe("updated");
    // Every returned item carries its page id so consumers that key by
    // panel id (e.g. `ConsoleGrid.itemsById`) can filter to the active
    // tab and avoid cross-page id collisions.
    expect(summary.items[0].pageId).toBe("overview");
    // The diff header must scope the changed panel to the page so
    // reviewers can tell which of the two `shared` panels moved.
    const headerLine = summary.items[0].lines.find((line) => line.text.includes("diff --git"));
    expect(headerLine?.text).toContain("console/pages/overview/panels/shared.yaml");
  });

  it("treats a panel moved between pages as remove + add, not a silent update", () => {
    const live: { pages: ConsolePage[] } = {
      pages: [
        {
          id: "overview",
          panels: [{ id: "intro", type: "markdown", content: { body: "hi" } }],
          layout: [{ i: "intro", x: 0, y: 0, w: 4, h: 2 }],
        },
        { id: "details", panels: [], layout: [] },
      ],
    };
    const draft: { pages: ConsolePage[] } = {
      pages: [
        { id: "overview", panels: [], layout: [] },
        {
          id: "details",
          panels: [{ id: "intro", type: "markdown", content: { body: "hi" } }],
          layout: [{ i: "intro", x: 0, y: 0, w: 4, h: 2 }],
        },
      ],
    };

    const summary = buildDraftConsoleDiffSummary(live, draft);
    const changeTypes = summary.items.map((item) => item.changeType).sort();
    expect(changeTypes).toEqual(["added", "removed"]);
    expect(summary.addedCount).toBe(1);
    expect(summary.removedCount).toBe(1);
    expect(summary.updatedCount).toBe(0);
  });

  it("marks layout-only panel changes as updated", () => {
    const live = singlePage(
      [{ id: "panel-1", type: "markdown", content: { body: "same" } }],
      [{ i: "panel-1", x: 0, y: 0, w: 4, h: 2 }],
    );
    const draft = singlePage(
      [{ id: "panel-1", type: "markdown", content: { body: "same" } }],
      [{ i: "panel-1", x: 6, y: 0, w: 4, h: 2 }],
    );

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
