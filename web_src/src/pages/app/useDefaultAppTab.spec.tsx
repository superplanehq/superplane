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

function renderDefaultAppTab({
  urlViewFlags = CANVAS_FLAGS,
  searchParams = new URLSearchParams(),
}: {
  urlViewFlags?: UrlViewFlags;
  searchParams?: URLSearchParams;
} = {}) {
  const setSearchParams = vi.fn();
  const view = renderHook(
    (props: { urlViewFlags: UrlViewFlags }) =>
      useDefaultAppTab({
        organizationId: "org-1",
        canvasId: "canvas-1",
        urlViewFlags: props.urlViewFlags,
        searchParams,
        setSearchParams,
        consoleQuery: undefined,
      }),
    { initialProps: { urlViewFlags } },
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
    rerender({ urlViewFlags: CONSOLE_FLAGS });

    expect(mutate).not.toHaveBeenCalled();
  });

  it("records a genuine tab change after the redirect has settled", () => {
    mockPreferenceQuery = { isPending: false, data: { lastVisitedTab: "console" } };

    const { rerender } = renderDefaultAppTab();
    rerender({ urlViewFlags: CONSOLE_FLAGS });
    rerender({ urlViewFlags: MEMORY_FLAGS });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" });
  });

  it("records the current tab when there is no stored preference and no redirect", () => {
    renderDefaultAppTab();

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" });
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
