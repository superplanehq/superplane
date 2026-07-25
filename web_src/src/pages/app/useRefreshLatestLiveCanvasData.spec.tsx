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
  it("invalidates live canvas queries by default, including canvas staging", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", "live-version-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current();
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.versionHistory("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.canvasStaging("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.version("canvas-1", "live-version-1"), { exact: true });
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

  it("skips version invalidation when no live version id is available", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", undefined), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current();
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.versionHistory("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.canvasStaging("canvas-1"));
    expect(invalidateQueries).toHaveBeenCalledTimes(3);
  });

  it("uses the provided live version id for version invalidation", async () => {
    const { queryClient, invalidateQueries } = createInvalidateSpy();

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", "old-live-version"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current({ liveVersionId: "published-version" });
    });

    expectInvalidation(invalidateQueries, canvasKeys.detail("org-1", "canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.versionHistory("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.canvasStaging("canvas-1"));
    expectInvalidation(invalidateQueries, canvasKeys.version("canvas-1", "published-version"), { exact: true });
  });
});
