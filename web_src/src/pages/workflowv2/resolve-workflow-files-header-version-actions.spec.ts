import { describe, expect, it, vi } from "vitest";

import { resolveWorkflowFilesHeaderVersionActions } from "./lib/resolve-workflow-files-header-version-actions";

describe("resolveWorkflowFilesHeaderVersionActions", () => {
  it("passes Commit/Publish label through in builder edit mode", () => {
    const handlePublishVersion = vi.fn();

    const commitActions = resolveWorkflowFilesHeaderVersionActions({
      useFilesHeaderActions: false,
      filesHeaderActions: null,
      isChangeManagementDisabled: true,
      handlePublishVersion,
      handleCreateChangeRequest: vi.fn(),
      handleResetDraftChanges: vi.fn(),
      publishVersionDisabled: false,
      resetDraftDisabled: false,
      hasUnpublishedDraftChanges: true,
      publishVersionLabel: "Commit",
    });

    expect(commitActions.publishVersionLabel).toBe("Commit");
    expect(commitActions.onPublishVersion).toBe(handlePublishVersion);

    const publishActions = resolveWorkflowFilesHeaderVersionActions({
      useFilesHeaderActions: false,
      filesHeaderActions: null,
      isChangeManagementDisabled: true,
      handlePublishVersion,
      handleCreateChangeRequest: vi.fn(),
      handleResetDraftChanges: vi.fn(),
      publishVersionDisabled: false,
      resetDraftDisabled: false,
      hasUnpublishedDraftChanges: false,
      publishVersionLabel: "Publish",
    });

    expect(publishActions.publishVersionLabel).toBe("Publish");
  });

  it("exposes the contextual reset/discard label and keeps the button visible in builder edit mode", () => {
    const resetActions = resolveWorkflowFilesHeaderVersionActions({
      useFilesHeaderActions: false,
      filesHeaderActions: null,
      isChangeManagementDisabled: true,
      handlePublishVersion: vi.fn(),
      handleCreateChangeRequest: vi.fn(),
      handleResetDraftChanges: vi.fn(),
      publishVersionDisabled: false,
      resetDraftDisabled: false,
      resetDraftLabel: "Reset",
      hasUnpublishedDraftChanges: true,
      publishVersionLabel: "Commit",
    });

    expect(resetActions.discardVersionLabel).toBe("Reset");
    expect(resetActions.discardVersionVisible).toBe(true);

    const discardActions = resolveWorkflowFilesHeaderVersionActions({
      useFilesHeaderActions: false,
      filesHeaderActions: null,
      isChangeManagementDisabled: true,
      handlePublishVersion: vi.fn(),
      handleCreateChangeRequest: vi.fn(),
      handleResetDraftChanges: vi.fn(),
      publishVersionDisabled: false,
      resetDraftDisabled: false,
      resetDraftLabel: "Discard",
      hasUnpublishedDraftChanges: false,
      publishVersionLabel: "Publish",
    });

    expect(discardActions.discardVersionLabel).toBe("Discard");
    // Always visible while editing so the draft can be discarded even with a clean staging area.
    expect(discardActions.discardVersionVisible).toBe(true);
  });

  it("uses Publish for files header actions regardless of builder label", () => {
    const actions = resolveWorkflowFilesHeaderVersionActions({
      useFilesHeaderActions: true,
      filesHeaderActions: {
        onPublish: vi.fn(),
        onDiscardAll: vi.fn(),
        publishDisabled: false,
        discardDisabled: false,
        hasPendingChanges: true,
        publishPending: false,
      },
      isChangeManagementDisabled: true,
      handlePublishVersion: vi.fn(),
      handleCreateChangeRequest: vi.fn(),
      handleResetDraftChanges: vi.fn(),
      publishVersionDisabled: false,
      resetDraftDisabled: false,
      hasUnpublishedDraftChanges: true,
      publishVersionLabel: "Commit",
    });

    expect(actions.publishVersionLabel).toBe("Publish");
    expect(actions.hasUnpublishedDraftChanges).toBe(true);
  });
});
