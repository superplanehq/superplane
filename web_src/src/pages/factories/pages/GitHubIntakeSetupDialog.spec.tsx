import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { INTAKE_SKIP_INITIAL_IMPORT_COPY } from "./intakeSkipInitialImportCopy";
import { GitHubIntakeSetupDialog } from "./GitHubIntakeSetupDialog";
import { GITHUB_INTAKE_SETUP_COPY } from "./githubIntakeSetupCopy";

const mocks = vi.hoisted(() => ({
  subscription: false,
  createIntake: vi.fn(),
}));

vi.mock("../layout/factoriesLayoutContext", () => ({
  useOptionalFactoriesLayout: () => ({ factory: { onboarding: { codexSubscription: mocks.subscription } } }),
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useFactoryOwnerEligibility", () => ({
  useFactoryOwnerEligibility: () => ({
    data: { members: [{ userId: "alice", eligible: true }] },
    isPending: false,
    isError: false,
  }),
}));
vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

function renderDialog(onCreated = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/github"]}>
      <GitHubIntakeSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        backlogRepository="acme/payments-service"
        onClose={vi.fn()}
        onCreated={onCreated}
      />
    </MemoryRouter>,
  );
}

describe("GitHubIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.subscription = false;
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1", canvasId: "canvas-1" });
  });

  it("requires and submits a connected default owner in subscription mode", async () => {
    mocks.subscription = true;
    const user = userEvent.setup();
    renderDialog();
    await user.click(screen.getByTestId("github-setup-finish"));
    expect(mocks.createIntake).not.toHaveBeenCalled();
    expect(screen.getByRole("alert")).toHaveTextContent("Select a default task owner");
    await user.click(screen.getByRole("combobox", { name: /Default task owner/ }));
    await user.click(await screen.findByRole("option", { name: "Alice" }));
    await user.click(screen.getByTestId("github-setup-finish"));
    await waitFor(() =>
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_GITHUB_ISSUES",
        settings: { defaultOwnerId: "alice" },
      }),
    );
  });
  it("shows the repository and import checkbox checked by default", () => {
    renderDialog();

    expect(screen.getByRole("heading", { name: GITHUB_INTAKE_SETUP_COPY.headline })).toBeInTheDocument();
    expect(screen.getByText(GITHUB_INTAKE_SETUP_COPY.helper)).toBeInTheDocument();
    expect(screen.getByTestId("github-setup-repository")).toHaveTextContent("acme/payments-service");
    expect(screen.getByText(GITHUB_INTAKE_SETUP_COPY.importExistingHelper)).toHaveTextContent("up to 3");
    expect(screen.getByTestId("github-skip-initial-import")).not.toBeChecked();
  });

  it("creates a GitHub intake with the default import", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await user.click(screen.getByTestId("github-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates a GitHub intake when no backlog repository is set", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/github"]}>
        <GitHubIntakeSetupDialog
          organizationId="org-1"
          factoryId="factory-1"
          backlogRepository=""
          onClose={vi.fn()}
          onCreated={onCreated}
        />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("github-setup-repository-missing")).toHaveTextContent(
      GITHUB_INTAKE_SETUP_COPY.repositoryMissing,
    );
    await user.click(screen.getByTestId("github-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({ source: "SOURCE_GITHUB_ISSUES" });
    });
    expect(onCreated).toHaveBeenCalled();
  });

  it("creates a GitHub intake without importing existing issues", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    expect(screen.getByText(INTAKE_SKIP_INITIAL_IMPORT_COPY.label)).toBeInTheDocument();
    await user.click(screen.getByTestId("github-skip-initial-import"));
    expect(screen.getByText(GITHUB_INTAKE_SETUP_COPY.importExistingHelperOff)).toBeInTheDocument();
    await user.click(screen.getByTestId("github-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_GITHUB_ISSUES",
        skipInitialImport: true,
      });
    });
    expect(onCreated).toHaveBeenCalled();
  });
});
