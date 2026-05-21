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

  it("renders the namespace count badge when memoryNamespaceCount > 0", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectMemory={vi.fn()} memoryNamespaceCount={3} />,
    );

    const badge = screen.getByTestId("canvas-view-mode-memory-badge");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveTextContent("3");
  });

  it("hides the badge entirely when memoryNamespaceCount is 0", () => {
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
