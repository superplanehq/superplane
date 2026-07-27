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
