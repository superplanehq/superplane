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
});
