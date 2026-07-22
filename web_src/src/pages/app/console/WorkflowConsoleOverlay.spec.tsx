import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { CanvasConsoleQueryResult, UpdateCanvasConsoleMutationResult } from "@/hooks/useCanvasData";

import { WorkflowConsoleOverlay } from "./WorkflowConsoleOverlay";

vi.mock("./ConsoleOverlay", () => ({
  ConsoleOverlay: ({
    canRunNodes,
    runNodesDisabledReason,
  }: {
    canRunNodes: boolean;
    runNodesDisabledReason?: string;
  }) => (
    <div
      data-testid="console-run-state"
      data-can-run={String(canRunNodes)}
      data-disabled-reason={runNodesDisabledReason}
    />
  ),
}));

const requiredProps = {
  isConsoleMode: true,
  editLocked: false,
  consoleQuery: {} as CanvasConsoleQueryResult,
  updateConsoleMutation: {} as UpdateCanvasConsoleMutationResult,
  addPanelDialogOpen: false,
  onAddPanelDialogOpenChange: vi.fn(),
  yamlModalOpen: false,
  onYamlModalOpenChange: vi.fn(),
};

describe("WorkflowConsoleOverlay runtime actions", () => {
  it("keeps actions enabled when there are no uncommitted canvas changes", () => {
    render(<WorkflowConsoleOverlay {...requiredProps} canActOnCanvas hasUncommittedCanvasDraftChanges={false} />);

    const state = screen.getByTestId("console-run-state");
    expect(state).toHaveAttribute("data-can-run", "true");
    expect(state).not.toHaveAttribute("data-disabled-reason");
  });

  it("blocks actions when the rendered canvas differs from the live canvas", () => {
    render(<WorkflowConsoleOverlay {...requiredProps} canActOnCanvas hasUncommittedCanvasDraftChanges />);

    const state = screen.getByTestId("console-run-state");
    expect(state).toHaveAttribute("data-can-run", "false");
    expect(state).toHaveAttribute("data-disabled-reason", "uncommitted-canvas-changes");
  });
});
