import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasRun } from "@/api-client";
import { RunInspectorHeader } from "./RunInspectorHeader";

const baseRun: CanvasesCanvasRun = {
  id: "child-run-id",
  canvasId: "child-canvas-id",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  createdAt: "2026-05-01T12:00:00Z",
};

function renderHeader(run: CanvasesCanvasRun, onAskAgentAboutEvent?: () => void) {
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
        onAskAgentAboutEvent={onAskAgentAboutEvent}
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

  it("shows and invokes the agent event action when a root event is present", () => {
    const onAskAgentAboutEvent = vi.fn();

    renderHeader(
      {
        ...baseRun,
        rootEvent: { id: "root-event-id" },
      },
      onAskAgentAboutEvent,
    );

    fireEvent.click(screen.getByRole("button", { name: "Ask agent about this event" }));

    expect(onAskAgentAboutEvent).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Rerun" })).toBeInTheDocument();
  });

  it("hides the agent event action when the run has no root event", () => {
    renderHeader(baseRun, vi.fn());

    expect(screen.queryByRole("button", { name: "Ask agent about this event" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Rerun" })).toBeInTheDocument();
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
});
