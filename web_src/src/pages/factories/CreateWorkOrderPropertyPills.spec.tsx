import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { REFUND_FACTORY_LINES } from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderPropertyPills } from "./CreateWorkOrderPropertyPills";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      {
        metadata: { id: "user-reviewer-alex", email: "alex@superplane.dev" },
        spec: { displayName: "Alex Reviewer" },
      },
    ],
    isLoading: false,
  }),
}));

describe("CreateWorkOrderPropertyPills", () => {
  it("shows the owner picker inside the dialog when Owner is clicked", async () => {
    const user = userEvent.setup();
    render(
      <CreateWorkOrderPropertyPills
        organizationId="org-1"
        assigneeIds={[]}
        lines={REFUND_FACTORY_LINES}
        selectedLineName=""
        isSaving={false}
        onAssigneeChange={vi.fn()}
        onLineSelect={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("work-order-assignees-button"));

    const panel = screen.getByTestId("work-order-assignee-picker-panel");
    expect(panel).toBeInTheDocument();
    expect(
      within(panel).getByText((_, element) => element?.tagName === "SPAN" && element.textContent === "Alex Reviewer"),
    ).toBeInTheDocument();
    expect(screen.getByTestId("work-order-save-assignees")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-assignees-button")).toHaveAttribute("aria-expanded", "true");
  });

  it("shows the line picker inside the dialog when Line is clicked", async () => {
    const user = userEvent.setup();
    render(
      <CreateWorkOrderPropertyPills
        organizationId="org-1"
        assigneeIds={[]}
        lines={REFUND_FACTORY_LINES}
        selectedLineName=""
        isSaving={false}
        onAssigneeChange={vi.fn()}
        onLineSelect={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("work-order-line-button"));

    expect(screen.getByTestId("work-order-line-picker-panel")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-line-option-plan-and-implement")).toBeInTheDocument();
  });

  it("does not open the line picker when the user cannot dispatch", async () => {
    const user = userEvent.setup();
    render(
      <CreateWorkOrderPropertyPills
        organizationId="org-1"
        assigneeIds={[]}
        lines={REFUND_FACTORY_LINES}
        selectedLineName=""
        isSaving={false}
        canDispatch={false}
        onAssigneeChange={vi.fn()}
        onLineSelect={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("work-order-line-button"));

    expect(screen.queryByTestId("work-order-line-picker-panel")).not.toBeInTheDocument();
  });
});
