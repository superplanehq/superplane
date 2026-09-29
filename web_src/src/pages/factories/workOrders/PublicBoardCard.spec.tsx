import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";

import { PublicBoardCard } from "./WorkOrderCard";

describe("PublicBoardCard", () => {
  it("does not render a link or open a dialog", async () => {
    const user = userEvent.setup();
    render(<PublicBoardCard card={{ title: "Ship the board", createdAt: "2026-09-29T12:00:00Z" }} />);

    expect(screen.getByText("Ship the board")).toBeTruthy();
    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.queryByRole("button")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();

    await user.click(screen.getByText("Ship the board"));
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
