import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

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
        isSaving={false}
        onAssigneeChange={vi.fn()}
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

  it("does not open the owner picker while saving", async () => {
    const user = userEvent.setup();
    render(
      <CreateWorkOrderPropertyPills organizationId="org-1" assigneeIds={[]} isSaving onAssigneeChange={vi.fn()} />,
    );

    await user.click(screen.getByTestId("work-order-assignees-button"));

    expect(screen.queryByTestId("work-order-assignee-picker-panel")).not.toBeInTheDocument();
  });
});
