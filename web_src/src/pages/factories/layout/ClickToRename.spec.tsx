import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ClickToRename } from "./ClickToRename";

function renderRename(onSave = vi.fn(), canEdit = true) {
  return {
    onSave,
    user: userEvent.setup(),
    ...render(
      <ClickToRename
        value="Plan and Implement"
        onSave={onSave}
        canEdit={canEdit}
        testId="rename-target"
        ariaLabel="Line name"
      />,
    ),
  };
}

describe("ClickToRename", () => {
  it("opens an input on click and saves on Enter", async () => {
    const { user, onSave } = renderRename();

    await user.click(screen.getByTestId("rename-target"));
    const input = await screen.findByTestId("rename-target-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Refund line");
    await user.keyboard("{Enter}");

    expect(onSave).toHaveBeenCalledWith("Refund line");
    expect(screen.queryByTestId("rename-target-input")).not.toBeInTheDocument();
  });

  it("saves on blur", async () => {
    const { user, onSave } = renderRename();

    await user.click(screen.getByTestId("rename-target"));
    const input = await screen.findByTestId("rename-target-input");
    await waitFor(() => expect(input).toHaveFocus());
    await user.clear(input);
    await user.type(input, "Inbox");
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

    expect(onSave).not.toHaveBeenCalled();
    expect(screen.getByTestId("rename-target")).toHaveTextContent("Plan and Implement");
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
    const { user } = renderRename(vi.fn(), false);

    await user.click(screen.getByTestId("rename-target"));
    expect(screen.queryByTestId("rename-target-input")).not.toBeInTheDocument();
  });
});
