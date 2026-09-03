import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { RunExecutionNodeRow } from "./RunExecutionNodeRow";

function makeExecution(overrides: Partial<CanvasesCanvasNodeExecution> = {}): CanvasesCanvasNodeExecution {
  return {
    id: "exec-current",
    nodeId: "action-1",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    ...overrides,
  };
}

describe("RunExecutionNodeRow", () => {
  it("names the source execution on the replay badge of a replay execution", () => {
    render(
      <RunExecutionNodeRow
        nodeId="action-1"
        componentIconMap={{}}
        execution={makeExecution({ isReplay: true, replaySourceExecutionId: "exec-original-1" })}
        isTrigger={false}
        isSelected={false}
        onSelect={vi.fn()}
      />,
    );

    const badge = screen.getByTestId("replay-badge");
    expect(badge).toHaveTextContent(/replay/i);
    expect(badge).toHaveAttribute("title", expect.stringContaining("exec-original-1"));
  });

  it("shows no replay badge for a non-replay execution", () => {
    render(
      <RunExecutionNodeRow
        nodeId="action-1"
        componentIconMap={{}}
        execution={makeExecution({ isReplay: false })}
        isTrigger={false}
        isSelected={false}
        onSelect={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("replay-badge")).not.toBeInTheDocument();
  });

  it("shows the badge without a source execution when the run is a replay but the source id is unknown (boundary)", () => {
    render(
      <RunExecutionNodeRow
        nodeId="action-1"
        componentIconMap={{}}
        execution={makeExecution({ isReplay: true, replaySourceExecutionId: "" })}
        isTrigger={false}
        isSelected={false}
        onSelect={vi.fn()}
      />,
    );

    const badge = screen.getByTestId("replay-badge");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveAttribute("title", "Replay");
  });

  it("names each row's own source execution when several replay rows are rendered side by side", () => {
    render(
      <>
        <RunExecutionNodeRow
          nodeId="action-1"
          componentIconMap={{}}
          execution={makeExecution({ isReplay: true, replaySourceExecutionId: "exec-original-1" })}
          isTrigger={false}
          isSelected={false}
          onSelect={vi.fn()}
        />
        <RunExecutionNodeRow
          nodeId="action-2"
          componentIconMap={{}}
          execution={makeExecution({
            id: "exec-current-2",
            nodeId: "action-2",
            isReplay: true,
            replaySourceExecutionId: "exec-original-2",
          })}
          isTrigger={false}
          isSelected={false}
          onSelect={vi.fn()}
        />
      </>,
    );

    const [firstRow, secondRow] = screen.getAllByTestId("run-execution-node-row");

    const firstBadge = within(firstRow).getByTestId("replay-badge");
    expect(firstBadge.getAttribute("title")).toContain("exec-original-1");
    expect(firstBadge.getAttribute("title")).not.toContain("exec-original-2");

    const secondBadge = within(secondRow).getByTestId("replay-badge");
    expect(secondBadge.getAttribute("title")).toContain("exec-original-2");
    expect(secondBadge.getAttribute("title")).not.toContain("exec-original-1");
  });
});
