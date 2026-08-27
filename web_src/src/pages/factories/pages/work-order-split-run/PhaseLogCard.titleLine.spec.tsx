import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
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
    expect(artifact.className).toMatch(/font-bold/);
    // Artifacts and the duration share one right-aligned cluster. Separate
    // `ml-auto` margins would split the free space and strand the artifacts
    // mid-row.
    const artifacts = within(row).getByTestId("split-run-phase-artifacts-plan");
    expect(artifacts.parentElement).toBe(duration.parentElement);
    expect(duration.parentElement?.className).toMatch(/ml-auto/);
    expect(artifacts.className).not.toMatch(/ml-auto/);
  });

  it("keeps produced artifacts on the title line when the phase is expanded", () => {
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
        expanded
      />,
    );

    const row = screen.getByTestId("split-run-phase-plan");
    const artifact = within(row).getByRole("button", { name: "PLAN.md" });
    const duration = within(row).getByTestId("split-run-phase-duration-plan");
    expect(within(row).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(artifact.compareDocumentPosition(duration) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});

describe("PhaseLogCard phase actions", () => {
  const RUN_HREF = "/org-1/workspaces/RF/apps/app-refund-planner/split-run?run=run-1";
  const EDIT_HREF = "/org-1/workspaces/RF/apps/app-refund-planner?configure=1";

  function renderCard(ui: ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it("stays off collapsed phases", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded={false} runHref={RUN_HREF} editHref={EDIT_HREF} />);

    expect(screen.queryByRole("link", { name: "View Automation Run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit Automation" })).not.toBeInTheDocument();
  });

  it("puts View Automation Run and Edit Automation next to an expanded name", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded runHref={RUN_HREF} editHref={EDIT_HREF} />);

    const view = screen.getByRole("link", { name: "View Automation Run" });
    const edit = screen.getByRole("link", { name: "Edit Automation" });
    expect(view).toHaveAttribute("href", RUN_HREF);
    expect(edit).toHaveAttribute("href", EDIT_HREF);
    expect(view.previousElementSibling).toBe(screen.getByRole("button", { name: "Plan" }));
    expect(edit.previousElementSibling).toBe(view);
  });

  it("opens the run or canvas without collapsing the phase", async () => {
    const onToggle = vi.fn();
    const user = userEvent.setup();
    renderCard(<PhaseLogCard phase={PHASE} expanded runHref={RUN_HREF} editHref={EDIT_HREF} onToggle={onToggle} />);

    await user.click(screen.getByRole("link", { name: "View Automation Run" }));
    await user.click(screen.getByRole("link", { name: "Edit Automation" }));

    expect(onToggle).not.toHaveBeenCalled();
  });

  it("is absent when the log has no run or canvas path", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded />);

    expect(screen.queryByRole("link", { name: "View Automation Run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit Automation" })).not.toBeInTheDocument();
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

  it("aligns the node icon with the phase glyph column", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={NODE_STREAM} />);

    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/w-3/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
