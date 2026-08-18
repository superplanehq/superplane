import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { MissionAssignmentProvider } from "./MissionAssignmentContext";
import { WorkOrderMissionOverviewRow } from "./WorkOrderMissionOverviewRow";

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
}));

describe("WorkOrderMissionOverviewRow", () => {
  it("uses the same empty action style as assignee and line", () => {
    render(
      <MissionAssignmentProvider>
        <WorkOrderMissionOverviewRow workOrderId="wo-draft-refunds" />
      </MissionAssignmentProvider>,
    );

    const button = screen.getByTestId("work-order-edit-mission");
    expect(button).toHaveTextContent("Attach to mission");
    expect(button).not.toHaveTextContent("No mission");
    expect(button).toHaveClass("text-muted-foreground");
  });
});
