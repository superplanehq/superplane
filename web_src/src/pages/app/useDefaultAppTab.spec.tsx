import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// The hook only needs the live-console query result; mocking the data hook
// keeps the spec free of QueryClient and network plumbing. The stored tab is
// persisted to localStorage, which vitest resets per test file via jsdom.
type ConsoleQueryLike = {
  isSuccess: boolean;
  isError: boolean;
  data: { canvasId: string; panels: object[]; layout: object[]; consoleYaml: string } | undefined;
};
let mockLiveConsoleQuery: ConsoleQueryLike;

function consoleLoaded(panels: object[]): ConsoleQueryLike {
  return {
    isSuccess: true,
    isError: false,
    data: { canvasId: "canvas-1", panels, layout: [], consoleYaml: "" },
  };
}

function consoleLoading(): ConsoleQueryLike {
  return { isSuccess: false, isError: false, data: undefined };
}

function consoleErrored(): ConsoleQueryLike {
  return { isSuccess: false, isError: true, data: undefined };
}

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasConsole: () => mockLiveConsoleQuery,
}));

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

function renderDefaultAppTab({
  urlViewFlags = CANVAS_FLAGS,
  searchParams = new URLSearchParams(),
  canvasId = "canvas-1",
}: {
  urlViewFlags?: UrlViewFlags;
  searchParams?: URLSearchParams;
  canvasId?: string;
} = {}) {
  const setSearchParams = vi.fn();
  type HookProps = { urlViewFlags: UrlViewFlags; canvasId: string; searchParams?: URLSearchParams };
  const view = renderHook(
    (props: HookProps) =>
      useDefaultAppTab({
        canvasId: props.canvasId,
        urlViewFlags: props.urlViewFlags,
        searchParams: props.searchParams ?? searchParams,
        setSearchParams,
      }),
    { initialProps: { urlViewFlags, canvasId } as HookProps },
  );
  return { ...view, setSearchParams };
}

beforeEach(() => {
  window.localStorage.clear();
  mockLiveConsoleQuery = consoleLoaded([]);
});

