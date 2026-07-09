import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { canvasesDescribeCanvas, canvasesUpdateCanvasPreference } = vi.hoisted(() => ({
  canvasesDescribeCanvas: vi.fn(),
  canvasesUpdateCanvasPreference: vi.fn(),
}));

vi.mock("../api-client/sdk.gen", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    canvasesDescribeCanvas,
    canvasesUpdateCanvasPreference,
  };
});

import { canvasKeys, useCanvas, useCanvasPreference, useUpdateCanvasPreference } from "@/hooks/useCanvasData";

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });

  return { promise, resolve, reject };
}

describe("useCanvas + useCanvasPreference fetch sharing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("issues a single describe request when both hooks mount together", async () => {
    const queryClient = createQueryClient();
    canvasesDescribeCanvas.mockResolvedValue({
      data: {
        canvas: { metadata: { id: "canvas-1", name: "My canvas" } },
        preference: { lastVisitedTab: "console" },
      },
    });

    const { result } = renderHook(
      () => ({
        canvas: useCanvas("org-1", "canvas-1"),
        preference: useCanvasPreference("org-1", "canvas-1"),
      }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(result.current.canvas.data?.metadata?.id).toBe("canvas-1");
      expect(result.current.preference.data?.lastVisitedTab).toBe("console");
    });

    expect(canvasesDescribeCanvas).toHaveBeenCalledTimes(1);
  });

  it("seeds the preference cache from a canvas detail fetch", async () => {
    const queryClient = createQueryClient();
    canvasesDescribeCanvas.mockResolvedValue({
      data: {
        canvas: { metadata: { id: "canvas-1", name: "My canvas" } },
        preference: { lastVisitedTab: "memory" },
      },
    });

    const { result } = renderHook(() => useCanvas("org-1", "canvas-1"), {
      wrapper: createWrapper(queryClient),
    });

    await waitFor(() => {
      expect(result.current.data?.metadata?.id).toBe("canvas-1");
    });

    expect(queryClient.getQueryData(canvasKeys.preference("org-1", "canvas-1"))).toEqual({
      lastVisitedTab: "memory",
    });
  });
});

describe("useUpdateCanvasPreference", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("sequences writes for the same canvas so a slow older write cannot overwrite a newer one", async () => {
    const queryClient = createQueryClient();
    const firstWrite = createDeferred<unknown>();
    const secondWrite = createDeferred<unknown>();
    canvasesUpdateCanvasPreference.mockReturnValueOnce(firstWrite.promise).mockReturnValueOnce(secondWrite.promise);

    const { result } = renderHook(() => useUpdateCanvasPreference("org-1"), {
      wrapper: createWrapper(queryClient),
    });

    let firstMutation!: Promise<unknown>;
    let secondMutation!: Promise<unknown>;
    act(() => {
      firstMutation = result.current.mutateAsync({ canvasId: "canvas-1", lastVisitedTab: "console" });
      secondMutation = result.current.mutateAsync({ canvasId: "canvas-1", lastVisitedTab: "memory" });
    });

    // The second request must wait for the first to settle; issuing them
    // concurrently is what allowed an older response to land last.
    await waitFor(() => expect(canvasesUpdateCanvasPreference).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(canvasesUpdateCanvasPreference).toHaveBeenCalledTimes(1);

    firstWrite.resolve({ data: { preference: { lastVisitedTab: "console" } } });
    await act(() => firstMutation);

    await waitFor(() => expect(canvasesUpdateCanvasPreference).toHaveBeenCalledTimes(2));
    expect(canvasesUpdateCanvasPreference).toHaveBeenLastCalledWith(
      expect.objectContaining({
        path: { canvasId: "canvas-1" },
        body: expect.objectContaining({ lastVisitedTab: "memory" }),
      }),
    );

    secondWrite.resolve({ data: { preference: { lastVisitedTab: "memory" } } });
    await act(() => secondMutation);

    expect(queryClient.getQueryData(canvasKeys.preference("org-1", "canvas-1"))).toEqual({
      lastVisitedTab: "memory",
    });
  });

  it("still runs a queued write when the write before it fails", async () => {
    const queryClient = createQueryClient();
    const firstWrite = createDeferred<unknown>();
    const secondWrite = createDeferred<unknown>();
    canvasesUpdateCanvasPreference.mockReturnValueOnce(firstWrite.promise).mockReturnValueOnce(secondWrite.promise);

    const { result } = renderHook(() => useUpdateCanvasPreference("org-1"), {
      wrapper: createWrapper(queryClient),
    });

    let firstMutation!: Promise<unknown>;
    let secondMutation!: Promise<unknown>;
    act(() => {
      firstMutation = result.current.mutateAsync({ canvasId: "canvas-1", lastVisitedTab: "console" });
      secondMutation = result.current.mutateAsync({ canvasId: "canvas-1", lastVisitedTab: "memory" });
    });

    firstWrite.reject(new Error("request failed"));
    await act(async () => {
      await expect(firstMutation).rejects.toThrow("request failed");
    });

    await waitFor(() => expect(canvasesUpdateCanvasPreference).toHaveBeenCalledTimes(2));

    secondWrite.resolve({ data: { preference: { lastVisitedTab: "memory" } } });
    await act(() => secondMutation);

    expect(queryClient.getQueryData(canvasKeys.preference("org-1", "canvas-1"))).toEqual({
      lastVisitedTab: "memory",
    });
  });
});
