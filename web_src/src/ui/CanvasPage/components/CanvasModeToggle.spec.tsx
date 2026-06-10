import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { CanvasModeToggle } from "./CanvasModeToggle";

const routerWrapper = ({ children }: { children: React.ReactNode }) => <MemoryRouter>{children}</MemoryRouter>;

describe("CanvasModeToggle", () => {
  it("exits runs mode when clicking the Canvas tab", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();

    render(<CanvasModeToggle mode="runs" onSelectLive={onSelectLive} onSelectConsole={vi.fn()} />, {
      wrapper: routerWrapper,
    });

    await user.click(screen.getByRole("link", { name: "Canvas" }));

    expect(onSelectLive).toHaveBeenCalledTimes(1);
  });

  it("does not drop subsequent Canvas clicks when mode does not update immediately", async () => {
    const user = userEvent.setup();
    const onSelectLive = vi.fn();

    render(<CanvasModeToggle mode="runs" onSelectLive={onSelectLive} onSelectConsole={vi.fn()} />, {
      wrapper: routerWrapper,
    });

    await user.click(screen.getByRole("link", { name: "Canvas" }));
    await user.click(screen.getByRole("link", { name: "Canvas" }));

    expect(onSelectLive).toHaveBeenCalledTimes(2);
  });

  it("invokes onSelectMemory when clicking the Memory tab", async () => {
    const user = userEvent.setup();
    const onSelectMemory = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        onSelectMemory={onSelectMemory}
      />,
      { wrapper: routerWrapper },
    );

    await user.click(screen.getByRole("link", { name: "Memory" }));

    expect(onSelectMemory).toHaveBeenCalledTimes(1);
  });

  it("hides the Memory tab when onSelectMemory is not provided", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectConsole={vi.fn()} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByRole("link", { name: "Memory" })).not.toBeInTheDocument();
  });

  it("shows orange uncommitted dots and orange tab styling when edits are uncommitted", () => {
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        editing
        hasCanvasUncommitted
        hasConsoleUncommitted
        editTabTone="uncommitted"
      />,
      { wrapper: routerWrapper },
    );

    const tabList = screen.getByRole("navigation", { name: "Canvas view" });
    expect(tabList.className).toContain("bg-orange-50");
    expect(screen.getByTestId("canvas-view-mode-live-uncommitted-dot")).toHaveClass("bg-orange-500");
    expect(screen.getByTestId("canvas-view-mode-console-uncommitted-dot")).toHaveClass("bg-orange-500");
  });

  it("shows blue committed dots and blue tab styling when ready to publish", () => {
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        editing
        hasCanvasCommitted
        editTabTone="ready"
      />,
      { wrapper: routerWrapper },
    );

    const tabList = screen.getByRole("navigation", { name: "Canvas view" });
    expect(tabList.className).toContain("bg-blue-50");
    expect(screen.getByTestId("canvas-view-mode-live-committed-dot")).toHaveClass("bg-blue-500");
  });

  it("invokes onSelectFiles when clicking the Files tab", async () => {
    const user = userEvent.setup();
    const onSelectFiles = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        onSelectFiles={onSelectFiles}
      />,
      { wrapper: routerWrapper },
    );

    await user.click(screen.getByRole("link", { name: "Files" }));

    expect(onSelectFiles).toHaveBeenCalledTimes(1);
  });
});
