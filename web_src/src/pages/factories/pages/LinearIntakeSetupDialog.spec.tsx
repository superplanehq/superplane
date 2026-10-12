import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { LinearIntakeSetupDialog } from "./LinearIntakeSetupDialog";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createIntegration: vi.fn(),
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
  linearDefinition: { name: "linear", label: "Linear", hostedAppInstall: false } as {
    name: string;
    label: string;
    hostedAppInstall: boolean;
  },
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: mocks.connected,
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [mocks.linearDefinition], isLoading: false }),
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: [
      { id: "project-1", name: "Checkout" },
      { id: "project-2", name: "Billing" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="linear-credential-dialog" /> : null,
}));

function renderDialog(onCreated = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/linear"]}>
      <LinearIntakeSetupDialog organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} onCreated={onCreated} />
    </MemoryRouter>,
  );
}

describe("LinearIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.linearDefinition.hostedAppInstall = false;
    mocks.connected.splice(0, mocks.connected.length, {
      metadata: { id: "integration-1", name: "Acme Linear", integrationName: "linear" },
      status: { state: "ready" },
    });
  });

  it("opens the project step when a ready Linear connection already exists", async () => {
    renderDialog();

    expect(
      await screen.findByRole("heading", { name: LINEAR_INTAKE_SETUP_COPY.wizardStepProject }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("linear-skip-initial-import")).toBeChecked();
    expect(screen.getByText(LINEAR_INTAKE_SETUP_COPY.importExistingHelper)).toBeInTheDocument();
    expect(screen.getByTestId("linear-setup-finish")).toBeDisabled();
  });

  it("creates the intake for the selected projects and every label", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    expect(
      await screen.findByRole("heading", { name: LINEAR_INTAKE_SETUP_COPY.wizardStepProject }),
    ).toBeInTheDocument();
    expect(screen.queryByText(LINEAR_INTAKE_SETUP_COPY.labelsLabel)).not.toBeInTheDocument();
    expect(screen.queryByText(LINEAR_INTAKE_SETUP_COPY.labelsHelper)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.labelNew })).not.toBeInTheDocument();

    await user.click(screen.getByTestId("linear-project-project-1"));
    await user.click(screen.getByTestId("linear-project-project-2"));
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_LINEAR_ISSUES",
        integrationId: "integration-1",
        settings: {
          linearProjectIds: ["project-1", "project-2"],
          linearLabels: [],
        },
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("shows only Connect Linear when the hosted application is available and no connection exists", async () => {
    mocks.linearDefinition.hostedAppInstall = true;
    mocks.connected.splice(0, mocks.connected.length);
    renderDialog();

    expect(await screen.findByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.wizardConnect })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use your own Linear app" })).not.toBeInTheDocument();
  });

  it("opens the credential dialog when the hosted Linear application is not configured", async () => {
    mocks.linearDefinition.hostedAppInstall = false;
    mocks.connected.splice(0, mocks.connected.length);
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByRole("button", { name: LINEAR_INTAKE_SETUP_COPY.wizardConnect }));

    expect(await screen.findByTestId("linear-credential-dialog")).toBeInTheDocument();
  });

  it("skips the initial import when the checkbox is off", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("linear-project-project-1"));
    await user.click(screen.getByTestId("linear-skip-initial-import"));
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith(
        expect.objectContaining({
          skipInitialImport: true,
          settings: { linearProjectIds: ["project-1"], linearLabels: [] },
        }),
      );
    });
  });
});
