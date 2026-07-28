import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ConsolePage } from "@/hooks/useCanvasData";

import { useConsolePagesState } from "./useConsolePagesState";

const PAGE: ConsolePage = {
  id: "main",
  name: "Main",
  panels: [
    {
      id: "deploy",
      type: "node",
      content: { title: "Deploy", node: "deploy-prod", showRun: true },
    },
  ],
  layout: [{ i: "deploy", x: 0, y: 0, w: 12, h: 6, minW: 2, minH: 2 }],
};

describe("useConsolePagesState — legacy `node` panel migration", () => {
  it("rewrites the type to `nodes` when the merged card emits a nodes-list body", () => {
    const onChange = vi.fn();
    const onEffectiveChange = vi.fn();
    const onActivePageIdChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: [PAGE],
          onChange,
          onEffectiveChange,
          activePageId: "main",
          onActivePageIdChange,
        }),
      );

      act(() => {
        result.current.handlePanelContentChange("deploy", {
          title: "Deploy",
          nodes: [{ node: "deploy-prod", showRun: true }],
        });
      });

      const migrated = result.current.activePanels[0];
      expect(migrated.type).toBe("nodes");
      expect(migrated.content).toEqual({
        title: "Deploy",
        nodes: [{ node: "deploy-prod", showRun: true }],
      });

      act(() => {
        vi.runAllTimers();
      });

      expect(onChange).toHaveBeenCalledTimes(1);
      const [saved] = onChange.mock.calls[0];
      expect(saved.pages[0].panels[0].type).toBe("nodes");
    } finally {
      vi.useRealTimers();
    }
  });

  it("leaves the panel type untouched when content shape does not match", () => {
    const onChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: [PAGE],
          onChange,
          activePageId: "main",
          onActivePageIdChange: vi.fn(),
        }),
      );

      act(() => {
        result.current.handlePanelContentChange("deploy", { title: "Renamed", node: "deploy-prod" });
      });

      expect(result.current.activePanels[0].type).toBe("node");
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("useConsolePagesState — page management", () => {
  const twoPages: ConsolePage[] = [
    { id: "overview", name: "Overview", panels: [], layout: [] },
    { id: "details", name: "Details", panels: [], layout: [] },
  ];

  it("adds a new page and switches to it", () => {
    const onChange = vi.fn();
    const onActivePageIdChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: twoPages,
          onChange,
          activePageId: "overview",
          onActivePageIdChange,
        }),
      );

      act(() => {
        result.current.handleAddPage();
      });

      expect(result.current.localPages).toHaveLength(3);
      expect(onActivePageIdChange).toHaveBeenCalledWith(result.current.localPages[2]!.id);
    } finally {
      vi.useRealTimers();
    }
  });

  it("refuses to add a page past MAX_CONSOLE_PAGES", async () => {
    // Rapid successive clicks / direct calls should not drift local
    // state past the cap. The tab strip disables the button in the UI
    // but that guard only fires on the next render.
    const { MAX_CONSOLE_PAGES } = await import("./consoleYaml");
    const atCap: ConsolePage[] = Array.from({ length: MAX_CONSOLE_PAGES }, (_v, i) => ({
      id: `page-${i}`,
      name: `Page ${i}`,
      panels: [],
      layout: [],
    }));

    const onChange = vi.fn();
    const onActivePageIdChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: atCap,
          onChange,
          activePageId: atCap[0]!.id,
          onActivePageIdChange,
        }),
      );

      act(() => {
        result.current.handleAddPage();
        result.current.handleAddPage();
      });

      expect(result.current.localPages).toHaveLength(MAX_CONSOLE_PAGES);
      expect(onActivePageIdChange).not.toHaveBeenCalled();
    } finally {
      vi.useRealTimers();
    }
  });

  it("renames a page in place", () => {
    const onChange = vi.fn();
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: twoPages,
          onChange,
          activePageId: "overview",
          onActivePageIdChange: vi.fn(),
        }),
      );

      act(() => {
        result.current.handleRenamePage("overview", "New name");
      });

      expect(result.current.localPages[0]!.name).toBe("New name");
    } finally {
      vi.useRealTimers();
    }
  });

  it("removes a page and picks the previous one as active", () => {
    const onChange = vi.fn();
    const onActivePageIdChange = vi.fn();
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: twoPages,
          onChange,
          activePageId: "details",
          onActivePageIdChange,
        }),
      );

      act(() => {
        result.current.handleRemovePage("details");
      });

      expect(result.current.localPages).toHaveLength(1);
      expect(onActivePageIdChange).toHaveBeenCalledWith("overview");
    } finally {
      vi.useRealTimers();
    }
  });

  it("never removes the last page", () => {
    const onChange = vi.fn();
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: [twoPages[0]!],
          onChange,
          activePageId: "overview",
          onActivePageIdChange: vi.fn(),
        }),
      );

      act(() => {
        result.current.handleRemovePage("overview");
      });

      expect(result.current.localPages).toHaveLength(1);
    } finally {
      vi.useRealTimers();
    }
  });

  it("reorders pages by index", () => {
    const onChange = vi.fn();
    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: twoPages,
          onChange,
          activePageId: "overview",
          onActivePageIdChange: vi.fn(),
        }),
      );

      act(() => {
        result.current.handleReorderPages(0, 1);
      });

      expect(result.current.localPages.map((p) => p.id)).toEqual(["details", "overview"]);
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("useConsolePagesState — empty console bootstrap", () => {
  it("auto-creates the default page when the first panel is added to an empty console", () => {
    // A brand-new or freshly-cleared console has zero pages. Clicking
    // "Add panel" from the header or empty-state CTA must still work:
    // the hook must materialize a default `main` page in the same
    // update so the panel lands somewhere.
    const onChange = vi.fn();
    const onActivePageIdChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result } = renderHook(() =>
        useConsolePagesState({
          pages: [],
          onChange,
          activePageId: null,
          onActivePageIdChange,
        }),
      );

      let createdId = "";
      act(() => {
        createdId = result.current.handleAddPanel("First panel", "markdown");
      });

      expect(result.current.localPages).toHaveLength(1);
      expect(result.current.localPages[0].id).toBe("main");
      expect(result.current.localPages[0].panels).toHaveLength(1);
      expect(result.current.localPages[0].panels[0].id).toBe(createdId);
      expect(onActivePageIdChange).toHaveBeenCalledWith("main");

      act(() => {
        vi.runAllTimers();
      });

      expect(onChange).toHaveBeenCalledTimes(1);
      expect(onChange.mock.calls[0][0].pages[0].panels[0].id).toBe(createdId);
    } finally {
      vi.useRealTimers();
    }
  });
});

