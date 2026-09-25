import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";
import { ProductiveIntakeSetupDialog } from "./ProductiveIntakeSetupDialog";
import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";

const PERMISSION_ERROR =
  "This API token cannot create webhooks. In Productive, go to Settings > API Integrations and create a token with read and write access.";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createIntegration: vi.fn(),
  deleteIntegration: vi.fn(),
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
  projectsError: false,
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsDeleteIntegration: mocks.deleteIntegration,
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  integrationKeys: {
    connected: (organizationId: string) => ["integrations", "connected", organizationId],
    integration: (organizationId: string, integrationId: string) => [
      "integrations",
      "connected",
      organizationId,
      integrationId,
    ],
  },
  useConnectedIntegrations: () => ({
    data: mocks.connected,
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [{ name: "productive", label: "Productive" }] }),
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, reset: vi.fn() }),
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
  IntegrationCreateDialog: ({
    open,
    onCreateIntegration,
    onCreated,
  }: {
    open: boolean;
    onCreateIntegration: (payload: { integrationName: string; name: string }) => Promise<
      | {
          integration?: { metadata?: { id?: string } };
        }
      | undefined
    >;
    onCreated: (id: string) => void;
  }) =>
    open ? (
      <button
        type="button"
        data-testid="finish-connect"
        onClick={() => {
          void onCreateIntegration({ integrationName: "productive", name: "Productive" }).then(
            (result) => {
              onCreated(result?.integration?.metadata?.id ?? "integration-new");
            },
            (cause: unknown) => {
              const alert = document.createElement("p");
              alert.setAttribute("role", "alert");
              alert.setAttribute("data-testid", "connect-error");
              alert.textContent = cause instanceof Error ? cause.message : "";
              document.body.appendChild(alert);
            },
          );
        }}
      >
        Finish connect
      </button>
    ) : null,
}));

function renderDialog(onCreated = vi.fn()) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/productive"]}>
        <ProductiveIntakeSetupDialog
          organizationId="org-1"
          factoryId="factory-1"
          integrationsBasePath="/org-1/workspaces/sp/settings/organization/integrations"
          onClose={vi.fn()}
          onCreated={onCreated}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ProductiveIntakeSetupDialog", () => {
  beforeEach(() => {
    document.querySelector("[data-testid='connect-error']")?.remove();
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.createIntegration.mockReset();
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-new", name: "Productive", integrationName: "productive" },
          status: { state: "ready" },
        },
      },
    });
    mocks.deleteIntegration.mockReset();
    mocks.deleteIntegration.mockResolvedValue({});
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
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent("10 newest");
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
    expect(screen.getByTestId("productive-skip-initial-import")).not.toBeChecked();
    await user.click(screen.getByTestId("productive-skip-initial-import"));
    expect(screen.getByText(PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProjectHelperSkip)).toBeInTheDocument();
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

  it("stays on the connection step when the token cannot create webhooks", async () => {
    mocks.connected.splice(0, mocks.connected.length);
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-bad", name: "Productive", integrationName: "productive" },
          status: { state: "error", stateDescription: PERMISSION_ERROR },
        },
      },
    });
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("productive-setup-connect"));
    await user.click(screen.getByTestId("finish-connect"));

    expect(await screen.findByTestId("connect-error")).toHaveTextContent(PERMISSION_ERROR);
    expect(mocks.deleteIntegration).toHaveBeenCalledWith(
      expect.objectContaining({
        path: { id: "org-1", integrationId: "integration-bad" },
      }),
    );
    expect(screen.getByRole("heading", { name: PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepConnect })).toBeInTheDocument();
    expect(
      screen.queryByRole("heading", { name: PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepProject }),
    ).not.toBeInTheDocument();
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
