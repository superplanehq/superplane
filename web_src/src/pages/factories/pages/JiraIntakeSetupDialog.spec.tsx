import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import type { OrganizationsIntegration } from "@/api-client";

import { JiraIntakeSetupDialog } from "./JiraIntakeSetupDialog";

const SETUP_RETURN_TO = "/org-1/workspaces/sp/lines/line-plan?jiraIntake=1&pick=newest";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  connected: [] as OrganizationsIntegration[],
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
  useAvailableIntegrations: () => ({ data: [{ name: "jira", label: "Jira" }] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
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

function jiraConnection(id: string, name: string, createdAt: string, state = "ready"): OrganizationsIntegration {
  return {
    metadata: { id, name, integrationName: "jira", createdAt },
    status: { state },
  };
}

function renderDialog(options?: { selectNewest?: boolean }) {
  return render(
    <JiraIntakeSetupDialog
      open
      organizationId="org-1"
      factoryId="factory-1"
      onClose={vi.fn()}
      setupReturnTo={SETUP_RETURN_TO}
      selectNewest={options?.selectNewest}
    />,
  );
}

describe("JiraIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.connected.splice(
      0,
      mocks.connected.length,
      jiraConnection("integration-1", "Acme Jira", "2026-01-01T00:00:00Z"),
    );
  });

  it("creates the intake once a connection and a project are chosen", async () => {
    const user = userEvent.setup();
    renderDialog();

    // A single ready connection is preselected, so Continue opens the
    // project step.
    await waitFor(() => expect(screen.getByRole("button", { name: "Continue" })).toBeEnabled());
    await user.click(screen.getByRole("button", { name: "Acme Jira" }));
    await user.click(screen.getByRole("button", { name: "Continue" }));

    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByRole("button", { name: "Create intake" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-1",
        resourceId: "ENG",
      });
    });

    expect(screen.getByText("Jira intake is ready")).toBeInTheDocument();
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.type(screen.getByLabelText("Search projects"), "oper");

    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
    expect(screen.getByTestId("jira-project-OPS")).toBeInTheDocument();
  });

  it("returns to the connection step from the project step", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(screen.getByRole("button", { name: "Back to connection" }));

    expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
  });

  it("opens project selection as soon as a new site is connected", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: "Connect another site" }));
    expect(screen.getByTestId("setup-return-to")).toHaveTextContent(SETUP_RETURN_TO);
    await user.click(screen.getByTestId("finish-connect"));

    expect(screen.getByTestId("jira-project-ENG")).toBeInTheDocument();
  });

  it("selects the newest ready connection after OAuth and opens the project step", async () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      jiraConnection("integration-1", "Acme Jira", "2026-01-01T00:00:00Z"),
      jiraConnection("integration-new", "New Jira", "2026-09-16T00:00:00Z"),
    );
    renderDialog({ selectNewest: true });

    expect(await screen.findByTestId("jira-project-ENG")).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByTestId("jira-project-ENG"));
    await user.click(screen.getByRole("button", { name: "Create intake" }));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_JIRA_ISSUES",
        integrationId: "integration-new",
        resourceId: "ENG",
      });
    });
  });

  it("waits for the newest connection to become ready before opening projects", () => {
    mocks.connected.splice(
      0,
      mocks.connected.length,
      jiraConnection("integration-1", "Acme Jira", "2026-01-01T00:00:00Z"),
      jiraConnection("integration-new", "New Jira", "2026-09-16T00:00:00Z", "pending"),
    );
    renderDialog({ selectNewest: true });

    expect(screen.getByRole("button", { name: "Continue" })).toBeInTheDocument();
    expect(screen.queryByTestId("jira-project-ENG")).not.toBeInTheDocument();
  });
});
