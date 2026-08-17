import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { WorkOrderReactionsBar } from "./WorkOrderReactionsBar";
import { WorkOrderReactionsProvider } from "./WorkOrderReactionsProvider";

function renderBar(workOrderId: string) {
  return render(
    <WorkOrderReactionsProvider>
      <WorkOrderReactionsBar workOrderId={workOrderId} />
    </WorkOrderReactionsProvider>,
  );
}

describe("WorkOrderReactionsBar", () => {
  it("renders the seeded reactions for a work order, with the viewer's own reaction highlighted", () => {
    renderBar("wo-open-refunds");

    const minePill = screen.getByTestId("work-order-reaction-pill-👍");
    expect(minePill).toHaveTextContent("3");
    expect(minePill).toHaveAttribute("aria-pressed", "true");
  });

  it("has no seeded reactions for a work order outside the fixture — just Add reaction", () => {
    renderBar("wo-draft-refunds");

    expect(screen.queryByTestId(/work-order-reaction-pill-/)).toBeNull();
    expect(screen.getByTestId("work-order-reaction-add")).toHaveTextContent("Add reaction");
  });

  it("adding a reaction from the picker joins that group as mine, and clicking it again removes it", async () => {
    const user = userEvent.setup();
    renderBar("wo-running-refunds");

    await user.click(screen.getByTestId("work-order-reaction-add"));
    await user.click(screen.getByTestId("work-order-reaction-option-🚀"));

    const minePill = screen.getByTestId("work-order-reaction-pill-🚀");
    expect(minePill).toHaveTextContent("1");
    expect(minePill).toHaveAttribute("aria-pressed", "true");

    await user.click(minePill);
    expect(screen.queryByTestId("work-order-reaction-pill-🚀")).toBeNull();
  });
});
