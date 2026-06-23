import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useDraftRecovery } from "./useDraftRecovery";

const { ensureDraftVersionExists } = vi.hoisted(() => ({
  ensureDraftVersionExists: vi.fn(),
}));

const { showSuccessToast } = vi.hoisted(() => ({
  showSuccessToast: vi.fn(),
}));

vi.mock("@/hooks/useCanvasData", () => ({
  ensureDraftVersionExists,
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showInfoToast: vi.fn(),
  showSuccessToast,
}));

vi.mock("@/lib/errors", () => ({
  getApiErrorMessage: vi.fn((_: unknown, fallback: string) => fallback),
}));

vi.mock("@/lib/usageLimits", () => ({
  getUsageLimitToastMessage: vi.fn((error: unknown) => String(error)),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useDraftRecovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    ensureDraftVersionExists.mockResolvedValue(true);
  });

  it("publishes the committed draft after flushing pending saves without committing staging", async () => {
    const ensureVersionActionDraftReady = vi.fn().mockResolvedValue(true);
    const publishCanvasVersionMutation = { mutateAsync: vi.fn().mockResolvedValue({}) };
    const setIsPreparingVersionAction = vi.fn();
    const setSearchParams = vi.fn();
    const refreshLatestLiveCanvasData = vi.fn().mockResolvedValue(undefined);
    const activeCanvasVersionIdRef = { current: "draft-1" };

    const { result } = renderHook(
      () =>
        useDraftRecovery({
          organizationId: "org-1",
          canvasId: "canvas-1",
          activeCanvasVersionId: "draft-1",
          activeCanvasVersionIdRef,
          draftCanvasSpecsRef: { current: new Map([["draft-1", { nodes: [], edges: [] }]]) },
          setActiveCanvasVersion: vi.fn(),
          setDraftCanvasSpec: vi.fn(),
          exitToLive: vi.fn(),
          setSearchParams,
          refreshLatestLiveCanvasData,
          cancelPendingCanvasSaves: vi.fn(),
          ensureVersionActionDraftReady,
          publishCanvasVersionMutation,
          setIsPreparingVersionAction,
        }),
      { wrapper: createWrapper() },
    );

    await act(async () => {
      await result.current.handlePublishVersion();
    });

    expect(ensureVersionActionDraftReady).toHaveBeenCalledWith(
      "Unable to prepare the latest version changes for publishing",
    );
    expect(ensureDraftVersionExists).toHaveBeenCalledWith(expect.any(QueryClient), "org-1", "canvas-1", "draft-1");
    expect(publishCanvasVersionMutation.mutateAsync).toHaveBeenCalledWith("draft-1");
    expect(refreshLatestLiveCanvasData).toHaveBeenCalledTimes(1);
    expect(showSuccessToast).toHaveBeenCalledWith("Version published");
    expect(setIsPreparingVersionAction).toHaveBeenNthCalledWith(1, true);
    expect(setIsPreparingVersionAction).toHaveBeenLastCalledWith(false);
  });

  it("does not publish when pending saves fail to settle", async () => {
    const ensureVersionActionDraftReady = vi.fn().mockResolvedValue(false);
    const publishCanvasVersionMutation = { mutateAsync: vi.fn().mockResolvedValue({}) };
    const setIsPreparingVersionAction = vi.fn();
    const activeCanvasVersionIdRef = { current: "draft-1" };

    const { result } = renderHook(
      () =>
        useDraftRecovery({
          organizationId: "org-1",
          canvasId: "canvas-1",
          activeCanvasVersionId: "draft-1",
          activeCanvasVersionIdRef,
          draftCanvasSpecsRef: { current: new Map([["draft-1", { nodes: [], edges: [] }]]) },
          setActiveCanvasVersion: vi.fn(),
          setDraftCanvasSpec: vi.fn(),
          exitToLive: vi.fn(),
          setSearchParams: vi.fn(),
          refreshLatestLiveCanvasData: vi.fn(),
          cancelPendingCanvasSaves: vi.fn(),
          ensureVersionActionDraftReady,
          publishCanvasVersionMutation,
          setIsPreparingVersionAction,
        }),
      { wrapper: createWrapper() },
    );

    await act(async () => {
      await result.current.handlePublishVersion();
    });

    expect(publishCanvasVersionMutation.mutateAsync).not.toHaveBeenCalled();
    expect(showSuccessToast).not.toHaveBeenCalled();
    expect(setIsPreparingVersionAction).toHaveBeenLastCalledWith(false);
  });
});
