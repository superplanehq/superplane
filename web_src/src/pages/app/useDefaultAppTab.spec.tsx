import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// The hook only needs the preference query result and the mutation's
// `mutate`; mocking the data hooks keeps the spec free of QueryClient and
// network plumbing.
const mutate = vi.fn();
type MockPreferenceQuery = {
  isPending: boolean;
  isError: boolean;
  isSuccess: boolean;
  data: { lastVisitedTab?: string } | null;
};
let mockPreferenceQuery: MockPreferenceQuery;

function preferenceLoaded(data: { lastVisitedTab?: string } | null): MockPreferenceQuery {
  return { isPending: false, isError: false, isSuccess: true, data };
}

function preferenceLoading(): MockPreferenceQuery {
  return { isPending: true, isError: false, isSuccess: false, data: null };
}

function preferenceErrored(): MockPreferenceQuery {
  return { isPending: false, isError: true, isSuccess: false, data: null };
}

vi.mock("@/hooks/useCanvasData", () => ({
  useCanvasPreference: () => mockPreferenceQuery,
  useUpdateCanvasPreference: () => ({ mutate }),
}));

import { useDefaultAppTab } from "./useDefaultAppTab";

type UrlViewFlags = Parameters<typeof useDefaultAppTab>[0]["urlViewFlags"];
type ConsoleQueryLike = Parameters<typeof useDefaultAppTab>[0]["consoleQuery"];

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
  consoleQuery = undefined,
}: {
  urlViewFlags?: UrlViewFlags;
  searchParams?: URLSearchParams;
  canvasId?: string;
  consoleQuery?: ConsoleQueryLike;
} = {}) {
  const setSearchParams = vi.fn();
  const view = renderHook(
    (props: { urlViewFlags: UrlViewFlags; canvasId: string; consoleQuery: ConsoleQueryLike }) =>
      useDefaultAppTab({
        organizationId: "org-1",
        canvasId: props.canvasId,
        urlViewFlags: props.urlViewFlags,
        searchParams,
        setSearchParams,
        consoleQuery: props.consoleQuery,
      }),
    { initialProps: { urlViewFlags, canvasId, consoleQuery } },
  );
  return { ...view, setSearchParams };
}

beforeEach(() => {
  mutate.mockClear();
  mockPreferenceQuery = preferenceLoaded(null);
});

