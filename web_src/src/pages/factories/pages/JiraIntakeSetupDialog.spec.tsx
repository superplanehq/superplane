import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";
import { JiraIntakeSetupDialog } from "./JiraIntakeSetupDialog";
import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createIntegration: vi.fn(),
  followBrowserAction: vi.fn(() => true),
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
  jiraDefinition: { name: "jira", label: "Jira", hostedAppInstall: false } as {
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
  useAvailableIntegrations: () => ({ data: [mocks.jiraDefinition], isLoading: false }),
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, reset: vi.fn() }),
  useIntegrationResources: () => ({
    data: [
      { id: "ENG", name: "Engineering (ENG)" },
      { id: "OPS", name: "Operations (OPS)" },
    ],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: mocks.followBrowserAction,
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({
    open,
    onCreated,
    setupReturnTo,
  }: {
    open: boolean;
    onCreated: (id: string) => void;
    setupReturnTo?: string;
  }) =>
    open ? (
      <div>
        <span data-testid="setup-return-to">{setupReturnTo ?? ""}</span>
        <button type="button" data-testid="finish-connect" onClick={() => onCreated("integration-new")}>
          Finish connect
        </button>
      </div>
    ) : null,
}));

function renderDialog(onCreated = vi.fn(), selectIntegrationId = "") {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/jira"]}>
      <JiraIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        onClose={vi.fn()}
        onCreated={onCreated}
        selectIntegrationId={selectIntegrationId}
      />
    </MemoryRouter>,
  );
}

describe("JiraIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.createIntegration.mockReset();
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-pending" },
          status: { browserAction: { url: "https://auth.atlassian.com/authorize" } },
        },
      },
    });
    mocks.followBrowserAction.mockClear();
    mocks.jiraDefinition.hostedAppInstall = false;
    mocks.connected.splice(0, mocks.connected.length, {
      metadata: { id: "integration-1", name: "Acme Jira", integrationName: "jira" },
      status: { state: "ready" },
    });
  });

  it("opens the project step when a ready Jira connection already exists", async () => {
    renderDialog();

    expect(await screen.findByRole("heading", { name: JIRA_INTAKE_SETUP_COPY.wizardStepProject })).toBeInTheDocument();
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toBeInTheDocument();
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent("10 newest");
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent(
      "listens for new issues",
    );
    expect(screen.getByTestId("jira-intake-setup-stepper")).toBeInTheDocument();
    expect(screen.getByTestId("jira-intake-setup-sphere")).toBeInTheDocument();
    expect(screen.queryByTestId("jira-setup-preview")).not.toBeInTheDocument();
  });

  it("creates a bound intake after a project is chosen", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await screen.findByTestId("jira-project-ENG");
    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByTestId("jira-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-1",
        resourceId: "ENG",
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates a bound intake without importing existing issues", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await screen.findByTestId("jira-project-ENG");
    expect(screen.getByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label)).toBeInTheDocument();
    expect(screen.getByTestId("jira-skip-initial-import")).not.toBeChecked();
    await user.click(screen.getByTestId("jira-skip-initial-import"));
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelperSkip)).toBeInTheDocument();
    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByTestId("jira-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-1",
        resourceId: "ENG",
        skipInitialImport: true,
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    renderDialog();

    await screen.findByLabelText("Search projects");
    await user.type(screen.getByLabelText("Search projects"), "oper");

    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
    expect(screen.getByTestId("jira-project-OPS")).toBeInTheDocument();
  });

  it("returns to the connection step from the project step", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("first-run-back"));

    expect(screen.getByRole("heading", { name: JIRA_INTAKE_SETUP_COPY.wizardStepConnect })).toBeInTheDocument();
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepConnectHelper)).toBeInTheDocument();
    expect(screen.getByText(JIRA_INTAKE_SETUP_COPY.wizardStepConnectHelper)).toHaveTextContent("authorize it again");
    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
  });

  it("binds the intake to the connection chosen from the list", async () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      { metadata: { id: "integration-1", name: "Acme Jira", integrationName: "jira" }, status: { state: "ready" } },
      { metadata: { id: "integration-2", name: "Acme EU Jira", integrationName: "jira" }, status: { state: "ready" } },
    );
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("first-run-back"));
    await user.click(await screen.findByTestId("jira-connection-integration-2"));
    await user.click(screen.getByTestId("jira-setup-continue"));

    await user.click(await screen.findByTestId("jira-project-ENG"));
    await user.click(screen.getByTestId("jira-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-2",
        resourceId: "ENG",
      });
    });
  });

  it("places Connect Jira on the right of the connection step", async () => {
    mocks.connected.splice(0, mocks.connected.length);
    renderDialog();

    const button = await screen.findByTestId("jira-setup-connect");
    expect(button.parentElement).toHaveClass("ml-auto");
  });

  it("opens project selection as soon as a new site is connected", async () => {
    mocks.connected.splice(0, mocks.connected.length);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("jira-setup-connect"));
    expect(screen.getByTestId("setup-return-to")).toHaveTextContent("/org-1/workspaces/sp/lines/line-plan/setup/jira");
    await user.click(await screen.findByTestId("finish-connect"));

    expect(screen.getByRole("heading", { name: JIRA_INTAKE_SETUP_COPY.wizardStepProject })).toBeInTheDocument();
    expect(screen.getByTestId("jira-project-ENG")).toBeInTheDocument();
  });

  it("opens Atlassian authorization without a second dialog when Jira OAuth is hosted", async () => {
    mocks.jiraDefinition.hostedAppInstall = true;
    mocks.connected.splice(0, mocks.connected.length);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("jira-setup-connect"));

    await waitFor(() => {
      expect(mocks.createIntegration).toHaveBeenCalledWith({
        integrationName: "jira",
        name: "jira",
        configuration: { setupReturnPath: "/org-1/workspaces/sp/lines/line-plan/setup/jira" },
      });
    });
    expect(mocks.followBrowserAction).toHaveBeenCalledWith({ url: "https://auth.atlassian.com/authorize" });
    expect(screen.queryByTestId("finish-connect")).not.toBeInTheDocument();
    expect(screen.queryByTestId("setup-return-to")).not.toBeInTheDocument();
  });

  it("selects the returned connection after OAuth and opens the project step", async () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      { metadata: { id: "integration-1", name: "Acme Jira", integrationName: "jira" }, status: { state: "ready" } },
      { metadata: { id: "integration-new", name: "New Jira", integrationName: "jira" }, status: { state: "ready" } },
    );
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated, "integration-new");

    expect(await screen.findByTestId("jira-project-ENG")).toBeInTheDocument();
    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByTestId("jira-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-new",
        resourceId: "ENG",
      });
    });
  });

  it("does not select an older ready connection while the returned one is pending", async () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      { metadata: { id: "integration-1", name: "Acme Jira", integrationName: "jira" }, status: { state: "ready" } },
      { metadata: { id: "integration-new", name: "New Jira", integrationName: "jira" }, status: { state: "pending" } },
    );
    renderDialog(vi.fn(), "integration-new");

    expect(await screen.findByTestId("jira-connection-integration-1")).toBeInTheDocument();
    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
  });

  it("shows the create fallback when SuperPlane returns internal error", async () => {
    mocks.createIntake.mockRejectedValue({ response: { data: { message: "internal error" } } });
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("jira-project-ENG"));
    await user.click(screen.getByTestId("jira-setup-finish"));

    expect(await screen.findByRole("alert")).toHaveTextContent(JIRA_INTAKE_SETUP_COPY.wizardCreateError);
  });
});
