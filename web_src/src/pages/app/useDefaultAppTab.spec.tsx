import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// The hook only needs the preference query result and the mutation's
// `mutate`; mocking the data hooks keeps the spec free of QueryClient and
// network plumbing.
const mutate = vi.fn();
let mockPreferenceQuery: { isPending: boolean; data: { lastVisitedTab?: string } | null };
vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasPreference: () => mockPreferenceQuery,
  useUpdateCanvasPreference: () => ({ mutate }),
}));

import { useDefaultAppTab } from "./useDefaultAppTab";

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
  const view = renderHook(
    (props: { urlViewFlags: UrlViewFlags; canvasId: string }) =>
      useDefaultAppTab({
        organizationId: "org-1",
        canvasId: props.canvasId,
        urlViewFlags: props.urlViewFlags,
        searchParams,
        setSearchParams,
        consoleQuery: undefined,
      }),
    { initialProps: { urlViewFlags, canvasId } },
  );
  return { ...view, setSearchParams };
}

beforeEach(() => {
  mutate.mockClear();
  mockPreferenceQuery = { isPending: false, data: null };
});

describe("useDefaultAppTab — stored-tab redirect vs. tab recording", () => {
  it("does not persist the pre-redirect tab while the redirect to the stored tab is still pending", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { setSearchParams } = renderDefaultAppTab();

    // The redirect to the stored tab was scheduled…
    expect(setSearchParams).toHaveBeenCalledTimes(1);
    // …and the record effect, which runs in the same commit while the URL
    // still reports Canvas, must not overwrite the stored tab with "canvas".
    expect(mutate).not.toHaveBeenCalled();
  });

  it("does not write the stored tab back to the server once the redirect lands", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { rerender } = renderDefaultAppTab();
    // URL catches up with the redirect on the next render.
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1" });

    expect(mutate).not.toHaveBeenCalled();
  });

  it("records a genuine tab change after the redirect has settled", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { rerender } = renderDefaultAppTab();
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });

  it("skips the redirect when the stored tab already matches the current tab, preserving other params", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "canvas" } };

    const { setSearchParams } = renderDefaultAppTab({
      // Refresh on the Canvas tab with node selection + sidebar state in the
      // URL (no `view` param). Rewriting the URL here would delete
      // `node`/`sidebar`/`file` even though no tab change is needed.
      searchParams: new URLSearchParams("node=abc&sidebar=nodeEditor"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // Identical stored value — nothing to write back either.
    expect(mutate).not.toHaveBeenCalled();
  });

  it("still records a later tab change after the redirect was skipped as already-on-tab", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "canvas" } };

    const { rerender } = renderDefaultAppTab({
      searchParams: new URLSearchParams("node=abc"),
    });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });

  it("records the current tab when there is no stored preference and no redirect", () => {
    renderDefaultAppTab();

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" });
  });

  it("re-applies the stored-tab redirect for the next app when AppPage is reused across apps", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "canvas" } };

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // First app: already on the stored tab, so no redirect.
    expect(setSearchParams).not.toHaveBeenCalled();

    // Navigate to another app (same AppPage instance) whose stored tab is Console.
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2" });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("records the tab for the next app even when it matches the previous app's last recorded tab", () => {
    const { rerender } = renderDefaultAppTab();
    // First app: no stored preference, records "canvas".
    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" });

    // Second app also lands on Canvas; the recording guard must not treat it
    // as a duplicate of the previous app's write.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2" });

    expect(mutate).toHaveBeenCalledTimes(2);
    expect(mutate).toHaveBeenLastCalledWith({ canvasId: "canvas-2", lastVisitedTab: "canvas" });
  });

  it("yields to a tab the user picked while the stored preference was still loading", () => {
    mockPreferenceQuery = { isPending: true, data: null };

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // User opens Memory before the preference query resolves.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    // The stored preference arrives late and points elsewhere.
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    // No redirect: the user's explicit choice wins and is recorded.
    expect(setSearchParams).not.toHaveBeenCalled();
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });

  it("does not overwrite the stored tab when closing run inspection lands on Canvas", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    // Closing the run drops the `run` param and lands on Canvas.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });

    expect(mutate).not.toHaveBeenCalled();
  });

  it("still records a deliberate tab change made after closing run inspection", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1" });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });

  it("skips the redirect entirely when the URL already selects a view", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { setSearchParams } = renderDefaultAppTab({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // The explicitly selected tab is recorded as usual.
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });
});
