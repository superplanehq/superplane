import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { CanvasModeToggle } from "./CanvasModeToggle";

describe("CanvasModeToggle", () => {
  it("exits runs mode when clicking the Canvas tab", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();

    render(<CanvasModeToggle mode="runs" onSelectLive={onSelectLive} onSelectDashboard={vi.fn()} />);

    await user.click(screen.getByRole("tab", { name: "Canvas" }));

    expect(onSelectLive).toHaveBeenCalledTimes(1);
  });

  it("does not drop subsequent Canvas clicks when mode does not update immediately", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();

    render(<CanvasModeToggle mode="runs" onSelectLive={onSelectLive} onSelectDashboard={vi.fn()} />);

    await user.click(screen.getByRole("tab", { name: "Canvas" }));
    await user.click(screen.getByRole("tab", { name: "Canvas" }));

    expect(onSelectLive).toHaveBeenCalledTimes(2);
  });

  it("invokes onSelectMemory when clicking the Memory tab", async () => {
    const user = userEvent.setup();
    const onSelectMemory = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectDashboard={vi.fn()}
        onSelectMemory={onSelectMemory}
      />,
    );

    await user.click(screen.getByRole("tab", { name: "Memory" }));

    expect(onSelectMemory).toHaveBeenCalledTimes(1);
  });

  it("hides the Memory tab when onSelectMemory is not provided", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectDashboard={vi.fn()} />);

    expect(screen.queryByRole("tab", { name: "Memory" })).not.toBeInTheDocument();
  });

  it("shows a draft indicator on the Console tab when the console draft is dirty", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectDashboard={vi.fn()} hasDashboardDraft />,
    );

    expect(screen.getByTestId("canvas-view-mode-dashboard-draft-dot")).toBeInTheDocument();
    expect(screen.queryByTestId("canvas-view-mode-live-draft-dot")).not.toBeInTheDocument();
  });

  it("invokes onSelectFiles when clicking the Files tab", async () => {
    const user = userEvent.setup();
    const onSelectFiles = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectDashboard={vi.fn()}
        onSelectFiles={onSelectFiles}
      />,
    );

    await user.click(screen.getByRole("tab", { name: "Files" }));

    expect(onSelectFiles).toHaveBeenCalledTimes(1);
  });
});
