import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { catalogForColumn } from "../lib/columnAutomations";
import { AddColumnAutomationPicker } from "./AddColumnAutomationPicker";

describe("AddColumnAutomationPicker", () => {
  it("offers the catalog for the column", () => {
    render(
      <AddColumnAutomationPicker
        open
        onClose={vi.fn()}
        onSelect={vi.fn()}
        catalog={catalogForColumn("verify", { allowCustom: true, allowRiskScore: true })}
      />,
    );

    expect(screen.getByRole("heading", { name: "Add automation" })).toBeInTheDocument();
    expect(screen.getByTestId("add-column-automation-template-discussion")).toHaveTextContent(
      "Pull request discussion",
    );
    expect(screen.getByTestId("add-column-automation-template-checks")).toHaveTextContent("Pull request checks");
    expect(screen.getByTestId("add-column-automation-template-risk-score")).toHaveTextContent("Risk score");
    expect(screen.getByTestId("add-column-automation-template-risk-score")).toHaveTextContent(
      "Score the pull request diff from 1 to 5.",
    );
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
    expect(screen.getByTestId("add-column-automation-template-checks")).toBeEnabled();
  });
});
