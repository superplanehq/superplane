import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ParallelismSettingsDialog } from "./ParallelismSettingsDialog";

describe("ParallelismSettingsDialog", () => {
  it("saves a value from the input", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();

    render(<ParallelismSettingsDialog open value={10} onSave={onSave} onClose={vi.fn()} />);

    const input = screen.getByTestId("lines-parallelism-input");
    expect(input).toHaveValue(10);
    expect(screen.getByRole("slider")).toHaveAttribute("aria-valuenow", "10");

    await user.clear(input);
    await user.type(input, "25");
    await user.click(screen.getByTestId("lines-parallelism-save"));

    expect(onSave).toHaveBeenCalledWith(25);
  });

  it("rejects a value outside 1 to 100", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();

    render(<ParallelismSettingsDialog open value={10} onSave={onSave} onClose={vi.fn()} />);

    const input = screen.getByTestId("lines-parallelism-input");
    await user.clear(input);
    await user.type(input, "0");
    await user.click(screen.getByTestId("lines-parallelism-save"));

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText("Enter a whole number from 1 to 100.")).toBeInTheDocument();
  });
});
