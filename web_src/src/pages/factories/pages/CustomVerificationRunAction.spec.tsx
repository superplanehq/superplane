import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { CustomVerificationRunAction } from "./CustomVerificationRunAction";

const runVerification = vi.fn();
const mutation = { mutateAsync: runVerification, isPending: false };

vi.mock("@/hooks/useFactoryCustomVerificationData", () => ({
  useRunFactoryCustomVerification: () => mutation,
}));

beforeEach(() => {
  runVerification.mockReset();
  mutation.isPending = false;
});

describe("CustomVerificationRunAction", () => {
  it("starts the only manual verification and confirms success", async () => {
    const user = userEvent.setup();
    runVerification.mockResolvedValue({ runId: "run-1" });
    render(
      <CustomVerificationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        verifications={[{ id: "verification-1", name: "Security review" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));

    expect(runVerification).toHaveBeenCalledWith({
      automationId: "verification-1",
      workOrderId: "task-1",
    });
    expect(await screen.findByRole("status")).toHaveTextContent("Security review started.");
  });

  it("lets the user choose one of multiple manual verifications", async () => {
    const user = userEvent.setup();
    runVerification.mockResolvedValue({ runId: "run-2" });
    render(
      <CustomVerificationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        verifications={[
          { id: "verification-1", name: "Security review" },
          { id: "verification-2", name: "License review" },
        ]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));
    await user.click(screen.getByRole("menuitem", { name: "License review" }));

    expect(runVerification).toHaveBeenCalledWith({
      automationId: "verification-2",
      workOrderId: "task-1",
    });
  });

  it("shows an actionable error", async () => {
    const user = userEvent.setup();
    runVerification.mockRejectedValue(new Error("The task has no pull request."));
    render(
      <CustomVerificationRunAction
        organizationId="org-1"
        factoryId="factory-1"
        workOrderId="task-1"
        verifications={[{ id: "verification-1", name: "Security review" }]}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Run automation" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("The task has no pull request.");
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });
});
