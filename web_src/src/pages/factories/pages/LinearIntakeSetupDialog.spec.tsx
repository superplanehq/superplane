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
  resourceCalls: [] as Array<{ type: string; parameters?: Record<string, string> }>,
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
  useIntegrationResources: (
    _organizationId: string,
    _integrationId: string,
    type: string,
    parameters?: Record<string, string>,
  ) => {
    mocks.resourceCalls.push({ type, parameters });
    const resources: Record<string, Array<{ id: string; name: string }>> = {
      team: [
        { id: "team-1", name: "Platform (PLT)" },
        { id: "team-2", name: "Growth (GRO)" },
      ],
      project: [
        { id: "project-1", name: "Checkout" },
        { id: "project-2", name: "Billing" },
      ],
      label: [{ id: "label-1", name: "bug" }],
    };
    return { data: resources[type] ?? [], isLoading: false, isError: false, refetch: vi.fn() };
  },
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
    mocks.resourceCalls.length = 0;
    mocks.linearDefinition.hostedAppInstall = false;
    mocks.connected.splice(0, mocks.connected.length, {
      metadata: { id: "integration-1", name: "Acme Linear", integrationName: "linear" },
      status: { state: "ready" },
    });
  });

  it("opens the team step when a ready Linear connection already exists", async () => {
    renderDialog();

    expect(await screen.findByRole("heading", { name: LINEAR_INTAKE_SETUP_COPY.wizardStepTeam })).toBeInTheDocument();
    expect(screen.getByTestId("linear-skip-initial-import")).not.toBeChecked();
    expect(screen.getByText(LINEAR_INTAKE_SETUP_COPY.importExistingHelper)).toBeInTheDocument();
    expect(screen.getByTestId("linear-setup-finish")).toBeDisabled();
    expect(screen.queryByText(LINEAR_INTAKE_SETUP_COPY.projectsLabel)).not.toBeInTheDocument();
  });

  it("creates the intake for a team with no projects and every label", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await user.click(await screen.findByTestId("linear-team-team-1"));
    expect(screen.getByText(LINEAR_INTAKE_SETUP_COPY.projectsHelper)).toBeInTheDocument();
    expect(screen.queryByTestId("linear-include-no-project")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_LINEAR_ISSUES",
        integrationId: "integration-1",
        settings: {
          linearTeamId: "team-1",
          linearProjectIds: [],
          linearIncludeIssuesWithoutProject: true,
          linearLabels: [],
        },
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates the intake with the projects, labels, and no-project choice of the team", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("linear-team-team-1"));
    await user.click(screen.getByTestId("linear-project-project-1"));
    await user.click(screen.getByTestId("linear-project-project-2"));
    expect(screen.getByTestId("linear-include-no-project")).toBeChecked();
    await user.click(screen.getByTestId("linear-include-no-project"));
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_LINEAR_ISSUES",
        integrationId: "integration-1",
        settings: {
          linearTeamId: "team-1",
          linearProjectIds: ["project-1", "project-2"],
          linearIncludeIssuesWithoutProject: false,
          linearLabels: ["bug"],
        },
      });
    });
  });

  it("loads the projects and labels of the selected team", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("linear-team-team-2"));

    expect(mocks.resourceCalls).toContainEqual({ type: "project", parameters: { team: "team-2" } });
    expect(mocks.resourceCalls).toContainEqual({ type: "label", parameters: { team: "team-2" } });
  });

  it("clears the projects and labels when the user picks another team", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(await screen.findByTestId("linear-team-team-1"));
    await user.click(screen.getByTestId("linear-project-project-1"));
    await user.click(screen.getByRole("checkbox", { name: "bug" }));
    await user.click(screen.getByTestId("linear-team-team-2"));
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith(
        expect.objectContaining({
          settings: expect.objectContaining({ linearTeamId: "team-2", linearProjectIds: [], linearLabels: [] }),
        }),
      );
    });
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

    await user.click(await screen.findByTestId("linear-team-team-1"));
    await user.click(screen.getByTestId("linear-skip-initial-import"));
    await user.click(screen.getByTestId("linear-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith(
        expect.objectContaining({
          skipInitialImport: true,
          settings: expect.objectContaining({ linearTeamId: "team-1" }),
        }),
      );
    });
  });
});
