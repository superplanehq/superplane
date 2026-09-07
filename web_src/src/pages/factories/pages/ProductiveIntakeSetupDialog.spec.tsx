import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ProductiveIntakeSetupDialog } from "./ProductiveIntakeSetupDialog";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: [
      {
        metadata: { id: "integration-1", name: "Productive", integrationName: "productive" },
        status: { state: "ready" },
      },
    ],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [{ name: "productive", label: "Productive" }] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: [
      { id: "project-1", name: "Payments" },
      { id: "project-2", name: "Growth" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open, onCreated }: { open: boolean; onCreated: (id: string) => void }) =>
    open ? (
      <button type="button" data-testid="finish-connect" onClick={() => onCreated("integration-new")}>
        Finish connect
      </button>
    ) : null,
}));

describe("ProductiveIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
  });

  it("finishes setup once a connection and a project are chosen", async () => {
    const user = userEvent.setup();
    render(<ProductiveIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "Connect Productive.io" })).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: "Select project" })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: "Select project" }));

    expect(screen.getByRole("heading", { name: "Choose a project" })).toBeInTheDocument();
    await user.click(screen.getByTestId("productive-project-project-1"));
    await user.click(screen.getByRole("button", { name: "Finish setup" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_PRODUCTIVE_TASKS",
        integrationId: "integration-1",
        resourceId: "project-1",
      });
    });

    // The backend seeds the newest open tasks, so the wizard never asks the
    // user to pick tasks by hand.
    expect(screen.getByRole("heading", { name: "Setup complete" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "SuperPlane is adding the newest open tasks to the Backlog. New tasks arrive as your team creates them.",
      ),
    ).toBeInTheDocument();
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    render(<ProductiveIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Select project" }));
    await user.type(screen.getByLabelText("Search projects"), "grow");

    expect(screen.queryByTestId("productive-project-project-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("productive-project-project-2")).toBeInTheDocument();
  });

  it("opens project selection as soon as a new account is connected", async () => {
    const user = userEvent.setup();
    render(<ProductiveIntakeSetupDialog open organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Connect another account" }));
    await user.click(screen.getByTestId("finish-connect"));

    expect(screen.getByRole("heading", { name: "Choose a project" })).toBeInTheDocument();
    expect(screen.getByTestId("productive-project-project-1")).toBeInTheDocument();
  });
});
