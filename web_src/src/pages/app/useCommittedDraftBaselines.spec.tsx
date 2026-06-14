import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { fetchCanvasVersionWithSpecMock, fetchCanvasConsoleDataMock } = vi.hoisted(() => ({
  fetchCanvasVersionWithSpecMock: vi.fn(),
  fetchCanvasConsoleDataMock: vi.fn(),
}));

vi.mock("./lib/repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: fetchCanvasVersionWithSpecMock,
}));

vi.mock("@/hooks/useCanvasData", () => ({
  fetchCanvasConsoleData: fetchCanvasConsoleDataMock,
  canvasKeys: {
    versionDetail: (canvasId: string, versionId: string) => ["versionDetail", canvasId, versionId],
    console: (canvasId: string, versionId: string) => ["console", canvasId, versionId],
  },
}));

import { useCommittedDraftBaselines } from "./useCommittedDraftBaselines";

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

describe("useCommittedDraftBaselines", () => {
  let unhandledRejections: unknown[];
  let rejectionHandler: (event: PromiseRejectionEvent) => void;

  beforeEach(() => {
    unhandledRejections = [];
    rejectionHandler = (event: PromiseRejectionEvent) => {
      unhandledRejections.push(event.reason);
    };
    window.addEventListener("unhandledrejection", rejectionHandler);
  });

  afterEach(() => {
    window.removeEventListener("unhandledrejection", rejectionHandler);
    fetchCanvasVersionWithSpecMock.mockReset();
    fetchCanvasConsoleDataMock.mockReset();
  });

  it("returns ready baselines populated from committed reads", async () => {
    fetchCanvasVersionWithSpecMock.mockResolvedValue({
      metadata: { id: "version-1" },
      spec: { nodes: [], edges: [] },
    });
    fetchCanvasConsoleDataMock.mockResolvedValue({ panels: [], layout: [], consoleYaml: "" });

    const queryClient = createQueryClient();

    const { result } = renderHook(
      () =>
        useCommittedDraftBaselines({
          canvasId: "canvas-1",
          versionId: "version-1",
          enabled: true,
          stagingResetNonce: 0,
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.ready).toBe(true));
    expect(result.current.canvasSpec).toEqual({ nodes: [], edges: [] });
    expect(result.current.console).toEqual({ panels: [], layout: [] });
    expect(unhandledRejections).toEqual([]);
  });

  it("still marks baselines ready when the committed reads fail (no unhandled rejection)", async () => {
    fetchCanvasVersionWithSpecMock.mockRejectedValue(new Error("Failed to get file"));
    fetchCanvasConsoleDataMock.mockRejectedValue(new Error("Failed to get file"));

    const queryClient = createQueryClient();

    const { result } = renderHook(
      () =>
        useCommittedDraftBaselines({
          canvasId: "canvas-1",
          versionId: "version-1",
          enabled: true,
          stagingResetNonce: 0,
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => expect(result.current.ready).toBe(true));

    // Give the rejected promises one more macrotask to flush, then assert that
    // none of them surfaced as an unhandled rejection on window.
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(result.current.canvasSpec).toBeUndefined();
    expect(result.current.console).toEqual({ panels: [], layout: [] });
    expect(unhandledRejections).toEqual([]);
  });
});
