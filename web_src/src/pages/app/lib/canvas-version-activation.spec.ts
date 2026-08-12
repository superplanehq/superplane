import { describe, expect, it, vi } from "vitest";

import { activateCanvasVersionForEditing } from "./canvas-version-activation";

describe("activateCanvasVersionForEditing", () => {
  it("clears run inspection params when activating the live version for edit", () => {
    const setSearchParams = vi.fn();

    activateCanvasVersionForEditing({
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionID: "version-live",
      version: { metadata: { id: "version-live" }, spec: {} },
      options: { preserveStagedLayer: true },
      liveCanvasVersionId: "version-live",
      queryClient: {
        cancelQueries: vi.fn(),
      } as never,
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

  it("returns the same searchParams instance when activation is a no-op rewrite", () => {
    const setSearchParams = vi.fn();
    const current = new URLSearchParams("configure=1&from=automations");

    activateCanvasVersionForEditing({
      organizationId: "org-1",
      canvasId: "canvas-1",
      versionID: "version-live",
      version: { metadata: { id: "version-live" }, spec: {} },
      options: { preserveStagedLayer: true },
      liveCanvasVersionId: "version-live",
      queryClient: {
        cancelQueries: vi.fn(),
      } as never,
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
    expect(updater(current)).toBe(current);
  });
});
