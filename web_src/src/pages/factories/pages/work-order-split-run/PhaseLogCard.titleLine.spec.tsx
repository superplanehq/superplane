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
    expect(within(row).queryByText("Passed 01:00")).not.toBeInTheDocument();
    expect(within(row).getByTestId("split-run-phase-duration-plan")).toHaveTextContent("01:00");
    expect(row.firstElementChild?.className).toMatch(/rounded-md/);
    expect(row.firstElementChild?.className).toMatch(/\bbg-muted\b/);
    expect(row.firstElementChild?.className).toMatch(/hover:bg-/);
    expect(row.firstElementChild?.className).not.toMatch(/\bborder\b/);
    expect(within(row).getByRole("button", { name: "Plan" }).querySelector(".lucide-chevron-right")).toBeNull();
  });

  it("uses one mono face and size on the name and artifact", () => {
    render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded={false}
        onStop={vi.fn()}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const name = within(header).getByRole("button", { name: "Plan" });
    const artifact = within(header).getByRole("button", { name: "PLAN.md" });

    for (const el of [name, artifact]) {
      expect(el.className).toMatch(/font-mono/);
      expect(el.className).toMatch(/text-\[14px\]/);
    }
  });

  it("ticks the running clock on the automation and the node", async () => {
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

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("04:00");
    const badge = screen.getByTestId("split-run-stream-duration-planner-agent");
    expect(badge).toHaveAccessibleName("Running");
    expect(badge).toHaveTextContent("✓");
    expect(badge).toHaveTextContent("04:00");
    expect(badge).not.toHaveTextContent("Running");
    expect(badge.querySelector(".lucide-loader-circle")).toBeNull();
    expect(badge.className).not.toMatch(/bg-/);

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(badge).toHaveTextContent("04:01");
    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("04:01");

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
    expect(name.className).not.toMatch(/flex-1/);
    const artifacts = within(row).getByTestId("split-run-phase-artifacts-plan");
    const duration = within(row).getByTestId("split-run-phase-duration-plan");
    expect(artifacts.className).toMatch(/justify-end/);
    expect(artifacts.parentElement).toBe(duration.parentElement);
    expect(artifacts.parentElement?.className).toMatch(/ml-auto/);
    expect(duration).toHaveTextContent("01:00");

    rerender(<PhaseLogCard phase={phase} expanded stream={[]} />);
    expect(within(row).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-automation-header-plan")).getByRole("button", { name: "PLAN.md" }),
    ).toBeInTheDocument();
  });

  it("puts artifacts immediately before the duration on the right", () => {
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
    const artifacts = within(header).getByTestId("split-run-phase-artifacts-plan");
    const duration = within(header).getByTestId("split-run-phase-duration-plan");
    expect(artifacts.compareDocumentPosition(duration) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifacts.parentElement).toBe(duration.parentElement);
    expect(artifacts.nextElementSibling).toBe(duration);
  });

  it("offers Stop on a running automation and Rerun on a failed one", () => {
    const onStop = vi.fn();
    const onRerun = vi.fn();
    const { rerender } = render(
      <PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );

    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Stop" }).className).toMatch(/size-6/);
    expect(screen.getByRole("button", { name: "Stop" }).className).toMatch(/hover:text-destructive/);
    expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun" })).not.toBeInTheDocument();

    rerender(
      <PhaseLogCard phase={{ ...PHASE, status: "failed" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );
    const rerun = screen.getByRole("button", { name: "Rerun" });
    expect(rerun).toBeInTheDocument();
    expect(rerun.className).toMatch(/size-6/);
    expect(screen.queryByText("Rerun")).not.toBeInTheDocument();
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

  it("toggles from the gray header except nested controls", async () => {
    const onToggle = vi.fn();
    const onStop = vi.fn();
    const user = userEvent.setup();
    const { rerender } = render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded={false}
        onToggle={onToggle}
        onStop={onStop}
      />,
    );

    const card = screen.getByTestId("split-run-phase-plan").firstElementChild;
    expect(card?.className).toMatch(/cursor-pointer/);
    expect(screen.getByTestId("split-run-phase-expand-plan")).toBeInTheDocument();

    await user.click(screen.getByTestId("split-run-phase-duration-plan"));
    expect(onToggle).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "Stop" }));
    expect(onToggle).toHaveBeenCalledTimes(1);
    expect(onStop).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole("button", { name: "PLAN.md" }));
    expect(onToggle).toHaveBeenCalledTimes(1);
    await user.keyboard("{Escape}");

    rerender(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        stream={[line({ id: "planner-agent", componentName: "Agent - Plan for GH Issue" })]}
        onToggle={onToggle}
        onStop={onStop}
      />,
    );
    expect(screen.getByTestId("split-run-phase-expand-plan")).toBeInTheDocument();
    expect(screen.getByTestId("split-run-phase-plan").firstElementChild?.className).not.toMatch(/cursor-pointer/);
    expect(screen.getByTestId("split-run-automation-header-plan").className).toMatch(/cursor-pointer/);

    await user.click(screen.getByTestId("split-run-phase-duration-plan"));
    expect(onToggle).toHaveBeenCalledTimes(2);

    await user.click(screen.getByText("Agent - Plan for GH Issue"));
    expect(onToggle).toHaveBeenCalledTimes(2);
  });

  it("sits on a rounded block that tints on hover", () => {
    const { rerender } = render(<PhaseLogCard phase={PHASE} expanded={false} />);

    const card = () => screen.getByTestId("split-run-phase-plan").firstElementChild;
    expect(card()?.className).toMatch(/rounded-md/);
    expect(card()?.className).toMatch(/\bbg-muted\b/);
    expect(card()?.className).toMatch(/hover:bg-/);
    expect(card()?.className).not.toMatch(/\bbg-card\b/);
    expect(card()?.className).not.toMatch(/\bborder\b/);
    expect(card()?.className).not.toMatch(/border-l/);

    expect(card()?.className).toMatch(/\bpy-2\b/);

    rerender(<PhaseLogCard phase={PHASE} expanded stream={[]} />);
    expect(card()?.className).toMatch(/\bbg-muted\b/);
    expect(card()?.className).not.toMatch(/hover:bg-/);
    expect(card()?.className).toMatch(/\bpb-2\b/);
    expect(card()?.className).not.toMatch(/\bborder\b/);

    const header = screen.getByTestId("split-run-automation-header-plan");
    expect(header.className).toMatch(/\bh-8\b/);
    expect(header.className).toMatch(/\bbg-muted\b/);
    expect(header.className).not.toMatch(/\bbg-background\b/);
    expect(card()?.className).not.toMatch(/\bpx-2\b/);
    expect(header.className).toMatch(/\bpx-2\b/);
    expect(header.className).not.toMatch(/-mx-2/);
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
    expect(within(row).getByTestId("split-run-phase-duration-plan")).toHaveTextContent("01:00");
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

    expect(screen.queryByRole("link", { name: "View automation run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("puts icon actions next to the expanded name without pill chrome", async () => {
    const user = userEvent.setup();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        runHref={RUN_HREF}
        editHref={EDIT_HREF}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const name = within(header).getByRole("button", { name: "Plan" });
    const view = screen.getByRole("link", { name: "View automation run" });
    const edit = screen.getByRole("link", { name: "Edit automation" });
    const artifact = within(header).getByRole("button", { name: "PLAN.md" });
    const duration = within(header).getByTestId("split-run-phase-duration-plan");
    expect(view).toHaveAttribute("href", RUN_HREF);
    expect(edit).toHaveAttribute("href", EDIT_HREF);
    expect(screen.queryByText("View automation run")).not.toBeInTheDocument();
    expect(screen.queryByText("Edit automation")).not.toBeInTheDocument();
    expect(view.className).toMatch(/font-mono/);
    expect(view.className).not.toMatch(/rounded-full/);
    expect(view.className).not.toMatch(/border-border/);
    expect(name.compareDocumentPosition(view) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(view.compareDocumentPosition(edit) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(edit.compareDocumentPosition(artifact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.compareDocumentPosition(duration) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.closest("[data-testid='split-run-phase-artifacts-plan']")?.parentElement).toBe(
      duration.parentElement,
    );
    expect(duration.parentElement?.className).toMatch(/ml-auto/);
    expect(duration).toHaveTextContent("01:00");
    expect(duration.className).toMatch(/font-mono/);
    expect(duration.className).toMatch(/tabular-nums/);

    await user.hover(view);
    expect(await screen.findByRole("tooltip", { name: "View automation run" })).toBeInTheDocument();
    await user.unhover(view);
    await user.hover(edit);
    expect(await screen.findByRole("tooltip", { name: "Edit automation" })).toBeInTheDocument();
  });

  it("puts Stop after Edit on the left of a running automation", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        runHref={RUN_HREF}
        editHref={EDIT_HREF}
        onStop={onStop}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const edit = screen.getByRole("link", { name: "Edit automation" });
    const stop = screen.getByRole("button", { name: "Stop" });
    const artifact = within(header).getByRole("button", { name: "PLAN.md" });
    expect(edit.compareDocumentPosition(stop) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(stop.compareDocumentPosition(artifact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.closest("[data-testid='split-run-phase-artifacts-plan']")?.parentElement).not.toBe(
      stop.parentElement,
    );
    expect(stop.querySelector(".lucide-circle-x")).toBeTruthy();

    await user.hover(stop);
    expect(await screen.findByRole("tooltip", { name: "Stop" })).toBeInTheDocument();
    await user.click(stop);
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("puts Rerun after Edit on the left of a failed automation", async () => {
    const user = userEvent.setup();
    const onRerun = vi.fn();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "failed", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        runHref={RUN_HREF}
        editHref={EDIT_HREF}
        onRerun={onRerun}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const name = within(header).getByRole("button", { name: "Plan" });
    const edit = screen.getByRole("link", { name: "Edit automation" });
    const rerun = screen.getByRole("button", { name: "Rerun" });
    const artifact = within(header).getByRole("button", { name: "PLAN.md" });
    expect(name.compareDocumentPosition(edit) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(edit.compareDocumentPosition(rerun) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(rerun.compareDocumentPosition(artifact) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(artifact.closest("[data-testid='split-run-phase-artifacts-plan']")?.parentElement).not.toBe(
      rerun.parentElement,
    );
    expect(screen.queryByText("Rerun")).not.toBeInTheDocument();

    await user.hover(rerun);
    expect(await screen.findByRole("tooltip", { name: "Rerun" })).toBeInTheDocument();
    await user.click(rerun);
    expect(onRerun).toHaveBeenCalledTimes(1);
  });

  it("opens the run or canvas without collapsing the phase", async () => {
    const onToggle = vi.fn();
    const user = userEvent.setup();
    renderCard(<PhaseLogCard phase={PHASE} expanded runHref={RUN_HREF} editHref={EDIT_HREF} onToggle={onToggle} />);

    await user.click(screen.getByRole("link", { name: "View automation run" }));
    await user.click(screen.getByRole("link", { name: "Edit automation" }));

    expect(onToggle).not.toHaveBeenCalled();
  });

  it("is absent when the log has no run or canvas path", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded />);

    expect(screen.queryByRole("link", { name: "View automation run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
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
    expect(row.className).not.toMatch(/\bbg-muted\b/);
    expect(row.className).not.toMatch(/hover:bg-/);
    expect(row.className).not.toMatch(/border-b/);
    expect(row.className).not.toMatch(/-mx-2/);
    const toggle = screen.getByTestId("split-run-node-toggle-planner-agent");
    expect(toggle.className).toMatch(/gap-1\.5/);
    expect(toggle.firstElementChild?.className).toMatch(/size-3/);
    expect(screen.queryByTestId("split-run-node-indent")).not.toBeInTheDocument();
  });
});
