import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { LineBoardViewMenu } from "./LineBoardViewMenu";

describe("LineBoardViewMenu", () => {
  it("opens View options and reports the selected layout", async () => {
    const user = userEvent.setup();
    const onViewChange = vi.fn();
    render(<LineBoardViewMenu view="names" onViewChange={onViewChange} />);

    await user.click(screen.getByTestId("lines-board-view-menu"));
    expect(screen.getByText("View options")).toBeInTheDocument();
    await user.click(screen.getByTestId("lines-board-view-icons"));

    expect(onViewChange).toHaveBeenCalledWith("icons");
  });
});
