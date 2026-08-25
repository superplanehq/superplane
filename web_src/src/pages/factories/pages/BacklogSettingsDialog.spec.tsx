import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { BacklogSettingsDialog } from "./BacklogSettingsDialog";

describe("BacklogSettingsDialog", () => {
  it("saves the name and a positive size", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();

    render(<BacklogSettingsDialog open name="Backlog" size={null} onSave={onSave} onClose={vi.fn()} />);

    const name = screen.getByLabelText("Name");
    const size = screen.getByLabelText("Size");
    await user.clear(name);
    await user.type(name, "Inbox");
    await user.type(size, "8");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(onSave).toHaveBeenCalledWith({ name: "Inbox", size: 8 });
  });

  it("keeps size empty when the field is blank", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();

    render(<BacklogSettingsDialog open name="Backlog" size={12} onSave={onSave} onClose={vi.fn()} />);

    await user.clear(screen.getByLabelText("Size"));
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(onSave).toHaveBeenCalledWith({ name: "Backlog", size: null });
  });

  it("rejects a size that is not a positive whole number", async () => {
    const onSave = vi.fn();
    const user = userEvent.setup();

    render(<BacklogSettingsDialog open name="Backlog" size={null} onSave={onSave} onClose={vi.fn()} />);

    await user.type(screen.getByLabelText("Size"), "0");
    await user.click(screen.getByTestId("lines-backlog-settings-save"));

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByText("Enter a whole number of 1 or more.")).toBeInTheDocument();
  });
});
