import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { formatClockDuration } from "@/lib/duration";
import type { ExecutionInfo } from "@/pages/app/mappers/types";
import { RunnerLiveLogDialog } from "./RunnerLiveLogDialog";

const phaseLogCardMock = vi.fn(
  (props: {
    organizationId?: string;
    canvasId?: string;
    phase: { name: string; stream: Array<{ component?: string; executionId?: string }> };
  }) => (
    <div data-testid="phase-log-card">
      {props.phase.name} {props.organizationId} {props.canvasId}
    </div>
  ),
);

vi.mock("@/pages/factories/pages/work-order-split-run/PhaseLogCard", () => ({
  PhaseLogCard: (props: {
    organizationId?: string;
    canvasId?: string;
    phase: { name: string; stream: Array<{ component?: string; executionId?: string }> };
  }) => phaseLogCardMock(props),
}));

const startedAt = Date.now() - 90_000;
const execution = {
  id: "execution-1",
  state: "STATE_FINISHED",
  createdAt: new Date(startedAt).toISOString(),
  updatedAt: new Date(startedAt + 90_000).toISOString(),
} as ExecutionInfo;

describe("RunnerLiveLogDialog", () => {
  it("hides the logs link when the node has no execution", () => {
    render(<RunnerLiveLogDialog title="Run Shell Command" canvasMode="live" execution={null} />);

    expect(screen.queryByRole("button", { name: "See logs" })).not.toBeInTheDocument();
  });

  it("hides the logs link in edit mode", () => {
    render(<RunnerLiveLogDialog title="Run Shell Command" canvasMode="edit" execution={execution} />);

    expect(screen.queryByRole("button", { name: "See logs" })).not.toBeInTheDocument();
  });

  it("opens logs from an icon button with a See logs tooltip", async () => {
    render(
      <RunnerLiveLogDialog
        title="Run Shell Command"
        canvasMode="live"
        execution={execution}
        component="runnerBash"
        session={{ organizationId: "org-1", canvasId: "canvas-1" }}
      />,
    );

    const openLogs = screen.getByRole("button", { name: "See logs" });
    expect(openLogs).not.toHaveTextContent("See logs");

    fireEvent.click(openLogs);

    await waitFor(() => {
      expect(screen.getByTestId("phase-log-card")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Go back" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Close" })).not.toBeInTheDocument();
    expect(phaseLogCardMock).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: "org-1",
        canvasId: "canvas-1",
        expanded: true,
        collapsible: false,
        phase: expect.objectContaining({
          name: "Run Shell Command",
          duration: formatClockDuration(90_000),
          stream: [
            expect.objectContaining({
              component: "runnerBash",
              executionId: "execution-1",
            }),
          ],
        }),
      }),
    );
  });
});