describe("useDefaultAppTab — stored-tab redirect vs. tab recording", () => {
  it("does not persist the pre-redirect tab while the redirect to the stored tab is still pending", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab();

    expect(setSearchParams).toHaveBeenCalledTimes(1);
    // The record effect runs in the same commit while the URL still reports
    // Canvas; it must not overwrite the stored "console" tab with "canvas".
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("does not rewrite the stored tab once the redirect lands on it", () => {
    recordLastVisitedAppTab("canvas-1", "console");
    const setSpy = vi.spyOn(Storage.prototype, "setItem");

    const { rerender } = renderDefaultAppTab();
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1" });

    expect(setSpy).not.toHaveBeenCalled();
    setSpy.mockRestore();
  });

  it("records a genuine tab change after the redirect has settled", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderDefaultAppTab();
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("persists a tab the user picked after the redirect was scheduled but before it applied", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender, setSearchParams } = renderDefaultAppTab();
    expect(setSearchParams).toHaveBeenCalledTimes(1);

    // User switches to Memory before the redirect applies. Their explicit
    // choice must be recorded; bailing until the URL reports the redirect
    // target would block tab recording indefinitely.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");

    // Recording keeps working for later tab changes too.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });
    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("turns a late-applying redirect into a no-op once the user picked a different tab", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender, setSearchParams } = renderDefaultAppTab();
    expect(setSearchParams).toHaveBeenCalledTimes(1);
    const updater = setSearchParams.mock.calls[0][0] as (prev: URLSearchParams) => URLSearchParams;

    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    // When the queued redirect finally runs, it must not replace the user's
    // choice with the stored tab.
    const current = new URLSearchParams("view=memory");
    expect(updater(current)).toBe(current);
  });

  it("skips the redirect when the stored tab already matches the current tab, preserving other params", () => {
    recordLastVisitedAppTab("canvas-1", "canvas");

    const setSpy = vi.spyOn(Storage.prototype, "setItem");
    const { setSearchParams } = renderDefaultAppTab({
      // Refresh on the Canvas tab with node selection + sidebar state in the
      // URL (no `view` param). Rewriting the URL here would delete
      // `node`/`sidebar`/`file` even though no tab change is needed.
      searchParams: new URLSearchParams("node=abc&sidebar=nodeEditor"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // Identical stored value — nothing to write back either.
    expect(setSpy).not.toHaveBeenCalled();
    setSpy.mockRestore();
  });

  it("still records a later tab change after the redirect was skipped as already-on-tab", () => {
    recordLastVisitedAppTab("canvas-1", "canvas");

    const { rerender } = renderDefaultAppTab({
      searchParams: new URLSearchParams("node=abc"),
    });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("records the current tab when there is no stored preference and no redirect", () => {
    renderDefaultAppTab();

    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("re-applies the stored-tab redirect for the next app when AppPage is reused across apps", () => {
    recordLastVisitedAppTab("canvas-1", "canvas");
    recordLastVisitedAppTab("canvas-2", "console");

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // First app: already on the stored tab, so no redirect.
    expect(setSearchParams).not.toHaveBeenCalled();

    // Navigate to another app (same AppPage instance) whose stored tab is Console.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2" });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("records the tab for the next app under its own canvas id", () => {
    const { rerender } = renderDefaultAppTab();
    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");

    // Second app also lands on Canvas; recording must happen against the
    // new canvas id, not skipped as "same as before".
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2" });

    expect(readLastVisitedAppTab("canvas-2")).toBe("canvas");
  });

  it("does not overwrite the stored tab when closing run inspection lands on Canvas", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    // Closing the run drops the `run` param and lands on Canvas.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("still records a deliberate tab change made after closing run inspection", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });
});

describe("useDefaultAppTab — deep links and explicit view params", () => {
  it("skips the redirect entirely when the URL already selects a view", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // The explicitly selected tab is recorded as usual.
    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("skips the redirect and keeps the stored tab when the URL deep-links to a version preview", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    // A `?version=` link lands on the canvas view of that version; redirecting
    // to the stored Console tab would pull the user off the previewed version.
    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("version=version-1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // The landing was not a tab pick; it must not replace the stored tab.
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("skips the redirect and keeps the stored tab when the URL requests an edit session", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    // A `?edit=1` link enters an edit session on the canvas view; redirecting
    // to the stored Console tab would break the edit-session entry.
    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("edit=1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("skips the redirect and keeps the stored tab when the URL deep-links to a node selection", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    // A shared node link lands on the canvas with the node's sidebar open;
    // redirecting to the stored Console tab would drop that selection (the
    // redirect deletes `node`/`sidebar`) and open the wrong tab.
    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("node=node-1&sidebar=1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });

  it("skips the redirect when the URL deep-links to a file", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("file=components/app.yaml"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
  });

  it("skips the Console fallback when the URL deep-links to a version preview", () => {
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);

    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("version=version-1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
  });

  it("still records a deliberate tab change made after a deep-link landing", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { rerender } = renderDefaultAppTab({
      searchParams: new URLSearchParams("version=version-1"),
    });
    // Only the landing itself is exempt from recording; a tab the user picks
    // afterwards is persisted as usual.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });
});

describe("useDefaultAppTab — legacy view params", () => {
  // Legacy `view=runs` / `view=versions` bookmarks are deleted on mount by
  // useWorkflowViewSearchParams and land on the bare canvas URL. They select
  // no tab, so they must not pin navigation: the stored-tab redirect and the
  // Console fallback still apply for that visit.
  it("applies the stored-tab redirect when the URL carries the legacy runs view", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("view=runs"),
    });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("applies the stored-tab redirect when the URL carries the legacy versions view", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("view=versions"),
    });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("applies the Console fallback when the URL carries the legacy runs view", () => {
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);

    const { setSearchParams } = renderDefaultAppTab({
      searchParams: new URLSearchParams("view=runs"),
    });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("still pins navigation when the legacy runs view carries a run id (run inspection)", () => {
    recordLastVisitedAppTab("canvas-1", "console");

    const { setSearchParams } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("view=runs&run=run-1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
  });

  it("still pins navigation on the legacy dashboard alias for Console", () => {
    recordLastVisitedAppTab("canvas-1", "memory");

    const { setSearchParams } = renderDefaultAppTab({
      urlViewFlags: CONSOLE_FLAGS,
      searchParams: new URLSearchParams("view=dashboard"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // `view=dashboard` is an explicit Console pick; it is recorded as usual.
    expect(readLastVisitedAppTab("canvas-1")).toBe("console");
  });
});

describe("useDefaultAppTab — console query resolution", () => {
  it("applies the Console fallback once an in-flight console query succeeds", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender, setSearchParams } = renderDefaultAppTab();
    // An unfinished read must not lock in Canvas.
    expect(setSearchParams).not.toHaveBeenCalled();

    // The read succeeds and reports panels: the fallback still applies.
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("records the current tab when resolution settles on a later render without a URL change", () => {
    // Mount while the console read is still in flight: nothing settles yet.
    mockLiveConsoleQuery = consoleLoading();
    const { rerender } = renderDefaultAppTab();
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();

    // The read succeeds with no panels: resolution settles on Canvas without
    // touching the URL, and that alone must unblock tab recording.
    mockLiveConsoleQuery = consoleLoaded([]);
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("records the current tab when the console query errors on a later render", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender } = renderDefaultAppTab();
    expect(readLastVisitedAppTab("canvas-1")).toBeNull();

    mockLiveConsoleQuery = consoleErrored();
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");
  });

  it("settles without a redirect when the console query errors, so tab recording is not blocked", () => {
    mockLiveConsoleQuery = consoleErrored();
    const { rerender, setSearchParams } = renderDefaultAppTab();

    // No Console fallback on error, but the resolution settles on Canvas…
    expect(setSearchParams).not.toHaveBeenCalled();
    expect(readLastVisitedAppTab("canvas-1")).toBe("canvas");

    // …so a later tab change is recorded without the user having to switch twice.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("yields the Console fallback to a node selection made while the console query was still loading", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender, setSearchParams } = renderDefaultAppTab();

    // User opens a node on Canvas (no tab change) before the query resolves.
    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      searchParams: new URLSearchParams("node=node-1&sidebar=1"),
    });

    // The console query resolves late and would have redirected to Console —
    // stripping the `node`/`sidebar` selection in the process.
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);
    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      searchParams: new URLSearchParams("node=node-1&sidebar=1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
  });

  it("yields the Console fallback to a version preview opened while the console query was still loading", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender, setSearchParams } = renderDefaultAppTab();

    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      searchParams: new URLSearchParams("version=version-1"),
    });

    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);
    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      searchParams: new URLSearchParams("version=version-1"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
  });

  it("settles when a pinning param appears mid-resolution, so tab recording is not blocked", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender } = renderDefaultAppTab();

    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      searchParams: new URLSearchParams("node=node-1"),
    });

    // A later deliberate tab change is recorded even though the console query
    // never resolved.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("yields the Console fallback to a tab the user picked while the console query was still loading", () => {
    mockLiveConsoleQuery = consoleLoading();
    const { rerender, setSearchParams } = renderDefaultAppTab();

    // User opens Memory before the console query resolves.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    // The console query resolves late and would have redirected to Console.
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    // No redirect: the user's explicit choice wins and is recorded.
    expect(setSearchParams).not.toHaveBeenCalled();
    expect(readLastVisitedAppTab("canvas-1")).toBe("memory");
  });

  it("ignores an invalid stored tab and falls through to the Console-fallback path", () => {
    window.localStorage.setItem(LAST_VISITED_APP_TAB_STORAGE_KEY, JSON.stringify({ "canvas-1": "bogus" }));
    mockLiveConsoleQuery = consoleLoaded([{ id: "p1", type: "markdown", content: {} }]);

    const { setSearchParams } = renderDefaultAppTab();

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });
});
