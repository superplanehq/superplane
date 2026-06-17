import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";

import type { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";

const { fetchCanvasVersionWithSpecMock, syncCommittedCanvasDraftStateMock } = vi.hoisted(() => ({
  fetchCanvasVersionWithSpecMock: vi.fn(),
  syncCommittedCanvasDraftStateMock: vi.fn(),
}));

vi.mock("@/pages/app/lib/repository-spec-files", () => ({
  fetchCanvasVersionWithSpec: fetchCanvasVersionWithSpecMock,
}));

vi.mock("@/pages/app/lib/sync-committed-canvas-draft", () => ({
  syncCommittedCanvasDraftState: syncCommittedCanvasDraftStateMock,
}));

import { useCanvasDraftResync } from "./useCanvasDraftResync";

type CanvasSpec = CanvasesCanvas["spec"] | null;

function makeRef<T>(value: T): MutableRefObject<T> {
  return { current: value };
}

function renderResyncHook(activeVersionId = "version-1") {
  const queryClient = new QueryClient();
  const setDraftCanvasSpec: Dispatch<SetStateAction<CanvasSpec>> = vi.fn();
  const setActiveCanvasVersion: Dispatch<SetStateAction<CanvasesCanvasVersion | null>> = vi.fn();
  const setLastSavedWorkflowSnapshot = vi.fn();
  const setHasUnsavedChanges: Dispatch<SetStateAction<boolean>> = vi.fn();
  const setHasNonPositionalUnsavedChanges: Dispatch<SetStateAction<boolean>> = vi.fn();
  const setStagingResetNonce: Dispatch<SetStateAction<number>> = vi.fn();

  const { result } = renderHook(
    () =>
      useCanvasDraftResync({
        organizationId: "org-1",
        canvasId: "canvas-1",
        activeCanvasVersionIdRef: makeRef(activeVersionId),
        draftCanvasSpecsRef: makeRef(new Map<string, CanvasSpec>()),
        consoleMutationGenerationRef: makeRef(0),
        setDraftCanvasSpec,
        setActiveCanvasVersion,
        setLastSavedWorkflowSnapshot,
        setHasUnsavedChanges,
        setHasNonPositionalUnsavedChanges,
        setStagingResetNonce,
      }),
    {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    },
  );

  return result;
}

describe("useCanvasDraftResync", () => {
  beforeEach(() => {
    fetchCanvasVersionWithSpecMock.mockReset();
    syncCommittedCanvasDraftStateMock.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // Reproduces the Sentry "Non-Error promise rejection captured with value: Not Found"
  // case: the OpenAPI client throws the raw response text (a string, not an Error)
  // when the targeted version was deleted between the websocket event and the resync.
  it("resyncDraftToCommitted swallows a stale-id rejection so it does not leak as an unhandled rejection", async () => {
    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    syncCommittedCanvasDraftStateMock.mockRejectedValue("Not Found");

    const result = renderResyncHook();

    await expect(result.current.resyncDraftToCommitted("version-1")).resolves.toBeUndefined();

    expect(consoleSpy).toHaveBeenCalledWith("[useCanvasDraftResync] resyncDraftToCommitted failed", "Not Found");
  });

  it("resyncDraftToStaged swallows a stale-id rejection so it does not leak as an unhandled rejection", async () => {
    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    fetchCanvasVersionWithSpecMock.mockRejectedValue("Not Found");

    const result = renderResyncHook();

    await expect(result.current.resyncDraftToStaged("version-1")).resolves.toBeUndefined();

    expect(consoleSpy).toHaveBeenCalledWith("[useCanvasDraftResync] resyncDraftToStaged failed", "Not Found");
  });
});
