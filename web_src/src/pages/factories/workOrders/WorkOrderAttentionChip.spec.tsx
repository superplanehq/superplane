import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { WorkOrderChecksPassedMark } from "./WorkOrderAttentionChip";

describe("WorkOrderChecksPassedMark", () => {
  it("keeps 'Status checks passed' as the accessible name without a native title", () => {
    render(<WorkOrderChecksPassedMark />);

    const mark = screen.getByLabelText("Status checks passed");
    expect(mark).not.toHaveAttribute("title");
  });

  it("shows a tooltip explaining the check on hover", async () => {
    const user = userEvent.setup();
    render(<WorkOrderChecksPassedMark />);

    await user.hover(screen.getByLabelText("Status checks passed"));

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("All checks on the pull request have passed.");
  });

  it("shows the tooltip on keyboard focus", async () => {
    const user = userEvent.setup();
    render(<WorkOrderChecksPassedMark />);

    await user.tab();

    const tip = await screen.findByRole("tooltip");
    expect(tip).toHaveTextContent("All checks on the pull request have passed.");
  });
});