describe("useConsolePagesState — canvas switch invalidates pending saves", () => {
  it("does not stage a pending save from a previous canvas onto a new canvas", () => {
    // Regression test for the high-severity finding "Pending save
    // can corrupt other canvas". React Router keeps this route
    // component mounted across canvases, so a debounced save queued
    // for canvas A would previously fire ~500ms later against the
    // current mutation — which by then targets canvas B, staging
    // A's pages onto B. The `canvasId`-keyed effect in
    // `useDebouncedPages` must clear the pending payload and timer
    // on switch; the payload-canvas guard in the timer is a second
    // line of defense.
    const onChange = vi.fn();
    const onActivePageIdChange = vi.fn();

    vi.useFakeTimers();
    try {
      const { result, rerender } = renderHook(
        ({ canvasId, pages }: { canvasId: string; pages: ConsolePage[] }) =>
          useConsolePagesState({
            pages,
            onChange,
            activePageId: "main",
            onActivePageIdChange,
            canvasId,
          }),
        { initialProps: { canvasId: "canvas-a", pages: [PAGE] } },
      );

      // Edit on canvas A — debounce starts but has not yet elapsed.
      act(() => {
        result.current.handleAddPanel("Notes", "markdown");
      });
      expect(onChange).not.toHaveBeenCalled();

      // Simulate the React Router route reuse across canvases: same
      // hook instance, new `canvasId`, new `pages`. The old timer
      // must be cancelled and the pending payload must not leak.
      const canvasBPage: ConsolePage = { ...PAGE, panels: [], layout: [] };
      rerender({ canvasId: "canvas-b", pages: [canvasBPage] });

      // Fire any residual timers. Nothing should reach `onChange`.
      act(() => {
        vi.runAllTimers();
      });
      expect(onChange).not.toHaveBeenCalled();

      // Sanity: canvas B is editable in its own right, and the save
      // it triggers is not contaminated by canvas A's earlier edit.
      act(() => {
        result.current.handleAddPanel("B Notes", "markdown");
      });
      act(() => {
        vi.runAllTimers();
      });
      expect(onChange).toHaveBeenCalledTimes(1);
      const stagedPages = onChange.mock.calls[0][0].pages as ConsolePage[];
      // Only the panel added on canvas B (a markdown "B Notes") is
      // present; A's earlier markdown "Notes" is not.
      const allPanelTitles = stagedPages.flatMap((page) => page.panels).map((panel) => panel.content?.title);
      expect(allPanelTitles).not.toContain("Notes");
      expect(allPanelTitles).toContain("B Notes");
    } finally {
      vi.useRealTimers();
    }
  });
});
