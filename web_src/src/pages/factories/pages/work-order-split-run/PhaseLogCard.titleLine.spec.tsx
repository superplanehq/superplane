import { act, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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

afterEach(() => {
  vi.useRealTimers();
});

describe("PhaseLogCard title line", () => {
  it("shows the phase name and puts status plus time on the far right", () => {
    render(<PhaseLogCard phase={PHASE} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    expect(within(row).getByRole("button", { name: "Plan" })).toBeInTheDocument();
    expect(within(row).queryByText("Planning")).not.toBeInTheDocument();
    expect(within(row).queryByText("Completed")).not.toBeInTheDocument();
    const statusTime = within(row).getByTestId("split-run-phase-duration-plan");
    expect(statusTime).toHaveTextContent("Passed 01:00");
    expect(statusTime.className).toMatch(/ml-auto/);
    expect(row.firstElementChild?.className).toMatch(/font-mono/);
    expect(row.firstElementChild?.className).toMatch(/text-\[14px\]/);
    expect(row.firstElementChild?.className).not.toMatch(/font-semibold/);
  });

  it("ticks the running clock and rotates the line", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00.000Z"));

    render(<PhaseLogCard phase={{ ...PHASE, status: "running", duration: "4m so far" }} expanded={false} />);

    const badge = screen.getByTestId("split-run-phase-duration-plan");
    expect(badge).toHaveAccessibleName("Running");
    expect(badge).toHaveTextContent("|");
    expect(badge).toHaveTextContent("Running 04:00");

    await act(async () => {
      vi.advanceTimersByTime(250);
    });
    expect(badge).toHaveTextContent("/");

    await act(async () => {
      vi.advanceTimersByTime(750);
    });
    expect(badge).toHaveTextContent("Running 04:01");

    vi.useRealTimers();
  });

  it("puts bold artifacts before the duration on the far right", () => {
    render(
      <PhaseLogCard
        phase={{
          ...PHASE,
          artifacts: [
            {
              id: "art-plan",
              type: "TYPE_MARKDOWN",
              data: { name: "PLAN.md", title: "PLAN.md" },
            },
          ],
        }}
        expanded={false}
      />,
    );

    const row = screen.getByTestId("split-run-phase-plan");
    const name = within(row).getByRole("button", { name: "Plan" });
    const artifact = within(row).getByRole("button", { name: "PLAN.md" });
    const duration = within(row).getByTestId("split-run-phase-duration-plan");

    expect(name.compareDocumentPosition(artifact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.compareDocumentPosition(duration) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(duration.className).toMatch(/ml-auto/);
    expect(artifact.className).toMatch(/font-bold/);
    expect(name.className).toMatch(/flex-1/);
    expect(within(row).getByTestId("split-run-phase-artifacts-plan").className).toMatch(/justify-end/);
  });
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
    expect(statusTime).toHaveTextContent("Passed 01:20");
    expect(statusTime.className).toMatch(/ml-auto/);
    expect(statusTime.className).toMatch(/text-right/);
  });

  it("aligns the node icon with the phase glyph column", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/w-4/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
