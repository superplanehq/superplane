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

vi.mock("@/hooks/useCanvasData", async (importOriginal) => {
  const actual = await importOriginal();
  return {
    ...(actual as Record<string, unknown>),
    fetchCanvasConsoleData: fetchCanvasConsoleDataMock,
  };
});

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
  const onUnhandledRejection = (event: PromiseRejectionEvent) => {
    unhandledRejections.push(event.reason);
    event.preventDefault();
  };

  beforeEach(() => {
    vi.clearAllMocks();
    unhandledRejections = [];
    window.addEventListener("unhandledrejection", onUnhandledRejection);
  });

  afterEach(() => {
    window.removeEventListener("unhandledrejection", onUnhandledRejection);
  });

  it("loads baselines from committed snapshot reads", async () => {
    fetchCanvasVersionWithSpecMock.mockResolvedValue({ spec: { nodes: [] } });
    fetchCanvasConsoleDataMock.mockResolvedValue({ panels: [], layout: [] });

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

    await waitFor(() => {
      expect(result.current.ready).toBe(true);
    });
    expect(result.current.canvasSpec).toEqual({ nodes: [] });
    expect(result.current.console).toEqual({ panels: [], layout: [] });
  });

  it("does not surface a CancelledError as an unhandled rejection when queries are cancelled mid-flight", async () => {
    let resolveVersion: ((value: { spec: { nodes: never[] } }) => void) | undefined;
    fetchCanvasVersionWithSpecMock.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveVersion = resolve;
        }),
    );
    fetchCanvasConsoleDataMock.mockResolvedValue({ panels: [], layout: [] });

    const queryClient = createQueryClient();
    const { result, unmount } = renderHook(
      () =>
        useCommittedDraftBaselines({
          canvasId: "canvas-1",
          versionId: "version-1",
          enabled: true,
          stagingResetNonce: 0,
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(fetchCanvasVersionWithSpecMock).toHaveBeenCalled();
    });
    expect(result.current.ready).toBe(false);

    await queryClient.cancelQueries();
    unmount();
    resolveVersion?.({ spec: { nodes: [] } });

    await new Promise((resolve) => setTimeout(resolve, 20));

    expect(unhandledRejections).toEqual([]);
  });
});
