import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import { RunInspectorHeader } from "./RunInspectorHeader";

const baseRun: CanvasesCanvasRun = {
  id: "child-run-id",
  canvasId: "child-canvas-id",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  createdAt: "2026-05-01T12:00:00Z",
};

function renderHeader(run: CanvasesCanvasRun) {
  return render(
    <MemoryRouter initialEntries={["/org-1/apps/child-canvas-id?run=child-run-id"]}>
      <RunInspectorHeader
        run={run}
        title="Child run"
        stepCount={1}
        organizationId="org-1"
        actionPending={false}
        actionDisabled={false}
        onAction={() => {}}
      />
    </MemoryRouter>,
  );
}

describe("RunInspectorHeader", () => {
  it("shows a link to the parent run when parent ref is present", () => {
    renderHeader({
      ...baseRun,
      parent: {
        id: "parent-run-id",
        canvasId: "parent-canvas-id",
        state: "STATE_STARTED",
      },
    });

    const link = screen.getByRole("link", { name: "See parent" });
    expect(link).toHaveAttribute("href", "/org-1/apps/parent-canvas-id?run=parent-run-id");
  });

  it("hides the parent run link when parent ref is missing", () => {
    renderHeader(baseRun);

    expect(screen.queryByRole("link", { name: "See parent" })).not.toBeInTheDocument();
  });
});
