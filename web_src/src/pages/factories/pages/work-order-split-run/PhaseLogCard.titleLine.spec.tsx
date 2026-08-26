import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

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

describe("PhaseLogCard title line", () => {
  it("shows the phase name without the component or status word", () => {
    render(<PhaseLogCard phase={PHASE} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    expect(within(row).getByRole("button", { name: "Plan" })).toBeInTheDocument();
    expect(within(row).queryByText("Planning")).not.toBeInTheDocument();
    expect(within(row).queryByText("Completed")).not.toBeInTheDocument();
    expect(within(row).getByTestId("split-run-phase-duration-plan")).toHaveTextContent("01:00");
    expect(row.firstElementChild?.className).toMatch(/font-mono/);
    expect(row.firstElementChild?.className).toMatch(/text-\[14px\]/);
    expect(row.firstElementChild?.className).not.toMatch(/font-semibold/);
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
    expect(within(row).getByTestId("split-run-phase-artifacts-plan").className).toMatch(/ml-auto/);
  });
});

describe("PhaseLogCard edit control", () => {
  it("stays off collapsed phases", () => {
    render(<PhaseLogCard phase={PHASE} expanded={false} onEdit={vi.fn()} />);

    expect(screen.queryByTestId("split-run-phase-edit-plan")).not.toBeInTheDocument();
  });

  it("sits next to the name of an expanded phase", () => {
    render(<PhaseLogCard phase={PHASE} expanded onEdit={vi.fn()} />);

    const edit = screen.getByTestId("split-run-phase-edit-plan");
    expect(edit).toHaveAccessibleName("Edit Plan automation");
    expect(edit.previousElementSibling).toBe(screen.getByRole("button", { name: "Plan" }));
  });

  it("opens the automation editor without collapsing the phase", async () => {
    const onEdit = vi.fn();
    const onToggle = vi.fn();
    const user = userEvent.setup();
    render(<PhaseLogCard phase={PHASE} expanded onEdit={onEdit} onToggle={onToggle} />);

    await user.click(screen.getByTestId("split-run-phase-edit-plan"));

    expect(onEdit).toHaveBeenCalledTimes(1);
    expect(onToggle).not.toHaveBeenCalled();
  });

  it("is absent when the log cannot edit automations", () => {
    render(<PhaseLogCard phase={PHASE} expanded />);

    expect(screen.queryByTestId("split-run-phase-edit-plan")).not.toBeInTheDocument();
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

  it("shows the node name and the clock duration on the far right", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const row = screen.getByTestId("split-run-stream-line-planner-agent");
    expect(within(row).getByText("Agent - Plan for GH Issue")).toBeInTheDocument();
    expect(within(row).getByText("passed")).toBeInTheDocument();
    expect(within(row).queryByText("12:24:02")).not.toBeInTheDocument();
    expect(within(row).queryByText("Run Claude Code")).not.toBeInTheDocument();

    const duration = within(row).getByTestId("split-run-stream-duration-planner-agent");
    expect(duration).toHaveTextContent("01:20");
    expect(duration.className).toMatch(/text-right/);
  });

  it("aligns the node icon with the phase glyph column", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/w-3/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
