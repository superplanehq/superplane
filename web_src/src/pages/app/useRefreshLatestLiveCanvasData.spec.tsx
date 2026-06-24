import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { canvasKeys } from "@/hooks/useCanvasData";

import { useRefreshLatestLiveCanvasData } from "./useRefreshLatestLiveCanvasData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function createInvalidateSpy() {
  const queryClient = new QueryClient();
  const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue(undefined);
  return { queryClient, invalidateQueries };
}

function expectInvalidation(
  invalidateQueries: ReturnType<typeof vi.spyOn>,
  queryKey: readonly unknown[],
  extra?: Record<string, unknown>,
) {
  expect(invalidateQueries).toHaveBeenCalledWith({
    queryKey,
    refetchType: "all",
    ...extra,
  });
}

describe("useRefreshLatestLiveCanvasData", () => {
  it("invalidates live canvas queries by default, including draft branches", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", "live-version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current();
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.versionList("canvas-1"), { exact: true });
    expectInvalidation(invalidateQueries, canvasKeys.versionHistory("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.draftBranches("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.console("canvas-1", "live-version-1"), { exact: true });
    expectInvalidation(invalidateQueries, canvasKeys.console("canvas-1", undefined), { exact: true });
  });

  it("does nothing when organization or canvas id is missing", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result: missingOrg } = renderHook(
      () => useRefreshLatestLiveCanvasData(undefined, "canvas-1", "live-version-1"),
      {
        wrapper: createWrapper(queryClient),
      },
    );
    await act(async () => {
      await missingOrg.current();
    });
    expect(invalidateQueries).not.toHaveBeenCalled();

    const missingCanvasQueryClient = new QueryClient();
    const missingCanvasInvalidate = vi
      .spyOn(missingCanvasQueryClient, "invalidateQueries")
      .mockResolvedValue(undefined);
    const { result: missingCanvas } = renderHook(
      () => useRefreshLatestLiveCanvasData("org-1", undefined, "live-version-1"),
      { wrapper: createWrapper(missingCanvasQueryClient) },
    );
    await act(async () => {
      await missingCanvas.current();
    });
    expect(missingCanvasInvalidate).not.toHaveBeenCalled();
  });

  it("skips version-specific console invalidation when no live version id is available", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", undefined), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current();
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.console("canvas-1", undefined), { exact: true });

    const consoleInvalidations = invalidateQueries.mock.calls
      .map(([args]) => args.queryKey)
      .filter((key) => Array.isArray(key) && key.includes("console"));
    expect(consoleInvalidations).toEqual([canvasKeys.console("canvas-1", undefined)]);
  });

  it("invalidates publish console caches and skips draft branches when requested", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", "old-live-version"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current({ liveVersionId: "published-version", skipDraftBranchRefetch: true });
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.versionList("canvas-1"), { exact: true });
    expectInvalidation(invalidateQueries, canvasKeys.versionHistory("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.console("canvas-1", "published-version"), { exact: true });
    expectInvalidation(invalidateQueries, canvasKeys.console("canvas-1", undefined), { exact: true });
    expect(invalidateQueries).not.toHaveBeenCalledWith(
      expect.objectContaining({
        queryKey: canvasKeys.draftBranches("canvas-1"),
      }),
    );
  });
});
