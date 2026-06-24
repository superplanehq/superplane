import { describe, expect, it, vi } from "vitest";

import { processCanvasLifecycleEvent, shouldReactToCanvasVersionUpdated } from "./canvas-version-lifecycle";

describe("shouldReactToCanvasVersionUpdated", () => {
  it("ignores remote updates on a passive live-view tab", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-1",
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(false);
  });

  it("ignores version-less updates when the versions UI is closed", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: undefined,
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(false);
  });

  it("reacts when editing the affected draft", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-1",
        activeCanvasVersionId: "draft-1",
        isEditing: true,
        editSessionActive: true,
      }),
    ).toBe(true);
  });

  it("reacts when previewing the affected version", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "version-1",
        activeCanvasVersionId: "version-1",
        isEditing: false,
        editSessionActive: false,
      }),
    ).toBe(true);
  });

  it("reacts when the versions sidebar is open even for another draft", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: "draft-2",
        activeCanvasVersionId: "draft-1",
        isEditing: true,
        editSessionActive: true,
      }),
    ).toBe(true);
  });

  it("reacts to version-less updates while the versions UI is open", () => {
    expect(
      shouldReactToCanvasVersionUpdated({
        versionId: undefined,
        activeCanvasVersionId: "",
        isEditing: false,
        editSessionActive: true,
      }),
    ).toBe(true);
  });
});

describe("processCanvasLifecycleEvent", () => {
  it("prunes deleted draft cache on passive live-view tabs without lifecycle invalidation", () => {
    const pruneDeletedCanvasVersion = vi.fn();
    const resyncDraftToCommitted = vi.fn();
    const invalidateCanvasVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1", versionId: "draft-1" },
      eventName: "canvas_version_deleted",
      canvasId: "canvas-1",
      activeCanvasVersionId: "",
      isEditing: false,
      editSessionActive: false,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      consumeIgnoredCreateDraftEcho: () => false,
      consumeIgnoredCanvasVersionUpdatedEcho: () => false,
      invalidateCanvasVersionData,
      pruneDeletedCanvasVersion,
      resyncDraftToCommitted,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(false);
    expect(pruneDeletedCanvasVersion).toHaveBeenCalledWith("draft-1");
    expect(resyncDraftToCommitted).not.toHaveBeenCalled();
    expect(invalidateCanvasVersionData).not.toHaveBeenCalled();
  });

  it("prunes deleted draft cache and allows lifecycle invalidation when versions UI is open", () => {
    const pruneDeletedCanvasVersion = vi.fn();
    const resyncDraftToCommitted = vi.fn();
    const invalidateCanvasVersionData = vi.fn();

    const result = processCanvasLifecycleEvent({
      payload: { canvasId: "canvas-1", versionId: "draft-1" },
      eventName: "canvas_version_deleted",
      canvasId: "canvas-1",
      activeCanvasVersionId: "",
      isEditing: false,
      editSessionActive: true,
      hasLocalSaveActivity: false,
      consumeIgnoredCanvasUpdatedEcho: () => false,
      consumeIgnoredCreateDraftEcho: () => false,
      consumeIgnoredCanvasVersionUpdatedEcho: () => false,
      invalidateCanvasVersionData,
      pruneDeletedCanvasVersion,
      resyncDraftToCommitted,
      setCanvasDeletedRemotely: vi.fn(),
      setRemoteCanvasUpdatePending: vi.fn(),
    });

    expect(result).toBe(true);
    expect(pruneDeletedCanvasVersion).toHaveBeenCalledWith("draft-1");
    expect(resyncDraftToCommitted).not.toHaveBeenCalled();
    expect(invalidateCanvasVersionData).not.toHaveBeenCalled();
  });
});
