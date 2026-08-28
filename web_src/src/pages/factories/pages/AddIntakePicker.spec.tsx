import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { AddIntakePicker } from "./AddIntakePicker";

function renderPicker(onSelect = vi.fn(), onClose = vi.fn()) {
  render(<AddIntakePicker open onClose={onClose} onSelect={onSelect} />);
  return { onSelect, onClose };
}

describe("AddIntakePicker", () => {
  it("offers a searchable list of intake templates", () => {
    renderPicker();

    const picker = screen.getByTestId("add-intake-picker");
    expect(within(picker).getByRole("heading", { name: "Add intake" })).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-search")).toBeInTheDocument();
    expect(within(picker).getByTestId("add-intake-template-improve-ci-runtime")).toHaveTextContent(
      "Improve CI runtime",
    );
    expect(within(picker).getAllByTestId(/^add-intake-template-/)).toHaveLength(6);
  });

  it("filters templates from the search field", async () => {
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByTestId("add-intake-search"), "runtime");

    expect(screen.getByTestId("add-intake-template-improve-ci-runtime")).toBeInTheDocument();
    expect(screen.queryByTestId("add-intake-template-flaky-tests")).not.toBeInTheDocument();
  });

  it("reports the chosen template to the caller", async () => {
    const user = userEvent.setup();
    const { onSelect } = renderPicker();

    await user.click(screen.getByTestId("add-intake-template-github-issues"));

    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: "github-issues" }));
  });
});