describe("useDefaultAppTab — stored-tab redirect vs. tab recording", () => {
  it("does not persist the pre-redirect tab while the redirect to the stored tab is still pending", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { setSearchParams } = renderDefaultAppTab();

    // The redirect to the stored tab was scheduled…
    expect(setSearchParams).toHaveBeenCalledTimes(1);
    // …and the record effect, which runs in the same commit while the URL
    // still reports Canvas, must not overwrite the stored tab with "canvas".
    expect(mutate).not.toHaveBeenCalled();
  });

  it("does not write the stored tab back to the server once the redirect lands", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { rerender } = renderDefaultAppTab();
    // URL catches up with the redirect on the next render.
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(mutate).not.toHaveBeenCalled();
  });

  it("records a genuine tab change after the redirect has settled", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { rerender } = renderDefaultAppTab();
    rerender({ urlViewFlags: CONSOLE_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("skips the redirect when the stored tab already matches the current tab, preserving other params", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "canvas" });

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
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "canvas" });

    const { rerender } = renderDefaultAppTab({
      searchParams: new URLSearchParams("node=abc"),
    });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("records the current tab when there is no stored preference and no redirect", () => {
    renderDefaultAppTab();

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());
  });

  it("re-applies the stored-tab redirect for the next app when AppPage is reused across apps", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "canvas" });

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // First app: already on the stored tab, so no redirect.
    expect(setSearchParams).not.toHaveBeenCalled();

    // Navigate to another app (same AppPage instance) whose stored tab is Console.
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2", consoleQuery: undefined });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("records the tab for the next app even when it matches the previous app's last recorded tab", () => {
    const { rerender } = renderDefaultAppTab();
    // First app: no stored preference, records "canvas".
    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());

    // Second app also lands on Canvas; the recording guard must not treat it
    // as a duplicate of the previous app's write.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-2", consoleQuery: undefined });

    expect(mutate).toHaveBeenCalledTimes(2);
    expect(mutate).toHaveBeenLastCalledWith({ canvasId: "canvas-2", lastVisitedTab: "canvas" }, expect.anything());
  });

  it("yields to a tab the user picked while the stored preference was still loading", () => {
    mockPreferenceQuery = preferenceLoading();

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // User opens Memory before the preference query resolves.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    // The stored preference arrives late and points elsewhere.
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    // No redirect: the user's explicit choice wins and is recorded.
    expect(setSearchParams).not.toHaveBeenCalled();
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("does not overwrite the stored tab when closing run inspection lands on Canvas", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    // Closing the run drops the `run` param and lands on Canvas.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(mutate).not.toHaveBeenCalled();
  });

  it("still records a deliberate tab change made after closing run inspection", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: RUN_FLAGS,
      searchParams: new URLSearchParams("run=run-1"),
    });
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("skips the redirect entirely when the URL already selects a view", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { setSearchParams } = renderDefaultAppTab({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory"),
    });

    expect(setSearchParams).not.toHaveBeenCalled();
    // The explicitly selected tab is recorded as usual.
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("neither redirects nor records when the preference failed to load", () => {
    mockPreferenceQuery = preferenceErrored();

    const { rerender, setSearchParams } = renderDefaultAppTab();
    // Even an explicit tab switch is not persisted: without the stored tab we
    // cannot tell "no preference" from "failed to load", and writing could
    // overwrite a preference that actually exists.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });

    expect(setSearchParams).not.toHaveBeenCalled();
    expect(mutate).not.toHaveBeenCalled();
  });

  it("applies the Console fallback once an in-flight console query succeeds", () => {
    mockPreferenceQuery = preferenceLoaded(null);

    const pendingConsoleQuery: ConsoleQueryLike = { isSuccess: false, isError: false, data: undefined };
    const { rerender, setSearchParams } = renderDefaultAppTab({ consoleQuery: pendingConsoleQuery });
    // An unfinished read must not lock in Canvas.
    expect(setSearchParams).not.toHaveBeenCalled();

    // The read succeeds and reports panels: the fallback still applies.
    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      consoleQuery: {
        isSuccess: true,
        isError: false,
        data: {
          canvasId: "canvas-1",
          panels: [{ id: "p1", type: "markdown", content: {} }],
          layout: [],
          consoleYaml: "",
        },
      },
    });

    expect(setSearchParams).toHaveBeenCalledTimes(1);
  });

  it("records the current tab when resolution settles on a later render without a URL change", () => {
    mockPreferenceQuery = preferenceLoaded(null);

    // Mount while the console read is still in flight: nothing settles yet.
    const pendingConsoleQuery: ConsoleQueryLike = { isSuccess: false, isError: false, data: undefined };
    const { rerender } = renderDefaultAppTab({ consoleQuery: pendingConsoleQuery });
    expect(mutate).not.toHaveBeenCalled();

    // The read succeeds with no panels: resolution settles on Canvas without
    // touching the URL, and that alone must unblock tab recording.
    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      consoleQuery: {
        isSuccess: true,
        isError: false,
        data: { canvasId: "canvas-1", panels: [], layout: [], consoleYaml: "" },
      },
    });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());
  });

  it("records the current tab when the console query errors on a later render", () => {
    mockPreferenceQuery = preferenceLoaded(null);

    const pendingConsoleQuery: ConsoleQueryLike = { isSuccess: false, isError: false, data: undefined };
    const { rerender } = renderDefaultAppTab({ consoleQuery: pendingConsoleQuery });
    expect(mutate).not.toHaveBeenCalled();

    rerender({
      urlViewFlags: CANVAS_FLAGS,
      canvasId: "canvas-1",
      consoleQuery: { isSuccess: false, isError: true, data: undefined },
    });

    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());
  });

  it("settles without a redirect when the console query errors, so tab recording is not blocked", () => {
    mockPreferenceQuery = preferenceLoaded(null);

    const erroredConsoleQuery: ConsoleQueryLike = { isSuccess: false, isError: true, data: undefined };
    const { rerender, setSearchParams } = renderDefaultAppTab({ consoleQuery: erroredConsoleQuery });

    // No Console fallback on error, but the resolution settles on Canvas…
    expect(setSearchParams).not.toHaveBeenCalled();
    expect(mutate).toHaveBeenCalledTimes(1);
    expect(mutate).toHaveBeenCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());

    // …so a later tab change is recorded without the user having to switch twice.
    rerender({ urlViewFlags: MEMORY_FLAGS, canvasId: "canvas-1", consoleQuery: erroredConsoleQuery });

    expect(mutate).toHaveBeenCalledTimes(2);
    expect(mutate).toHaveBeenLastCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("retries a failed tab write immediately instead of treating it as recorded", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    renderDefaultAppTab({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory"),
    });
    expect(mutate).toHaveBeenCalledTimes(1);

    // The PUT fails; clearing the recorded-tab guard must re-run the record
    // effect and retry the write without any further user interaction.
    const mutateOptions = mutate.mock.calls[0][1] as { onError: () => void };
    act(() => mutateOptions.onError());

    expect(mutate).toHaveBeenCalledTimes(2);
    expect(mutate).toHaveBeenLastCalledWith({ canvasId: "canvas-1", lastVisitedTab: "memory" }, expect.anything());
  });

  it("stops retrying a persistently failing tab write after the attempt budget is exhausted", () => {
    mockPreferenceQuery = preferenceLoaded({ lastVisitedTab: "console" });

    const { rerender } = renderDefaultAppTab({
      urlViewFlags: MEMORY_FLAGS,
      searchParams: new URLSearchParams("view=memory"),
    });

    // Every write fails: initial attempt plus retries, capped at 3 total.
    const failLatestWrite = () => {
      const call = mutate.mock.calls[mutate.mock.calls.length - 1];
      const options = call[1] as { onError: () => void };
      act(() => options.onError());
    };
    failLatestWrite();
    failLatestWrite();
    failLatestWrite();
    expect(mutate).toHaveBeenCalledTimes(3);

    // A genuine tab switch grants a fresh budget.
    rerender({ urlViewFlags: CANVAS_FLAGS, canvasId: "canvas-1", consoleQuery: undefined });
    expect(mutate).toHaveBeenCalledTimes(4);
    expect(mutate).toHaveBeenLastCalledWith({ canvasId: "canvas-1", lastVisitedTab: "canvas" }, expect.anything());
  });
});
