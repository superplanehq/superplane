import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";
import type { SplitRunStreamLine } from "./splitRunMocks";

const useLiveLogStreamMock = vi.fn();

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: (...args: unknown[]) => useLiveLogStreamMock(...args),
}));

beforeEach(() => {
  useLiveLogStreamMock.mockReturnValue(idleLiveLogStream(vi.fn()));
});

describe("PhaseLogCard node line", () => {
  const NODE_STREAM: SplitRunStreamLine[] = [
    line({
      id: "planner-agent",
      componentName: "Agent - Plan for GH Issue",
      componentType: "Run Claude Code",
      at: "12:24:02",
      duration: "1m 20s",
    }),
  ];

  it("shows the node name and puts status plus time on the far right", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(within(row).getByText("Agent - Plan for GH Issue")).toBeInTheDocument();
    expect(within(row).queryByText("passed")).not.toBeInTheDocument();
    expect(within(row).queryByText(">")).not.toBeInTheDocument();
    expect(within(row).queryByText("12:24:02")).not.toBeInTheDocument();
    expect(within(row).queryByText("Run Claude Code")).not.toBeInTheDocument();

    const statusTime = within(row).getByTestId("split-run-stream-duration-planner-agent");
    expect(statusTime).toHaveAccessibleName("Passed");
    expect(statusTime).toHaveTextContent("✓");
    expect(statusTime).toHaveTextContent("01:20");
    expect(statusTime).not.toHaveTextContent("Passed");
    expect(statusTime.className).not.toMatch(/bg-/);
    expect(statusTime.parentElement?.className).toMatch(/ml-auto/);
    expect(statusTime.className).toMatch(/text-right/);
    expect(statusTime.className).toMatch(/font-mono/);
    expect(statusTime.className).toMatch(/text-\[14px\]/);
    expect(screen.getByTestId("split-run-stream-plan").className).toMatch(/font-mono/);
  });

  it("floats the produced artifact next to the duration", () => {
    render(
      <PhaseLogCard
        phase={PHASE}
        expanded
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            duration: "1m 20s",
            artifact: {
              id: "art-plan",
              type: "TYPE_MARKDOWN",
              data: { name: "PLAN.md", title: "PLAN.md" },
            },
          }),
        ]}
      />,
    );

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    const artifact = within(row).getByRole("button", { name: "PLAN.md" });
    const duration = within(row).getByTestId("split-run-stream-duration-planner-agent");

    expect(artifact.parentElement).toBe(duration.parentElement);
    expect(artifact.parentElement?.className).toMatch(/ml-auto/);
    expect(artifact.compareDocumentPosition(duration) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("styles the node line as a table header without a caret", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(row.querySelector(".lucide-chevron-right")).toBeNull();
    expect(row.className).toMatch(/\bbg-muted\b/);
    expect(row.className).not.toMatch(/\bbg-background\b/);
    expect(row.className).not.toMatch(/hover:bg-/);
    expect(row.className).not.toMatch(/border-b/);
    expect(row.className).not.toMatch(/-mx-2/);
    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/size-3/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
