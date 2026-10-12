import { act, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { BoardColorThemeProvider } from "./BoardColorThemeProvider";
import { useBoardColorTheme } from "./boardColorThemeContext";
import { persistBoardColorTheme, readBoardColorTheme } from "./boardColorThemes";
import { LineBoardViewMenu } from "../pages/LineBoardViewMenu";
import { WorkOrderBoardLane } from "../workOrders/WorkOrderBoardChrome";

function ThemeControls() {
  const selection = useBoardColorTheme();
  return <button onClick={() => selection?.setTheme("spectrum")}>{selection?.theme}</button>;
}

describe("board color themes", () => {
  afterEach(() => localStorage.clear());

  it("keeps a saved choice isolated to its board and restores the default", () => {
    persistBoardColorTheme("a", "pastel");
    expect(readBoardColorTheme("a")).toBe("pastel");
    expect(readBoardColorTheme("b")).toBe("default");
    persistBoardColorTheme("a", "default");
    expect(localStorage.getItem("factories-board-theme:a")).toBeNull();
  });

  it("ignores unknown saved palettes", () => {
    localStorage.setItem("factories-board-theme:a", "deleted-theme");
    expect(readBoardColorTheme("a")).toBe("default");
  });

  it("loads the correct selection when navigating to another board and after remount", () => {
    const view = render(
      <BoardColorThemeProvider boardKey="a">
        <ThemeControls />
      </BoardColorThemeProvider>,
    );
    fireEvent.click(screen.getByRole("button", { name: "default" }));
    expect(screen.getByRole("button", { name: "spectrum" })).toBeInTheDocument();
    view.rerender(
      <BoardColorThemeProvider boardKey="b">
        <ThemeControls />
      </BoardColorThemeProvider>,
    );
    expect(screen.getByRole("button", { name: "default" })).toBeInTheDocument();
    view.unmount();
    render(
      <BoardColorThemeProvider boardKey="a">
        <ThemeControls />
      </BoardColorThemeProvider>,
    );
    expect(screen.getByRole("button", { name: "spectrum" })).toBeInTheDocument();
  });

  it("offers all five palettes and applies the selected palette to the board", async () => {
    const user = userEvent.setup();
    render(
      <BoardColorThemeProvider boardKey="menu">
        <LineBoardViewMenu colorView="vivid" onColorViewChange={vi.fn()} />
        <WorkOrderBoardLane
          title="Implement"
          count={0}
          emptyDescription="Nothing here."
          paletteRole="implement"
          paletteView="vivid"
          testId="palette-lane"
        />
      </BoardColorThemeProvider>,
    );
    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.hover(screen.getByTestId("lines-board-view-theme"));
    await screen.findByTestId("lines-board-theme-flow");
    for (const name of ["SuperPlane Flow", "Quiet Brand", "Clear System", "Clean Pastel", "Cool Spectrum"]) {
      expect(screen.getByText(name)).toBeInTheDocument();
    }
    fireEvent.click(screen.getByTestId("lines-board-theme-quiet"));
    expect(screen.getByTestId("palette-lane")).toHaveAttribute("data-board-theme", "quiet");
    expect(readBoardColorTheme("menu")).toBe("quiet");
    expect(screen.getByTestId("lines-board-view-column-color")).toHaveTextContent("Vivid");
  });

  it("still allows an in-memory selection if browser storage is disabled", () => {
    const getItem = vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
      throw new Error("disabled");
    });
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new Error("disabled");
    });
    try {
      render(
        <BoardColorThemeProvider boardKey="a">
          <ThemeControls />
        </BoardColorThemeProvider>,
      );
      act(() => fireEvent.click(screen.getByRole("button", { name: "default" })));
      expect(screen.getByRole("button", { name: "spectrum" })).toBeInTheDocument();
    } finally {
      getItem.mockRestore();
      setItem.mockRestore();
    }
  });
});
