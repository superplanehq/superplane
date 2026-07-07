import { describe, expect, it, vi } from "vitest";

import { activateCanvasVersionForEditing } from "./canvas-version-activation";

describe("activateCanvasVersionForEditing", () => {
  it("clears run inspection params when activating a version", () => {
    const setSearchParams = vi.fn();
    const queryClient = {
      cancelQueries: vi.fn(),
      invalidateQueries: vi.fn().mockResolvedValue(undefined),
      setQueryData: vi.fn(),
    };

    activateCanvasVersionForEditing({
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionID: "version-live",
      version: { metadata: { id: "version-live" }, spec: {} },
      effectiveLiveCanvasVersionId: "version-live",
      liveCanvasVersionId: "version-live",
      queryClient: queryClient as never,
      draftCanvasSpec: null,
      draftCanvasSpecsRef: { current: new Map() },
      activeCanvasVersionIdRef: { current: "" },
      lastAppliedVersionSnapshotRef: { current: "" },
      clearPendingAutoSaveWork: vi.fn(),
      setDraftCanvasSpec: vi.fn(),
      setActiveCanvasVersion: vi.fn(),
      setLastSavedWorkflowSnapshot: vi.fn(),
      setSearchParams,
      initializeFromWorkflow: vi.fn(),
    });

    const updater = setSearchParams.mock.calls[0]?.[0] as (current: URLSearchParams) => URLSearchParams;
    const next = updater(
      new URLSearchParams({
        run: "run-42",
        sidebar: "1",
        node: "node-1",
        version: "old-version",
      }),
    );

    expect(next.get("run")).toBeNull();
    expect(next.get("sidebar")).toBeNull();
    expect(next.get("node")).toBeNull();
    expect(next.get("version")).toBeNull();
  });
});
