import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";

import { SecondaryHeaderActions } from "./HeaderSecondaryActions";

describe("SecondaryHeaderActions", () => {
  it("shows the console diff badge while editing console changes", () => {
    render(
      <SecondaryHeaderActions
        canvasName="Canvas"
        mode="console"
        isEditing
        hasUnpublishedDraftChanges
        hasUnpublishedConsoleDraftChanges
        draftConsoleDiff={{ diffCounts: { added: 1, updated: 0, removed: 0 } }}
        onShowConsoleDiff={vi.fn()}
        toolSidebarState={{} as CanvasToolSidebarState}
      />,
    );

    expect(screen.getByText("+1")).toBeInTheDocument();
  });

  it("shows the canvas diff badge for uncommitted (staged) changes", () => {
    render(
      <SecondaryHeaderActions
        canvasName="Canvas"
        mode="version-edit"
        isEditing
        hasUncommittedCanvasDraftChanges
        draftVisualDiff={{
          diffCounts: { added: 1, updated: 0, removed: 0 },
          diffToggles: {
            showDeletedNodes: false,
            toggleShowDeletedNodes: vi.fn(),
            showEdgeDiff: false,
            toggleShowEdgeDiff: vi.fn(),
          },
        }}
        onShowDiff={vi.fn()}
        toolSidebarState={{} as CanvasToolSidebarState}
      />,
    );

    expect(screen.getByText("+1")).toBeInTheDocument();
  });

  it("keeps the commit controls pending while a commit settles after staging clears", () => {
    render(
      <SecondaryHeaderActions
        canvasName="Canvas"
        mode="version-edit"
        isEditing
        hasStagingChanges={false}
        commitStagingPending
        onCommitStaging={vi.fn()}
        onResetStaging={vi.fn()}
        onPublishVersion={vi.fn()}
        toolSidebarState={{} as CanvasToolSidebarState}
      />,
    );

    expect(screen.getByTestId("canvas-commit-staging-button")).toBeDisabled();
    expect(screen.getByTestId("canvas-reset-staging-button")).toBeDisabled();
    expect(screen.queryByTestId("canvas-publish-version-button")).not.toBeInTheDocument();
  });
});
