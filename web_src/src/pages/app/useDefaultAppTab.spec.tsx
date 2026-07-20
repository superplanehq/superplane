import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useDefaultAppTab } from "./useDefaultAppTab";
import {
  LAST_VISITED_APP_TAB_STORAGE_KEY,
  readLastVisitedAppTab,
  recordLastVisitedAppTab,
} from "@/lib/lastVisitedAppTab";

type UrlViewFlags = Parameters<typeof useDefaultAppTab>[0]["urlViewFlags"];

const CANVAS_FLAGS: UrlViewFlags = {
  isRunInspectionMode: false,
  isMemoryMode: false,
  isFilesMode: false,
  isConsoleMode: false,
};

const CONSOLE_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isConsoleMode: true };
const MEMORY_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isMemoryMode: true };
const RUN_FLAGS: UrlViewFlags = { ...CANVAS_FLAGS, isRunInspectionMode: true };

function renderPersistHook({
  urlViewFlags = CANVAS_FLAGS,
  searchParams = new URLSearchParams(),
  canvasId = "canvas-1",
}: {
  urlViewFlags?: UrlViewFlags;
  searchParams?: URLSearchParams;
  canvasId?: string;
} = {}) {
  type HookProps = { urlViewFlags: UrlViewFlags; canvasId: string; searchParams?: URLSearchParams };
  return renderHook(
    (props: HookProps) =>
      useDefaultAppTab({
        canvasId: props.canvasId,
        urlViewFlags: props.urlViewFlags,
        searchParams: props.searchParams ?? searchParams,
      }),
    { initialProps: { urlViewFlags, canvasId } as HookProps },
  );
}

beforeEach(() => {
  window.localStorage.clear();
});

describe("useDefaultAppTab — landing persistence", () => {
  it("records the current tab on mount when no stored preference exists yet", () => {
    renderPersistHook();

    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("records the current tab on mount even when a stored preference exists (gate has already landed us here)", () => {
    recordLastVisitedAppTab("canvas-1", "canvas");
    renderPersistHook({ urlViewFlags: CONSOLE_FLAGS });

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("avoids rewriting the stored tab when it already matches the current tab", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    const setSpy = vi.spyOn(Storage.prototype, "setItem");

    renderPersistHook({ urlViewFlags: CONSOLE_FLAGS });

    expect(setSpy).not.toHaveBeenCalled();
    setSpy.mockRestore();
  });

  it("records the tab under the new canvas id when the same instance switches apps", () => {
    const { rerender } = renderPersistHook();
    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");

    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-2" });

    expect(readLastVisitedAppTab("canvas-2")).toBe("console");
    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("skips persistence when the canvas id is missing", () => {
    const setSpy = vi.spyOn(Storage.prototype, "setItem");

    renderPersistHook({ canvasId: "" });

    expect(setSpy).not.toHaveBeenCalled();
    setSpy.mockRestore();
  });
});

describe("useDefaultAppTab — tab changes on the same visit", () => {
  it("records a deliberate tab change after mount", () => {
    const { rerender } = renderPersistHook();
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("records successive tab changes as they happen", () => {
    const { rerender } = renderPersistHook();
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1" });
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");

    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });
    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("handles a stored invalid value the same way as no stored value", () => {
    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify({ "canvas-1": "bogus" }));

    renderPersistHook({ urlViewFlags: MEMORY_FLAGS });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });
});

describe("useDefaultAppTab — run inspection", () => {
  it("does not overwrite the stored tab while run inspection is active", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    renderPersistHook({ urlViewFlags: RUN_FLAGS, searchParams: new URLSearchParams("run=run-1") });

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("does not overwrite the stored tab when closing run inspection lands on Canvas", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderPersistHook({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("still records a deliberate tab change made after closing run inspection", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderPersistHook({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });
});

describe("useDefaultAppTab — deep-link landings", () => {
  it.each([
    ["version preview", "version=v1"],
    ["edit session", "edit=1"],
    ["node selection", "node=n1&sidebar=1"],
    ["file selection", "file=components%2Fapp.yaml"],
  ])("does not overwrite the stored tab on a %s landing", (_label, query) => {
    recordLastVisitedAppTab("canvas-1", "console");

    renderPersistHook({ searchParams: new URLSearchParams(query) });

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("still records a deliberate tab change made after a deep-link landing", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderPersistHook({
      searchParams: new URLSearchParams("version=v1"),
    });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("records the tab when a deep link is paired with an explicit view (the tab was picked)", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    renderPersistHook({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory&node=n1"),
    });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });
});
