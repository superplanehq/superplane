import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { LineBoardViewMenu } from "./LineBoardViewMenu";

async function selectViewOption(user: ReturnType<typeof userEvent.setup>, parentTestId: string, optionTestId: string) {
  await user.hover(screen.getByTestId(parentTestId));
  fireEvent.click(await screen.findByTestId(optionTestId));
}

describe("LineBoardViewMenu", () => {
  it("opens View options and reports the selected automation layout", async () => {
    const user = userEvent.setup();
    const onViewChange = vi.fn();
    render(<LineBoardViewMenu view="names" onViewChange={onViewChange} colorView="dim" onColorViewChange={vi.fn()} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.getByText("View options")).toBeInTheDocument();
    expect(screen.getByTestId("lines-board-view-automations")).toHaveTextContent("Names");
    await selectViewOption(user, "lines-board-view-automations", "lines-board-view-icons");

    expect(onViewChange).toHaveBeenCalledWith("icons");
    expect(screen.getByText("View options")).toBeInTheDocument();
  });

  it("reports the selected column-color layout", async () => {
    const user = userEvent.setup();
    const onColorViewChange = vi.fn();
    render(<LineBoardViewMenu colorView="dim" onColorViewChange={onColorViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.queryByTestId("lines-board-view-automations")).not.toBeInTheDocument();
    expect(screen.getByTestId("lines-board-view-column-color")).toHaveTextContent("Soft");
    await selectViewOption(user, "lines-board-view-column-color", "lines-board-view-no-column-colors");

    expect(onColorViewChange).toHaveBeenCalledWith("off");
    expect(screen.getByText("View options")).toBeInTheDocument();
  });

  it("offers vivid fills as a column-color layout", async () => {
    const user = userEvent.setup();
    const onColorViewChange = vi.fn();
    render(<LineBoardViewMenu colorView="dim" onColorViewChange={onColorViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    await selectViewOption(user, "lines-board-view-column-color", "lines-board-view-vivid-column-colors");

    expect(onColorViewChange).toHaveBeenCalledWith("vivid");
    expect(screen.getByText("View options")).toBeInTheDocument();
  });

  it("offers colored borders as a column-color layout", async () => {
    const user = userEvent.setup();
    const onColorViewChange = vi.fn();
    render(<LineBoardViewMenu colorView="off" onColorViewChange={onColorViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.getByTestId("lines-board-view-column-color")).toHaveTextContent("None");
    await selectViewOption(user, "lines-board-view-column-color", "lines-board-view-colored-borders");

    expect(onColorViewChange).toHaveBeenCalledWith("borders");
    expect(screen.getByText("View options")).toBeInTheDocument();
  });
});
