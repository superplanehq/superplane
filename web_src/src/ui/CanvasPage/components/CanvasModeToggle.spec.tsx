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
});
