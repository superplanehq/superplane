import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { catalogForColumn } from "../lib/columnAutomations";
import { AddColumnAutomationPicker } from "./AddColumnAutomationPicker";

describe("AddColumnAutomationPicker", () => {
  it("offers the catalog for the column", () => {
    render(
      <AddColumnAutomationPicker open onClose={vi.fn()} onSelect={vi.fn()} catalog={catalogForColumn("verify")} />,
    );

    expect(screen.getByRole("heading", { name: "Add automation" })).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-discussion")).toHaveTextContent(
      "Pull request discussion",
    );
    expect(screen.queryByTestId("add-column-automation-template-checks")).not.toBeInTheDocument();
  });

  it("reports the chosen entry and disables taken ones", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <AddColumnAutomationPicker
        open
        onClose={vi.fn()}
        onSelect={onSelect}
        catalog={catalogForColumn("verify")}
        takenIds={["discussion"]}
      />,
    );

    const discussion = screen.getByTestId("add-column-automation-template-discussion");
    expect(discussion).toBeDisabled();
    expect(discussion).toHaveTextContent("This automation is already configured.");
    await user.click(discussion);
    expect(onSelect).not.toHaveBeenCalled();
    expect(screen.queryByTestId("add-column-automation-template-checks")).not.toBeInTheDocument();
  });
});
