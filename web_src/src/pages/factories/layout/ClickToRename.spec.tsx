import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import type { ComponentProps } from "react";

import { ClickToRename } from "./ClickToRename";

function renderRename(overrides: Partial<ComponentProps<typeof ClickToRename>> = {}) {
  const props = {
    value: "Plan and Implement",
    onSave: vi.fn(),
    canEdit: true,
    testId: "rename-target",
    ariaLabel: "Line name",
    ...overrides,
  };
  const view = render(<ClickToRename {...props} />);
  return {
    onSave: props.onSave,
    user: userEvent.setup(),
    ...view,
    rerenderRename: (next: Partial<typeof props>) => view.rerender(<ClickToRename {...props} {...next} />),
  };
}

describe("ClickToRename", () => {
  it("opens an input on click and saves on Enter", async () => {
    const { user, onSave } = renderRename();

    await user.click(screen.getByTestId("rename-target"));
    const input = await screen.findByTestId("rename-target-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "  Refund line  ");
    await user.keyboard("{Enter}");
    fireEvent.blur(input);

    expect(onSave).toHaveBeenCalledWith("Refund line");
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("rename-target-input")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("rename-target"));
    const reopened = screen.getByRole("textbox");
    await waitFor(() => expect(reopened).toHaveFocus());
    await user.clear(reopened);
    await user.type(reopened, "Inbox");
    await user.tab();
    expect(onSave).toHaveBeenLastCalledWith("Inbox");
    expect(onSave).toHaveBeenCalledTimes(2);
  });

  it("saves on blur", async () => {
    const { user, onSave } = renderRename();

    await user.click(screen.getByTestId("rename-target"));
    const input = await screen.findByTestId("rename-target-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "  Inbox  ");
    await user.tab();

    expect(onSave).toHaveBeenCalledWith("Inbox");
  });

  it("restores the previous name on Escape", async () => {
    const { user, onSave } = renderRename();

    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByTestId("rename-target-input");
    await user.clear(input);
    await user.type(input, "Scratch");
    await user.keyboard("{Escape}");
    fireEvent.blur(input);

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByTestId("rename-target")).toHaveTextContent("Plan and Implement");

    await user.click(screen.getByTestId("rename-target"));
    const reopened = screen.getByRole("textbox", { name: "Line name" });
    await waitFor(() => expect(reopened).toHaveFocus());
    await user.clear(reopened);
    await user.type(reopened, "Inbox");
    await user.tab();
    expect(onSave).toHaveBeenCalledWith("Inbox");
  });

  it("keeps the label in the document so the layout size does not change", async () => {
    const { user } = renderRename();

    const label = screen.getByTestId("rename-target");
    await user.click(label);
    await waitFor(() => expect(screen.getByTestId("rename-target-input")).toBeInTheDocument());

    expect(screen.getByTestId("rename-target")).toBeInTheDocument();
    expect(screen.getByTestId("rename-target")).toHaveClass("invisible");
  });

  it("does not open when canEdit is false", async () => {
    const { user } = renderRename({ canEdit: false });

    await user.click(screen.getByTestId("rename-target"));
    fireEvent.keyDown(screen.getByTestId("rename-target"), { key: "Enter" });
    fireEvent.keyDown(screen.getByTestId("rename-target"), { key: " " });
    expect(screen.queryByTestId("rename-target-input")).not.toBeInTheDocument();
  });

  it.each(["{Enter}", " "])("opens with %s and selects the current name", async (key) => {
    const { user } = renderRename();

    await user.tab();
    expect(screen.getByTestId("rename-target")).toHaveFocus();
    await user.keyboard(key);
    const input = screen.getByRole<HTMLInputElement>("textbox", { name: "Line name" });
    await waitFor(() => expect(input).toHaveFocus());
    expect(input.selectionStart).toBe(0);
    expect(input.selectionEnd).toBe(input.value.length);
  });

  it("closes an unchanged trimmed name without saving", async () => {
    const { user, onSave } = renderRename({ value: "  Inbox  " });

    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByRole("textbox");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
    await user.keyboard("{Enter}");

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it("validates a blank trimmed name and keeps editing without saving", async () => {
    const validate = vi.fn(() => "");
    const { user, onSave } = renderRename({ validate });

    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByRole("textbox");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "   ");
    await user.tab();

    expect(validate).toHaveBeenCalledWith("");
    expect(onSave).not.toHaveBeenCalled();
    expect(input).toBeInTheDocument();
    expect(input).toHaveFocus();
  });

  it("associates validation errors with the input and clears them on a new edit", async () => {
    const validate = vi.fn(() => "Choose another name.");
    const { user, onSave } = renderRename({ validate });

    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByRole("textbox", { name: "Line name" });
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "  Rejected  ");
    await user.tab();

    expect(validate).toHaveBeenCalledWith("Rejected");
    expect(onSave).not.toHaveBeenCalled();
    expect(input).toHaveFocus();
    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(input).toHaveAttribute("aria-describedby", screen.getByRole("alert").id);

    await user.keyboard("{Escape}");
    await user.click(screen.getByTestId("rename-target"));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Line name" })).toHaveAttribute("aria-invalid", "false");
  });

  it("uses updated values outside editing without overwriting an active draft", async () => {
    const { user, onSave, rerenderRename } = renderRename();
    rerenderRename({ value: "Inbox" });
    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByRole("textbox");
    await waitFor(() => expect(input).toHaveFocus());
    expect(input).toHaveValue("Inbox");
    await user.clear(input);
    await user.type(input, "Draft");
    rerenderRename({ value: "Updated" });
    expect(input).toHaveValue("Draft");

    await user.keyboard("{Escape}");
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByTestId("rename-target")).toHaveTextContent("Updated");
    await user.click(screen.getByTestId("rename-target"));
    expect(screen.getByRole("textbox")).toHaveValue("Updated");
  });

  it("does not start editing while busy", async () => {
    const { user } = renderRename({ busy: true });

    await user.click(screen.getByTestId("rename-target"));
    await user.tab();
    await user.keyboard("{Enter}");
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  it.each([{ busy: true }, { canEdit: false }])("blocks commits after props change to %j", async (next) => {
    const { user, onSave, rerenderRename } = renderRename();
    await user.click(screen.getByTestId("rename-target"));
    const input = screen.getByRole("textbox");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Draft");
    rerenderRename(next);
    expect(input).toHaveProperty("disabled", "busy" in next);

    fireEvent.keyDown(input, { key: "Enter" });
    fireEvent.blur(input);
    expect(onSave).not.toHaveBeenCalled();
    expect(input).toBeInTheDocument();
  });
});
