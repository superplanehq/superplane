import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { NewWorkOrderPage } from "./NewWorkOrderPage";
import { SoftwareFactoryPage } from "./SoftwareFactoryPage";
import { WorkOrderPage } from "./WorkOrderPage";
import type { Automation, SoftwareFactory, WorkOrder, WorkOrderEvent } from "./factoryTypes";

const factory: SoftwareFactory = {
  id: "factory-1",
  name: "Payments Factory",
  description: "Delegated implementation work for the payments platform.",
};

const draftWorkOrder: WorkOrder = {
  id: "wo-1",
  title: "Add refund reconciliation test",
  description: "Cover the refund reconciliation path before the next release.",
  state: "draft",
  createdByUserId: "user-darko",
  createdByName: "Darko",
  createdAt: "2026-07-30T08:00:00Z",
  updatedAt: "2026-07-30T08:00:00Z",
  automations: [
    { id: "automation-1", name: "Implementation pipeline", state: "planned" },
    { id: "automation-2", name: "Verification pipeline", state: "running" },
  ],
};

const automation: Automation = {
  id: "automation-1",
  name: "Implementation pipeline",
  description: "Implements approved Work Orders and opens a pull request.",
  state: "running",
  runningCount: 2,
  queuedCount: 3,
  lastRunAt: "2026-07-30T07:00:00Z",
};

const idleAutomation: Automation = {
  ...automation,
  id: "automation-2",
  name: "Verification pipeline",
  state: "idle",
  runningCount: 0,
  queuedCount: 0,
};

const createdEvent: WorkOrderEvent = {
  id: "event-1",
  kind: "created",
  summary: "Work Order created",
  actor: "Darko",
  occurredAt: "2026-07-30T08:00:00Z",
};

describe("SoftwareFactoryPage", () => {
  it("keeps Work Orders and Automations on one operational page", () => {
    const { container } = render(
      <SoftwareFactoryPage
        factory={factory}
        workOrders={[draftWorkOrder]}
        automations={[automation, idleAutomation]}
        currentUserId="user-darko"
        onNewWorkOrder={vi.fn()}
        onOpenWorkOrder={vi.fn()}
        onCreateAutomation={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    expect(screen.getByRole("heading", { name: "Work Orders" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Automations" })).toBeInTheDocument();
    expect(screen.queryByRole("tab")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open Canvas" })).not.toBeInTheDocument();
    expect(screen.getByText("Created by Darko")).toBeInTheDocument();
    expect(screen.getByLabelText("Implementation pipeline: Planned")).toBeInTheDocument();
    expect(screen.getByLabelText("Verification pipeline: Running")).toBeInTheDocument();
    expect(screen.getByLabelText("2 running now")).toBeInTheDocument();
    expect(screen.getByLabelText("3 in queue")).toBeInTheDocument();
    expect(screen.queryByText("Work Order ready")).not.toBeInTheDocument();
    expect(screen.queryByText(/nodes?/i)).not.toBeInTheDocument();
    expect(screen.queryByTitle("Active")).not.toBeInTheDocument();
    expect(container.querySelector(".lucide-bot")).not.toBeInTheDocument();
    expect(container.querySelector(".lucide-circle-dashed")).toBeInTheDocument();
  });

  it("requests the dedicated creation page from New Work Order", async () => {
    const user = userEvent.setup();
    const onNewWorkOrder = vi.fn();

    render(
      <SoftwareFactoryPage
        factory={factory}
        workOrders={[draftWorkOrder]}
        automations={[automation]}
        currentUserId="user-darko"
        onNewWorkOrder={onNewWorkOrder}
        onOpenWorkOrder={vi.fn()}
        onCreateAutomation={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "New Work Order" }));

    expect(onNewWorkOrder).toHaveBeenCalledOnce();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("combines ownership and status filters", async () => {
    const user = userEvent.setup();
    const successfulWorkOrder: WorkOrder = {
      ...draftWorkOrder,
      id: "wo-2",
      title: "Fix settlement rounding",
      state: "successful",
      createdByUserId: "user-maya",
    };
    const runningWorkOrder: WorkOrder = {
      ...draftWorkOrder,
      id: "wo-3",
      title: "Reject unsigned payment webhooks",
      state: "running",
    };

    render(
      <SoftwareFactoryPage
        factory={factory}
        workOrders={[draftWorkOrder, successfulWorkOrder, runningWorkOrder]}
        automations={[automation]}
        currentUserId="user-darko"
        onNewWorkOrder={vi.fn()}
        onOpenWorkOrder={vi.fn()}
        onCreateAutomation={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Mine" }));

    expect(screen.getByRole("button", { name: draftWorkOrder.title })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: runningWorkOrder.title })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: successfulWorkOrder.title })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Running" }));

    expect(screen.getByRole("button", { name: runningWorkOrder.title })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: draftWorkOrder.title })).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Successful" }));

    expect(screen.getByText("No matching Work Orders")).toBeInTheDocument();
  });
});

describe("NewWorkOrderPage", () => {
  it("creates a draft with the selected Automations", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();

    render(
      <NewWorkOrderPage
        factory={factory}
        automations={[automation, idleAutomation]}
        onCancel={vi.fn()}
        onCreate={onCreate}
      />,
    );

    const description = screen.getByLabelText("Description");
    expect(description).toHaveAttribute("rows", "14");

    await user.type(screen.getByLabelText("Title"), "Prevent duplicate webhook delivery");
    await user.type(description, "Reuse the delivery key when GitHub retries a webhook.");
    await user.click(screen.getByRole("checkbox", { name: /Implementation pipeline/ }));
    await user.click(screen.getByRole("checkbox", { name: /Verification pipeline/ }));
    await user.click(screen.getByRole("button", { name: "Create draft" }));

    expect(onCreate).toHaveBeenCalledWith({
      title: "Prevent duplicate webhook delivery",
      description: "Reuse the delivery key when GitHub retries a webhook.",
      automationIds: ["automation-1", "automation-2"],
    });
  });
});

describe("WorkOrderPage", () => {
  it("offers approval only while the Work Order is a draft", async () => {
    const user = userEvent.setup();
    const onApprove = vi.fn();

    render(
      <WorkOrderPage
        factory={factory}
        workOrder={draftWorkOrder}
        events={[createdEvent]}
        onBack={vi.fn()}
        onApprove={onApprove}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Approve and queue" }));
    expect(onApprove).toHaveBeenCalledOnce();
  });

  it("shows the produced pull request on a successful Work Order", () => {
    render(
      <WorkOrderPage
        factory={factory}
        workOrder={{
          ...draftWorkOrder,
          state: "successful",
          primaryPullRequest: {
            repository: "superplanehq/superplane",
            number: 6412,
            url: "https://github.com/superplanehq/superplane/pull/6412",
          },
        }}
        events={[createdEvent]}
        onBack={vi.fn()}
        onApprove={vi.fn()}
      />,
    );

    expect(screen.getByRole("link", { name: "superplanehq/superplane #6412" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve and queue" })).not.toBeInTheDocument();
  });
});
