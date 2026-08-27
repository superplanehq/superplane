import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";
import type { SplitRunStreamLine } from "./splitRunMocks";

const PLAN_MD_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-plan",
  type: "TYPE_MARKDOWN",
  data: { name: "PLAN.md", title: "PLAN.md" },
};

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
  it("shows the phase name without a status time chip", () => {
    render(<PhaseLogCard phase={PHASE} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    expect(within(row).getByRole("button", { name: "Plan" })).toBeInTheDocument();
    expect(within(row).queryByText("Planning")).not.toBeInTheDocument();
    expect(within(row).queryByText("Completed")).not.toBeInTheDocument();
    expect(within(row).queryByTestId("split-run-phase-duration-plan")).not.toBeInTheDocument();
    expect(within(row).queryByText("Passed 01:00")).not.toBeInTheDocument();
    expect(row.firstElementChild?.className).toMatch(/rounded-md/);
    expect(row.firstElementChild?.className).toMatch(/border/);
    expect(within(row).getByRole("button", { name: "Plan" }).querySelector(".lucide-chevron-right")).toBeNull();
    expect(within(row).getByRole("button", { name: "Plan" }).className).not.toMatch(/font-mono/);
  });

  it("ticks the running clock on a node, not the automation", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00.000Z"));

    render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", duration: "4m so far" }}
        expanded
        stream={[
          line({
            id: "planner-agent",
            componentName: "Agent - Plan for GH Issue",
            status: "running",
            duration: "4m so far",
          }),
        ]}
      />,
    );

    expect(screen.queryByTestId("split-run-phase-duration-plan")).not.toBeInTheDocument();
    const badge = screen.getByTestId("split-run-stream-duration-planner-agent");
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

  it("keeps bold artifacts on the automation header when the card is expanded", () => {
    const phase = {
      ...PHASE,
      artifacts: [PLAN_MD_ARTIFACT],
    };
    const { rerender } = render(<PhaseLogCard phase={phase} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    const name = within(row).getByRole("button", { name: "Plan" });
    const artifact = within(row).getByRole("button", { name: "PLAN.md" });

    expect(name.compareDocumentPosition(artifact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.className).toMatch(/font-bold/);
    expect(name.className).toMatch(/flex-1/);
    expect(within(row).getByTestId("split-run-phase-artifacts-plan").className).toMatch(/justify-end/);
    expect(within(row).queryByTestId("split-run-phase-duration-plan")).not.toBeInTheDocument();

    rerender(<PhaseLogCard phase={phase} expanded stream={[]} />);
    expect(within(row).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-automation-header-plan")).getByRole("button", { name: "PLAN.md" }),
    ).toBeInTheDocument();
  });

  it("puts Stop to the right of artifacts on a running automation", () => {
    render(
      <PhaseLogCard
        phase={{
          ...PHASE,
          status: "running",
          artifacts: [PLAN_MD_ARTIFACT],
        }}
        expanded={false}
        onStop={vi.fn()}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const artifact = within(header).getByRole("button", { name: "PLAN.md" });
    const stop = within(header).getByRole("button", { name: "Stop" });
    expect(artifact.compareDocumentPosition(stop) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("offers Stop on a running automation and Rerun on a failed one", () => {
    const onStop = vi.fn();
    const onRerun = vi.fn();
    const { rerender } = render(
      <PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );

    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop" }).className).toMatch(/text-destructive/);
    expect(screen.getByRole("button", { name: "Stop" }).className).not.toMatch(/bg-destructive/);
    expect(screen.getByRole("button", { name: "Stop" }).className).not.toMatch(/text-muted-foreground/);
    expect(screen.queryByRole("button", { name: "Rerun" })).not.toBeInTheDocument();

    rerender(
      <PhaseLogCard phase={{ ...PHASE, status: "failed" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );
    expect(screen.getByRole("button", { name: "Rerun" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();

    rerender(<PhaseLogCard phase={PHASE} expanded={false} onStop={onStop} onRerun={onRerun} />);
    expect(screen.queryByRole("button", { name: "Stop" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun" })).not.toBeInTheDocument();
  });

  it("disables Stop and Rerun while an action is busy", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    const onRerun = vi.fn();
    const { rerender } = render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running" }}
        expanded={false}
        onStop={onStop}
        onRerun={onRerun}
        actionBusy
      />,
    );

    const stop = screen.getByRole("button", { name: "Stop" });
    expect(stop).toBeDisabled();
    await user.click(stop);
    expect(onStop).not.toHaveBeenCalled();

    rerender(
      <PhaseLogCard
        phase={{ ...PHASE, status: "failed" }}
        expanded={false}
        onStop={onStop}
        onRerun={onRerun}
        actionBusy
      />,
    );
    const rerun = screen.getByRole("button", { name: "Rerun" });
    expect(rerun).toBeDisabled();
    await user.click(rerun);
    expect(onRerun).not.toHaveBeenCalled();
  });

  it("puts a muted fill on the expanded header, not the nested log", () => {
    render(<PhaseLogCard phase={PHASE} expanded stream={[]} />);

    const card = screen.getByTestId("split-run-phase-plan").firstElementChild;
    expect(card?.className).toMatch(/\bbg-card\b/);
    expect(card?.className).not.toMatch(/\bbg-muted\b/);

    const header = screen.getByTestId("split-run-automation-header-plan");
    expect(header.className).toMatch(/\bh-8\b/);
    expect(header.className).toMatch(/\bbg-muted\b/);
    expect(card?.className).not.toMatch(/\bpx-2\b/);
    expect(header.className).toMatch(/\bpx-2\b/);
    expect(header.className).not.toMatch(/-mx-2/);
  });

  it("keeps a collapsed automation on the card background", () => {
    render(<PhaseLogCard phase={PHASE} expanded={false} />);

    const card = screen.getByTestId("split-run-phase-plan").firstElementChild;
    expect(card?.className).toMatch(/\bbg-card\b/);
    expect(card?.className).not.toMatch(/\bbg-muted\b/);
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
    expect(within(row).getByRole("button", { name: "PLAN.md" })).toBeInTheDocument();
    expect(within(row).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(within(row).queryByTestId("split-run-phase-duration-plan")).not.toBeInTheDocument();
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
    expect(statusTime.parentElement?.className).toMatch(/ml-auto/);
    expect(statusTime.className).toMatch(/text-right/);
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
    expect(row.className).toMatch(/border-b/);
    expect(row.className).not.toMatch(/-mx-2/);
    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/size-3/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
