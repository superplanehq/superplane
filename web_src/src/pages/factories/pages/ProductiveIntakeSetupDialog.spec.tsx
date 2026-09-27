import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";
import { ProductiveIntakeSetupDialog } from "./ProductiveIntakeSetupDialog";
import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
  projectsError: false,
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
  useAvailableIntegrations: () => ({ data: [{ name: "productive", label: "Productive" }] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: mocks.projectsError
      ? []
      : [
          { id: "project-1", name: "Payments" },
          { id: "project-2", name: "Growth" },
        ],
    isLoading: false,
    isError: mocks.projectsError,
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
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/productive"]}>
      <ProductiveIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        integrationsBasePath="/org-1/workspaces/sp/settings/organization/integrations"
        onClose={vi.fn()}
        onCreated={onCreated}
      />
    </MemoryRouter>,
  );
}

describe("ProductiveIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.projectsError = false;
    mocks.connected.splice(0, mocks.connected.length, {
      metadata: { id: "integration-1", name: "Productive", integrationName: "productive" },
      status: { state: "ready" },
    });
  });

  it("opens the project step when a ready Productive.io connection exists", async () => {
    renderDialog();

    expect(
      await screen.findByRole("heading", { name: PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProject }),
    ).toBeInTheDocument();
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent(
      "listens for new tasks",
    );
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.importExistingHelper)).toHaveTextContent("10 newest");
    expect(screen.getByTestId("productive-skip-initial-import")).toBeChecked();
    expect(screen.getByTestId("productive-intake-setup-stepper")).toBeInTheDocument();
    expect(screen.getByTestId("productive-intake-setup-sphere")).toBeInTheDocument();
  });

  it("creates a bound intake after a project is chosen", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await user.click(await screen.findByTestId("productive-project-project-1"));
    await user.click(screen.getByTestId("productive-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_PRODUCTIVE_TASKS",
        integrationId: "integration-1",
        resourceId: "project-1",
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates a bound intake without importing existing issues", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await screen.findByTestId("productive-project-project-1");
    expect(screen.getByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label)).toBeInTheDocument();
    expect(screen.getByTestId("productive-skip-initial-import")).toBeChecked();
    await user.click(screen.getByTestId("productive-skip-initial-import"));
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.importExistingHelperOff)).toBeInTheDocument();
    await user.click(screen.getByTestId("productive-project-project-1"));
    await user.click(screen.getByTestId("productive-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_PRODUCTIVE_TASKS",
        integrationId: "integration-1",
        resourceId: "project-1",
        skipInitialImport: true,
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.type(await screen.findByLabelText("Search projects"), "grow");

    expect(screen.queryByTestId("productive-project-project-1")).not.toBeInTheDocument();
    expect(screen.getByTestId("productive-project-project-2")).toBeInTheDocument();
  });

  it("opens project selection as soon as a new account is connected", async () => {
    mocks.connected.splice(0, mocks.connected.length);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("productive-setup-connect"));
    await user.click(screen.getByTestId("finish-connect"));

    expect(screen.getByRole("heading", { name: PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProject })).toBeInTheDocument();
    expect(screen.getByTestId("productive-project-project-1")).toBeInTheDocument();
  });

  it("binds the intake to the selected Productive.io connection", async () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      {
        metadata: { id: "integration-1", name: "Productive", integrationName: "productive" },
        status: { state: "ready" },
      },
      {
        metadata: { id: "integration-2", name: "Productive EU", integrationName: "productive" },
        status: { state: "ready" },
      },
    );
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("first-run-back"));
    await user.click(screen.getByTestId("productive-connection-integration-2"));
    await user.click(screen.getByTestId("productive-setup-continue"));
    await user.click(await screen.findByTestId("productive-project-project-1"));
    await user.click(screen.getByTestId("productive-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_PRODUCTIVE_TASKS",
        integrationId: "integration-2",
        resourceId: "project-1",
      });
    });
  });

  it("links to the connection when the project list cannot load", async () => {
    mocks.projectsError = true;
    renderDialog();

    expect(await screen.findByText(PRODUCTIVE_INTAKE_SETUP_COPY.wizardProjectsError)).toBeInTheDocument();
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.wizardProjectsErrorHint)).toBeInTheDocument();
    expect(screen.getByTestId("productive-setup-check-connection")).toHaveAttribute(
      "href",
      "/org-1/workspaces/sp/settings/organization/integrations/integration-1",
    );
  });

  it("offers another account when a broken connection already exists", async () => {
    mocks.projectsError = true;
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("first-run-back"));

    const connect = screen.getByTestId("productive-setup-connect");
    expect(connect).toHaveTextContent(PRODUCTIVE_INTAKE_SETUP_COPY.wizardConnectAnother);
    expect(screen.getByTestId("productive-connection-integration-1")).toBeInTheDocument();
  });

  it("shows the create fallback when SuperPlane returns an internal error", async () => {
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "internal error" } } });
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("productive-project-project-1"));
    await user.click(screen.getByTestId("productive-setup-finish"));

    expect(await screen.findByRole("alert")).toHaveTextContent(PRODUCTIVE_INTAKE_SETUP_COPY.wizardCreateError);
  });
});
