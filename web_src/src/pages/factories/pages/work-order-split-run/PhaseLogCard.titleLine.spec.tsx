import { act, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { PhaseLogCard } from "./PhaseLogCard";
import { idleLiveLogStream, line, PHASE } from "./PhaseLogCard.testHelpers";

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
    expect(within(row).getByTestId("split-run-phase-duration-plan")).toHaveTextContent("1m");
  });

  it("shows cost and tokens next to duration when the phase has spend", () => {
    render(<PhaseLogCard phase={{ ...PHASE, costCents: "45", totalTokens: "1200" }} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    expect(within(row).getByTestId("split-run-phase-duration-plan")).toHaveTextContent("$0.45 · 1.2k · 1m");
  });

  it("shows tokens alone when the phase has no dollar spend", () => {
    render(<PhaseLogCard phase={{ ...PHASE, totalTokens: "1200" }} expanded={false} />);

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("1.2k · 1m");
  });

  it("shows the used model next to spend", () => {
    render(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "1200", model: "anthropic/claude-opus-4-6" }}
        expanded={false}
      />,
    );

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("$0.45 · 1.2k · opus 4-6 · 1m");
  });

  it("shows the thinking level in the model segment", () => {
    render(
      <PhaseLogCard
        phase={{
          ...PHASE,
          costCents: "45",
          totalTokens: "1200",
          model: "anthropic/claude-opus-4-6",
          thinkingLevel: "medium",
        }}
        expanded={false}
      />,
    );

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent(
      "$0.45 · 1.2k · opus 4-6 Medium · 1m",
    );
  });

  it("does not add a thinking word for Auto or an empty value", () => {
    const { rerender } = render(
      <PhaseLogCard
        phase={{
          ...PHASE,
          costCents: "45",
          totalTokens: "1200",
          model: "anthropic/claude-opus-4-6",
          thinkingLevel: "auto",
        }}
        expanded={false}
      />,
    );

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("$0.45 · 1.2k · opus 4-6 · 1m");
    expect(screen.getByTestId("split-run-phase-duration-plan")).not.toHaveTextContent("Medium");

    rerender(
      <PhaseLogCard
        phase={{
          ...PHASE,
          costCents: "45",
          totalTokens: "1200",
          model: "anthropic/claude-opus-4-6",
          thinkingLevel: "",
        }}
        expanded={false}
      />,
    );

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("$0.45 · 1.2k · opus 4-6 · 1m");
    expect(screen.getByTestId("split-run-phase-duration-plan")).not.toHaveTextContent("Default");
  });

  it("does not add a thinking word when the phase has no model", () => {
    render(
      <PhaseLogCard
        phase={{ ...PHASE, costCents: "45", totalTokens: "1200", thinkingLevel: "medium" }}
        expanded={false}
      />,
    );

    const metrics = screen.getByTestId("split-run-phase-duration-plan");
    expect(metrics).toHaveTextContent("$0.45 · 1.2k · 1m");
    expect(metrics).not.toHaveTextContent("Medium");
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

    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("4m");
    const badge = screen.getByTestId("split-run-stream-duration-planner-agent");
    expect(badge).toHaveAccessibleName("Running");
    expect(badge).not.toHaveTextContent("✓");
    expect(badge).toHaveTextContent("04:00");
    expect(badge).not.toHaveTextContent("Running");

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(badge).toHaveTextContent("04:01");
    expect(screen.getByTestId("split-run-phase-duration-plan")).toHaveTextContent("4m1s");

    vi.useRealTimers();
  });

  it("keeps a waiting timeout duration fixed", async () => {
    vi.useFakeTimers();

    render(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", duration: "2m", durationRunning: false }}
        expanded
        stream={[
          line({
            id: "timed-out-analysis",
            componentName: "Analysis",
            status: "running",
            duration: "2m",
            durationRunning: false,
          }),
        ]}
      />,
    );

    const phaseDuration = screen.getByTestId("split-run-phase-duration-plan");
    const streamDuration = screen.getByTestId("split-run-stream-duration-timed-out-analysis");
    expect(phaseDuration).toHaveTextContent("2m");
    expect(streamDuration).toHaveTextContent("02:00");

    await act(async () => {
      vi.advanceTimersByTime(1000);
    });
    expect(phaseDuration).toHaveTextContent("2m");
    expect(streamDuration).toHaveTextContent("02:00");
  });

  it("keeps artifacts on the automation header when the card is expanded", () => {
    const phase = {
      ...PHASE,
      artifacts: [PLAN_MD_ARTIFACT],
    };
    const { rerender } = render(<PhaseLogCard phase={phase} expanded={false} />);

    const row = screen.getByTestId("split-run-phase-plan");
    expect(within(row).getByRole("button", { name: "PLAN.md" })).toBeInTheDocument();

    const duration = within(row).getByTestId("split-run-phase-duration-plan");
    expect(duration).toHaveTextContent("1m");

    rerender(<PhaseLogCard phase={phase} expanded stream={[]} />);
    expect(within(row).getByTestId("split-run-phase-artifacts-plan")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("split-run-automation-header-plan")).getByRole("button", { name: "PLAN.md" }),
    ).toBeInTheDocument();
  });

  it("offers Stop on a running automation and Rerun on a failed one", () => {
    const onStop = vi.fn();
    const onRerun = vi.fn();
    const { rerender } = render(
      <PhaseLogCard phase={{ ...PHASE, status: "running" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );

    expect(screen.getByRole("button", { name: "Stop" })).toBeInTheDocument();
    expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Rerun" })).not.toBeInTheDocument();

    rerender(
      <PhaseLogCard phase={{ ...PHASE, status: "failed" }} expanded={false} onStop={onStop} onRerun={onRerun} />,
    );
    const rerun = screen.getByRole("button", { name: "Rerun" });
    expect(rerun).toBeInTheDocument();
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

  it("toggles from the header except nested controls", async () => {
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

    await user.click(screen.getByTestId("split-run-phase-duration-plan"));
    expect(onToggle).toHaveBeenCalledTimes(2);

    await user.click(screen.getByText("Agent - Plan for GH Issue"));
    expect(onToggle).toHaveBeenCalledTimes(2);
  });
});

describe("PhaseLogCard phase actions", () => {
  const RUN_HREF = "/org-1/workspaces/RF/apps/app-refund-planner/split-run?run=run-1";

  function renderCard(ui: ReactElement) {
    return render(<MemoryRouter>{ui}</MemoryRouter>);
  }

  it("stays off collapsed phases", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded={false} runHref={RUN_HREF} />);

    expect(screen.queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });

  it("offers a run link with a tooltip on expanded phases", async () => {
    const user = userEvent.setup();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, artifacts: [PLAN_MD_ARTIFACT], costCents: "45", totalTokens: "210" }}
        expanded
        runHref={RUN_HREF}
      />,
    );

    const header = screen.getByTestId("split-run-automation-header-plan");
    const view = screen.getByRole("link", { name: "View run" });
    const duration = within(header).getByTestId("split-run-phase-duration-plan");
    expect(view).toHaveAttribute("href", RUN_HREF);
    expect(screen.queryByText("View run")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
    expect(duration).toHaveTextContent("$0.45");
    expect(duration).toHaveTextContent("1m");

    await user.hover(view);
    expect(await screen.findByRole("tooltip", { name: "View run" })).toBeInTheDocument();
  });

  it("stops a running automation from the expanded header", async () => {
    const user = userEvent.setup();
    const onStop = vi.fn();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "running", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        runHref={RUN_HREF}
        onStop={onStop}
      />,
    );

    const stop = screen.getByRole("button", { name: "Stop" });
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();

    await user.hover(stop);
    expect(await screen.findByRole("tooltip", { name: "Stop" })).toBeInTheDocument();
    await user.click(stop);
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("reruns a failed automation from the expanded header", async () => {
    const user = userEvent.setup();
    const onRerun = vi.fn();
    renderCard(
      <PhaseLogCard
        phase={{ ...PHASE, status: "failed", artifacts: [PLAN_MD_ARTIFACT] }}
        expanded
        runHref={RUN_HREF}
        onRerun={onRerun}
      />,
    );

    const rerun = screen.getByRole("button", { name: "Rerun" });
    expect(screen.queryByText("Rerun")).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();

    await user.hover(rerun);
    expect(await screen.findByRole("tooltip", { name: "Rerun" })).toBeInTheDocument();
    await user.click(rerun);
    expect(onRerun).toHaveBeenCalledTimes(1);
  });

  it("opens the run without collapsing the phase", async () => {
    const onToggle = vi.fn();
    const user = userEvent.setup();
    renderCard(<PhaseLogCard phase={PHASE} expanded runHref={RUN_HREF} onToggle={onToggle} />);

    await user.click(screen.getByRole("link", { name: "View run" }));

    expect(onToggle).not.toHaveBeenCalled();
  });

  it("is absent when the log has no run path", () => {
    renderCard(<PhaseLogCard phase={PHASE} expanded />);

    expect(screen.queryByRole("link", { name: "View run" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: "Edit automation" })).not.toBeInTheDocument();
  });
});
