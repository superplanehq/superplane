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

  it("does not render the namespace count badge while MEMORY_TAB_NAMESPACE_BADGE_ENABLED is false", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectMemory={vi.fn()} memoryNamespaceCount={3} />,
    );

    expect(screen.queryByTestId("canvas-view-mode-memory-badge")).not.toBeInTheDocument();
  });

  it("hides the badge when memoryNamespaceCount is 0", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectMemory={vi.fn()} memoryNamespaceCount={0} />,
    );

    expect(screen.queryByTestId("canvas-view-mode-memory-badge")).not.toBeInTheDocument();
  });

  it("hides the Memory tab when onSelectMemory is not provided", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectDashboard={vi.fn()} />);

    expect(screen.queryByRole("tab", { name: "Memory" })).not.toBeInTheDocument();
  });
});
