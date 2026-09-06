import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
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

  it("shows a disabled cancelling action while the run is stopping", () => {
    render(
      <MemoryRouter initialEntries={["/org-1/apps/child-canvas-id?run=child-run-id"]}>
        <RunInspectorHeader
          run={{
            ...baseRun,
            state: "STATE_CANCELLING",
            result: "RESULT_UNKNOWN",
          }}
          title="Child run"
          stepCount={1}
          organizationId="org-1"
          actionPending={false}
          actionDisabled
          onAction={() => {}}
        />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText("Cancelling")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancelling" })).toBeDisabled();
  });

  it("disables the action and uses the permission reason for the tooltip when provided", async () => {
    render(
      <MemoryRouter initialEntries={["/org-1/apps/child-canvas-id?run=child-run-id"]}>
        <RunInspectorHeader
          run={baseRun}
          title="Child run"
          stepCount={1}
          organizationId="org-1"
          actionPending={false}
          actionDisabled
          actionDisabledReason="You do not have permission to restart this run."
          onAction={() => {}}
        />
      </MemoryRouter>,
    );

    const button = screen.getByRole("button", { name: "Rerun" });
    expect(button).toBeDisabled();

    await userEvent.hover(button);
    expect((await screen.findAllByText("You do not have permission to restart this run.")).length).toBeGreaterThan(0);
  });

  it("falls back to the status-based tooltip when no disabled reason is provided", () => {
    renderHeader(baseRun);

    expect(screen.getByRole("button", { name: "Rerun" })).not.toBeDisabled();
  });
});
