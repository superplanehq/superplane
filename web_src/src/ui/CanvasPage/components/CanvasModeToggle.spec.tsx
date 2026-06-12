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

  it("invokes onSelectRuns when clicking the Runs tab", async () => {
    const user = userEvent.setup();
    const onSelectRuns = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        onSelectRuns={onSelectRuns}
      />,
      { wrapper: routerWrapper },
    );

    await user.click(screen.getByRole("link", { name: "Runs" }));

    expect(onSelectRuns).toHaveBeenCalledTimes(1);
  });

  it("hides the Runs tab when onSelectRuns is not provided", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectConsole={vi.fn()} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByRole("link", { name: "Runs" })).not.toBeInTheDocument();
  });

  it("invokes onSelectVersions when clicking the Versions tab", async () => {
    const user = userEvent.setup();
    const onSelectVersions = vi.fn();

    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        onSelectVersions={onSelectVersions}
      />,
      { wrapper: routerWrapper },
    );

    await user.click(screen.getByRole("link", { name: "Versions" }));

    expect(onSelectVersions).toHaveBeenCalledTimes(1);
  });

  it("hides the Versions tab when onSelectVersions is not provided", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectConsole={vi.fn()} />, {
      wrapper: routerWrapper,
    });

    expect(screen.queryByRole("link", { name: "Versions" })).not.toBeInTheDocument();
  });

  it("orders tabs as Canvas, Runs, Versions, then Console and shows a divider before the secondary group", () => {
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        onSelectRuns={vi.fn()}
        onSelectVersions={vi.fn()}
        onSelectMemory={vi.fn()}
        onSelectFiles={vi.fn()}
      />,
      { wrapper: routerWrapper },
    );

    const tabs = screen.getAllByRole("link");
    expect(tabs.map((tab) => tab.textContent?.replace(/\s+/g, " ").trim())).toEqual([
      "Canvas",
      "Runs",
      "Versions",
      "Console",
      "Memory",
      "Files",
    ]);
    expect(screen.getByTestId("canvas-view-mode-group-divider")).toBeInTheDocument();
  });

  it("hides the group divider when only the canvas workflow tabs are shown", () => {
    render(
      <CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectRuns={vi.fn()} onSelectVersions={vi.fn()} />,
      { wrapper: routerWrapper },
    );

    expect(screen.queryByTestId("canvas-view-mode-group-divider")).not.toBeInTheDocument();
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

    expect(screen.queryByRole("tab", { name: "Memory" })).not.toBeInTheDocument();
  });

  it("shows a draft indicator on the Console tab when the console draft is dirty", () => {
    render(<CanvasModeToggle mode="version-live" onSelectLive={vi.fn()} onSelectConsole={vi.fn()} hasConsoleDraft />, {
      wrapper: routerWrapper,
    });

    expect(screen.getByTestId("canvas-view-mode-console-draft-dot")).toBeInTheDocument();
    expect(screen.queryByTestId("canvas-view-mode-live-draft-dot")).not.toBeInTheDocument();
  });

  it("uses blue tab styling in edit mode and shows blue draft dots", () => {
    render(
      <CanvasModeToggle
        mode="version-live"
        onSelectLive={vi.fn()}
        onSelectConsole={vi.fn()}
        editing
        hasDraft
        hasConsoleDraft
      />,
      { wrapper: routerWrapper },
    );

    const tabList = screen.getByRole("navigation", { name: "Canvas view" });
    expect(tabList.className).toContain("bg-blue-50");
    expect(tabList.className).not.toContain("bg-slate-100");
    expect(tabList.className).not.toContain("purple");

    // Inactive tabs get editing-specific styling
    const consoleTab = screen.getByRole("link", { name: "Console" });
    expect(consoleTab.className).toContain("text-blue-800/80");

    expect(screen.getByTestId("canvas-view-mode-live-draft-dot")).toHaveClass("bg-blue-500");
    expect(screen.getByTestId("canvas-view-mode-console-draft-dot")).toHaveClass("bg-blue-500");
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
