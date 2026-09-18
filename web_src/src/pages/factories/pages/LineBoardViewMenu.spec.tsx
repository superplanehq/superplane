import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { LineBoardViewMenu } from "./LineBoardViewMenu";

describe("LineBoardViewMenu", () => {
  it("opens View options and reports the selected automation layout", async () => {
    const user = userEvent.setup();
    const onViewChange = vi.fn();
    render(<LineBoardViewMenu view="names" onViewChange={onViewChange} colorView="fill" onColorViewChange={vi.fn()} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.getByText("View options")).toBeInTheDocument();
    await user.click(screen.getByTestId("lines-board-view-icons"));

    expect(onViewChange).toHaveBeenCalledWith("icons");
  });

  it("reports the selected column-color layout", async () => {
    const user = userEvent.setup();
    const onColorViewChange = vi.fn();
    render(<LineBoardViewMenu colorView="fill" onColorViewChange={onColorViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.queryByTestId("lines-board-view-names")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-board-view-column-colors")).toBeInTheDocument();
    await user.click(screen.getByTestId("lines-board-view-no-column-colors"));

    expect(onColorViewChange).toHaveBeenCalledWith("off");
  });

  it("offers colored borders as a column-color layout", async () => {
    const user = userEvent.setup();
    const onColorViewChange = vi.fn();
    render(<LineBoardViewMenu colorView="off" onColorViewChange={onColorViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await user.click(screen.getByTestId("lines-board-view-colored-borders"));

    expect(onColorViewChange).toHaveBeenCalledWith("borders");
  });
});
