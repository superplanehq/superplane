import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { CustomAutomationRunAction } from "./CustomAutomationRunAction";

const runAutomation = vi.fn();
const mutation = { mutateAsync: runAutomation, isPending: false };

vi.mock("@/hooks/useFactoryCustomAutomationData", () => ({
  useRunFactoryCustomAutomation: () => mutation,
}));

beforeEach(() => {
  runAutomation.mockReset();
  mutation.isPending = false;
});

describe("CustomAutomationRunAction", () => {
  it("starts the only manual automation and confirms success", async () => {
    const user = userEvent.setup();
    runAutomation.mockResolvedValue({ runId: "run-1" });
    render(
      <CustomAutomationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        automations={[{ id: "automation-1", name: "Security review" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));

    expect(runAutomation).toHaveBeenCalledWith({
      automationId: "automation-1",
      workOrderId: "task-1",
    });
    expect(await screen.findByRole("status")).toHaveTextContent("Security review started.");
  });

  it("lets the user choose one of multiple manual automations", async () => {
    const user = userEvent.setup();
    runAutomation.mockResolvedValue({ runId: "run-2" });
    render(
      <CustomAutomationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        automations={[
          { id: "automation-1", name: "Security review" },
          { id: "automation-2", name: "License review" },
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));
    await user.click(screen.getByRole("menuitem", { name: "License review" }));

    expect(runAutomation).toHaveBeenCalledWith({
      automationId: "automation-2",
      workOrderId: "task-1",
    });
  });

  it("shows an actionable error", async () => {
    const user = userEvent.setup();
    runAutomation.mockRejectedValue(new Error("The task has no pull request."));
    render(
      <CustomAutomationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        automations={[{ id: "automation-1", name: "Security review" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The task has no pull request.");
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("shows the start error in the task log", async () => {
    const user = userEvent.setup();
    runAutomation.mockRejectedValue(new Error("The task is not in Verify."));
    render(
      <CustomAutomationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        placement="logs"
        automations={[{ id: "automation-1", name: "Security review" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The task is not in Verify.");
  });

  it("does not confirm a started run in the task log", async () => {
    const user = userEvent.setup();
    runAutomation.mockResolvedValue({ runId: "run-1" });
    render(
      <CustomAutomationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        placement="logs"
        automations={[{ id: "automation-1", name: "Deploy to staging" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run" }));

    await waitFor(() => expect(runAutomation).toHaveBeenCalled());
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.queryByText("Deploy to staging started.")).not.toBeInTheDocument();
  });
});
