import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasToolSidebarState } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import type { CanvasRunsSidebarState } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import type { CanvasVersionsSidebarState } from "@/components/CanvasVersionsSidebar/useCanvasVersionsSidebarState";

import { SecondaryHeaderActions } from "./HeaderSecondaryActions";

const runsSidebarState = {
  isRunsSidebarOpen: true,
  showRunsSidebarToggle: false,
  handleRunsSidebarToggle: vi.fn(),
  openRunsSidebar: vi.fn(),
  closeRunsSidebar: vi.fn(),
} satisfies CanvasRunsSidebarState;

const versionsSidebarState = {
  isVersionsSidebarOpen: false,
  showVersionsSidebarToggle: false,
  handleVersionsSidebarToggle: vi.fn(),
  openVersionsSidebar: vi.fn(),
  closeVersionsSidebar: vi.fn(),
} satisfies CanvasVersionsSidebarState;

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
        runsSidebarState={runsSidebarState}
        versionsSidebarState={versionsSidebarState}
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
        runsSidebarState={runsSidebarState}
        versionsSidebarState={versionsSidebarState}
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
        runsSidebarState={runsSidebarState}
        versionsSidebarState={versionsSidebarState}
      />,
    );

    expect(screen.getByTestId("canvas-commit-staging-button")).toBeDisabled();
    expect(screen.getByTestId("canvas-reset-staging-button")).toBeDisabled();
    expect(screen.queryByTestId("canvas-publish-version-button")).not.toBeInTheDocument();
  });

  it("keeps the Discard/Publish controls (disabled) during a publish even though it runs an embedded commit", () => {
    render(
      <SecondaryHeaderActions
        canvasName="Canvas"
        mode="version-edit"
        isEditing
        hasStagingChanges={false}
        commitStagingPending
        publishVersionPending
        onCommitStaging={vi.fn()}
        onResetStaging={vi.fn()}
        onDiscardVersion={vi.fn()}
        onPublishVersion={vi.fn()}
        publishVersionDisabled
        toolSidebarState={{} as CanvasToolSidebarState}
        runsSidebarState={runsSidebarState}
        versionsSidebarState={versionsSidebarState}
      />,
    );

    const publishButton = screen.getByTestId("canvas-publish-version-button");
    expect(publishButton).toBeInTheDocument();
    expect(publishButton).toBeDisabled();
    expect(screen.getByText("Discard")).toBeDisabled();
    expect(screen.queryByTestId("canvas-commit-staging-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("canvas-reset-staging-button")).not.toBeInTheDocument();
  });
});
