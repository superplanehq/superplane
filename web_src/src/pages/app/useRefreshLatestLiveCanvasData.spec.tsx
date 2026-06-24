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

describe("useRefreshLatestLiveCanvasData", () => {
  it("invalidates both the version-specific and live console cache keys after publish", async () => {
    const queryClient = new QueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useRefreshLatestLiveCanvasData("org-1", "canvas-1", "old-live-version"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current({ liveVersionId: "published-version", skipDraftBranchRefetch: true });
    });

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.console("canvas-1", "published-version"),
      exact: true,
      refetchType: "all",
    });
    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: canvasKeys.console("canvas-1", undefined),
      exact: true,
      refetchType: "all",
    });
  });
});
