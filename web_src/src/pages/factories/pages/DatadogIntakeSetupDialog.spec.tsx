import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DatadogIntakeSetupDialog } from "./DatadogIntakeSetupDialog";
import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";
import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createIntegration: vi.fn(),
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
  services: [
    { id: "checkout", name: "checkout" },
    { id: "billing", name: "billing" },
  ] as Array<{ id: string; name: string }>,
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
  useAvailableIntegrations: () => ({ data: [{ name: "datadog", label: "Datadog" }] }),
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: mocks.services,
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

function renderDialog(onCreated = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/datadog"]}>
      <DatadogIntakeSetupDialog organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} onCreated={onCreated} />
    </MemoryRouter>,
  );
}

describe("DatadogIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.createIntegration.mockReset();
    mocks.connected.splice(0, mocks.connected.length);
    mocks.services.splice(
      0,
      mocks.services.length,
      { id: "checkout", name: "checkout" },
      { id: "billing", name: "billing" },
    );
  });

  it("asks for Datadog keys when the organization has no connection", () => {
    renderDialog();

    expect(screen.getByRole("heading", { name: DATADOG_INTAKE_SETUP_COPY.wizardStepConnect })).toBeInTheDocument();
    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.wizardStepConnectHelper)).toBeInTheDocument();
    expect(screen.getByTestId("datadog-setup-connect")).toBeInTheDocument();
    expect(screen.getByText("Choose service")).toBeInTheDocument();
    expect(screen.queryByText("Choose project")).not.toBeInTheDocument();
    expect(screen.getByTestId("datadog-setup-continue")).toBeDisabled();
  });

  it("opens the key form instead of an install page", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("datadog-setup-connect"));

    expect(screen.getByTestId("finish-connect")).toBeInTheDocument();
    expect(mocks.createIntegration).not.toHaveBeenCalled();
  });

  it("creates the intake for the connection the key form returns", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await user.click(screen.getByTestId("datadog-setup-connect"));
    await user.click(screen.getByTestId("finish-connect"));
    await user.click(await screen.findByTestId("datadog-service-checkout"));
    await user.click(screen.getByTestId("datadog-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_DATADOG",
        integrationId: "integration-new",
        resourceId: "checkout",
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("opens the service step when a ready Datadog connection already exists", async () => {
    mocks.connected.push({
      metadata: { id: "integration-1", name: "Datadog", integrationName: "datadog" },
      status: { state: "ready" },
    });
    renderDialog();

    expect(
      await screen.findByRole("heading", { name: DATADOG_INTAKE_SETUP_COPY.wizardStepService }),
    ).toBeInTheDocument();
    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.wizardStepServiceHelper)).toBeInTheDocument();
    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.importExistingHelper)).toHaveTextContent("10 newest");
    expect(screen.getByTestId("datadog-skip-initial-import")).toBeChecked();
    expect(screen.getByTestId("datadog-setup-finish")).toBeDisabled();
  });

  it("creates the intake for a selected Datadog service", async () => {
    mocks.connected.push({
      metadata: { id: "integration-1", name: "Datadog", integrationName: "datadog" },
      status: { state: "ready" },
    });
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await user.click(await screen.findByTestId("datadog-service-checkout"));
    await user.click(screen.getByTestId("datadog-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_DATADOG",
        integrationId: "integration-1",
        resourceId: "checkout",
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates the intake without importing existing errors", async () => {
    mocks.connected.push({
      metadata: { id: "integration-1", name: "Datadog", integrationName: "datadog" },
      status: { state: "ready" },
    });
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    expect(await screen.findByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label)).toBeInTheDocument();
    expect(screen.getByTestId("datadog-skip-initial-import")).toBeChecked();
    await user.click(screen.getByTestId("datadog-skip-initial-import"));
    expect(screen.getByText(DATADOG_INTAKE_SETUP_COPY.importExistingHelperOff)).toBeInTheDocument();
    await user.click(screen.getByTestId("datadog-service-checkout"));
    await user.click(screen.getByTestId("datadog-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_DATADOG",
        integrationId: "integration-1",
        resourceId: "checkout",
        skipInitialImport: true,
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates the intake for a typed service name", async () => {
    mocks.connected.push({
      metadata: { id: "integration-1", name: "Datadog", integrationName: "datadog" },
      status: { state: "ready" },
    });
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await screen.findByTestId("datadog-service-name");
    await user.type(screen.getByTestId("datadog-service-name"), "payments-api");
    await user.click(screen.getByTestId("datadog-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_DATADOG",
        integrationId: "integration-1",
        resourceId: "payments-api",
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("shows the create fallback when SuperPlane returns internal error", async () => {
    mocks.connected.push({
      metadata: { id: "integration-1", name: "Datadog", integrationName: "datadog" },
      status: { state: "ready" },
    });
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "internal error" } } });
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("datadog-service-checkout"));
    await user.click(screen.getByTestId("datadog-setup-finish"));

    expect(await screen.findByRole("alert")).toHaveTextContent(DATADOG_INTAKE_SETUP_COPY.wizardCreateError);
  });
});
