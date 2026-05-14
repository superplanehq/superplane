import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CanvasModeToggle } from "./CanvasModeToggle";

describe("CanvasModeToggle", () => {
  it("renders Apps | Live | Runs (no Editor tab) when all callbacks are provided", () => {
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLaunchpad={vi.fn()}
        onSelectLive={vi.fn()}
        onSelectRuns={vi.fn()}
      />,
    );

    expect(screen.getByTestId("canvas-view-mode-launchpad")).toBeInTheDocument();
    expect(screen.getByTestId("canvas-view-mode-live")).toBeInTheDocument();
    expect(screen.getByTestId("canvas-view-mode-runs")).toBeInTheDocument();
    expect(screen.queryByTestId("canvas-view-mode-editor")).toBeNull();
  });

  it("hides Apps and Runs when their callbacks are omitted", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} />);

    expect(screen.queryByTestId("canvas-view-mode-launchpad")).toBeNull();
    expect(screen.queryByTestId("canvas-view-mode-runs")).toBeNull();
    expect(screen.getByTestId("canvas-view-mode-live")).toBeInTheDocument();
  });

  it("calls onSelectLaunchpad when Apps is clicked", async () => {
    const user = userEvent.setup();
    const onSelectLaunchpad = vi.fn();
    const onSelectLive = vi.fn();
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLaunchpad={onSelectLaunchpad}
        onSelectLive={onSelectLive}
        onSelectRuns={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("canvas-view-mode-launchpad"));

    expect(onSelectLaunchpad).toHaveBeenCalled();
    expect(onSelectLive).not.toHaveBeenCalled();
  });

  it("calls onSelectLive when Live is clicked from another mode", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();
    render(
      <CanvasModeToggle
        mode="launchpad"
        onSelectLaunchpad={vi.fn()}
        onSelectLive={onSelectLive}
        onSelectRuns={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("canvas-view-mode-live"));

    expect(onSelectLive).toHaveBeenCalled();
  });

  it("calls onSelectRuns when Runs is clicked", async () => {
    const user = userEvent.setup();
    const onSelectRuns = vi.fn();
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLaunchpad={vi.fn()}
        onSelectLive={vi.fn()}
        onSelectRuns={onSelectRuns}
      />,
    );

    await user.click(screen.getByTestId("canvas-view-mode-runs"));

    expect(onSelectRuns).toHaveBeenCalled();
  });

  it("does not re-fire when clicking the already-active tab", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLaunchpad={vi.fn()}
        onSelectLive={onSelectLive}
        onSelectRuns={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("canvas-view-mode-live"));

    expect(onSelectLive).not.toHaveBeenCalled();
  });

  it("renders a runs notification badge when count > 0", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectRuns={vi.fn()} runsNotificationCount={3} />,
    );

    const runs = screen.getByTestId("canvas-view-mode-runs");
    expect(runs.textContent).toContain("Runs");
    expect(runs.textContent).toContain("3");
  });
});
