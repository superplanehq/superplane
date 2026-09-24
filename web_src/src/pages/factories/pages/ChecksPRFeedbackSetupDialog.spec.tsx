import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";
import { readDeferredWorkspaceNextStep, WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY } from "./workspaceNextStepDeferral";

const mocks = vi.hoisted(() => ({
  createHandler: vi.fn(),
  fetching: false,
  catalog: [
    { type: "status_check", id: "lint", name: "lint" },
    {
      type: "status_check",
      id: "e2e",
      name: "e2e",
      url: "https://app.circleci.com/pipelines/github/acme/api/1",
    },
  ],
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: mocks.createHandler, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: mocks.catalog,
    isPending: false,
    isFetching: mocks.fetching,
    isError: false,
  }),
  useConnectedIntegrations: () => ({
    data: mocks.connected,
    isPending: false,
    isFetching: false,
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({
    data: [
      { name: "circleci", label: "CircleCI" },
      { name: "semaphore", label: "Semaphore" },
      { name: "slack", label: "Slack" },
    ],
  }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="integration-create-dialog">Connect</div> : null,
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: () => <span />,
}));

Element.prototype.scrollIntoView ??= () => undefined;

const checksSource = PR_FEEDBACK_SOURCES.find((source) => source.id === "checks")!;

const defaultCatalog = [
  { type: "status_check", id: "lint", name: "lint" },
  {
    type: "status_check",
    id: "e2e",
    name: "e2e",
    url: "https://app.circleci.com/pipelines/github/acme/api/1",
  },
];

describe("ChecksPRFeedbackSetupDialog", () => {
  beforeEach(() => {
    mocks.createHandler.mockReset();
    mocks.createHandler.mockResolvedValue({ id: "handler-1" });
    mocks.fetching = false;
    mocks.catalog.splice(0, mocks.catalog.length, ...defaultCatalog);
    mocks.connected.splice(0);
    window.localStorage.removeItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY);
  });

  it("waits for the catalog before it offers continue", () => {
    mocks.fetching = true;
    mocks.catalog.splice(0);
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    const loading = screen.getByTestId("pr-feedback-check-names-loading");
    expect(loading).toHaveTextContent("Loading status checks");
    expect(loading).toHaveTextContent(
      "Reading recent pull requests and the required status checks for this repository",
    );
    expect(loading.querySelector("svg.animate-spin")).not.toBeNull();
    expect(screen.getByTestId("checks-setup-continue")).toBeDisabled();
    expect(screen.queryByTestId("checks-setup-maximum-attempts")).not.toBeInTheDocument();
  });

  it("preselects catalog checks and creates the handler after the tools step", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint");
      expect(screen.getByTestId("checks-setup-preview-check-lint")).toHaveAttribute("data-selected", "true");
    });
    expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e");
    expect(screen.getByTestId("checks-setup-preview-check-lint")).toHaveAttribute("data-outcome", "Fixing...");
    expect(screen.getByTestId("checks-setup-preview-check-e2e")).toHaveAttribute("data-selected", "true");
    expect(screen.getByTestId("checks-setup-preview-check-e2e")).toHaveAttribute("data-outcome", "Fixing...");
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "Selected checks start an automatic fix.",
    );
    expect(screen.getByTestId("checks-setup-preview-attempts")).toHaveTextContent("Stops after 3 attempts.");

    expect(screen.getByTestId("checks-setup-back")).toHaveTextContent("Back to board");
    expect(screen.getByRole("heading", { name: "Which status checks should be fixed?" })).toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-maximum-attempts")).toHaveValue(3);
    expect(screen.getByText("How many times should SuperPlane try before giving up?")).toBeInTheDocument();
    expect(
      screen.getByText(
        "SuperPlane pauses automatic fixes after this many consecutive attempts. Passing checks reset the count.",
      ),
    ).toBeInTheDocument();

    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.queryByTestId("checks-setup-maximum-attempts")).not.toBeInTheDocument();

    expect(screen.getByRole("heading", { name: "Which tools report these checks?" })).toBeInTheDocument();
    expect(screen.queryByText("Suggested")).not.toBeInTheDocument();
    expect(
      screen.getByText("Based on the status checks you selected, SuperPlane needs access to these tools."),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Grant that access so the agent can read CI logs and fix failing checks."),
    ).toBeInTheDocument();
    expect(screen.getByText("CircleCI")).toBeInTheDocument();
    expect(screen.queryByText("Semaphore")).not.toBeInTheDocument();
    expect(screen.queryByText("Slack")).not.toBeInTheDocument();
    expect(screen.queryByText("Other CI tools")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("checks-setup-connect-circleci"));
    expect(screen.getByTestId("integration-create-dialog")).toBeInTheDocument();
  });

  it("preselects observed checks from the catalog", async () => {
    mocks.catalog.splice(0, mocks.catalog.length, {
      type: "status_check",
      id: "e2e",
      name: "e2e",
      url: "https://app.circleci.com/pipelines/github/acme/api/1",
    });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e");
    });
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByTestId("pr-feedback-check-names-empty")).not.toBeInTheDocument();
  });

  it("lets the user deselect a catalog check", async () => {
    const user = userEvent.setup();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() =>
      expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "true"),
    );
    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));

    expect(screen.getByTestId("pr-feedback-check-option-lint")).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("pr-feedback-check-option-e2e")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("checks-setup-preview-check-lint")).toHaveAttribute("data-selected", "false");
    expect(screen.getByTestId("checks-setup-preview-check-lint")).toHaveAttribute("data-outcome", "Ignored");
    expect(screen.getByTestId("checks-setup-preview-check-e2e")).toHaveAttribute("data-selected", "true");
    expect(screen.getByTestId("checks-setup-preview-check-e2e")).toHaveAttribute("data-outcome", "Fixing...");
  });

  it("does not continue when the attempt limit is not a whole number", async () => {
    const user = userEvent.setup();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("checks-setup-maximum-attempts")).toHaveValue(3));
    await user.clear(screen.getByTestId("checks-setup-maximum-attempts"));
    await user.type(screen.getByTestId("checks-setup-maximum-attempts"), "5.5");

    expect(screen.getByTestId("checks-setup-continue")).toBeDisabled();
  });

  it("sends the maximum attempts from the first step", async () => {
    const user = userEvent.setup();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("checks-setup-maximum-attempts")).toHaveValue(3));
    expect(screen.getByTestId("checks-setup-preview-attempts")).toHaveTextContent("Stops after 3 attempts.");
    await user.clear(screen.getByTestId("checks-setup-maximum-attempts"));
    await user.type(screen.getByTestId("checks-setup-maximum-attempts"), "5");
    expect(screen.getByTestId("checks-setup-preview-attempts")).toHaveTextContent("Stops after 5 attempts.");
    await user.click(screen.getByTestId("checks-setup-continue"));
    await user.click(screen.getByTestId("checks-setup-finish"));

    await waitFor(() => {
      expect(mocks.createHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          settings: expect.objectContaining({
            checks: expect.objectContaining({ maximumAttempts: 5 }),
          }),
        }),
      );
    });
  });

  it("creates the handler with selected check names", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));
    expect(screen.getByTestId("checks-setup-finish")).toHaveTextContent("Finish");
    expect(
      screen.getByText("Based on the status checks you selected, SuperPlane needs access to these tools."),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Grant that access so the agent can read CI logs and fix failing checks."),
    ).toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "Some of the checks you selected require additional access that you are not granting.",
    );
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "That prevents the agent from having enough context to fix issues correctly.",
    );
    expect(screen.getByTestId("checks-setup-preview-tool-outcome")).toHaveTextContent("Not enough access");
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "SuperPlane does not have enough access to fix these checks.",
    );
    await user.click(screen.getByTestId("checks-setup-finish"));

    await waitFor(() => {
      expect(mocks.createHandler).toHaveBeenCalledWith({
        source: "SOURCE_PULL_REQUEST_CHECKS",
        name: "Fix pull request checks",
        settings: {
          subject: { repository: "acme/api" },
          checks: { names: ["lint", "e2e"], maximumAttempts: 3, runnerIntegrationIds: [] },
        },
      });
    });
    expect(onCreated).toHaveBeenCalledWith("handler-1");
  });

  it("says no additional access is required when no CI tool is suggested", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    mocks.catalog.splice(0, mocks.catalog.length, { type: "status_check", id: "lint", name: "lint" });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByTestId("checks-setup-tools-no-access")).toHaveTextContent(
      "The selected status checks do not need additional access.",
    );
    expect(screen.queryByTestId("checks-setup-preview-tool-outcome")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "These checks do not need additional access.",
    );
    expect(screen.queryByText("Suggested")).not.toBeInTheDocument();
    expect(screen.queryByText("CircleCI")).not.toBeInTheDocument();
    expect(screen.queryByText("Semaphore")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-tools-unknown")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-finish")).toBeEnabled();

    await user.click(screen.getByTestId("checks-setup-finish"));
    await waitFor(() => {
      expect(mocks.createHandler).toHaveBeenCalledWith({
        source: "SOURCE_PULL_REQUEST_CHECKS",
        name: "Fix pull request checks",
        settings: {
          subject: { repository: "acme/api" },
          checks: { names: ["lint"], maximumAttempts: 3, runnerIntegrationIds: [] },
        },
      });
    });
    expect(onCreated).toHaveBeenCalledWith("handler-1");
  });

  it("says GitHub Actions does not need additional access", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    mocks.catalog.splice(0, mocks.catalog.length, {
      type: "status_check",
      id: "build",
      name: "build",
      url: "https://github.com/acme/api/actions/runs/99",
    });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("build"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByTestId("checks-setup-tools-github-actions")).toHaveTextContent("GitHub Actions");
    expect(screen.getByTestId("checks-setup-tools-github-actions")).toHaveTextContent(
      "SuperPlane does not need additional access.",
    );
    expect(screen.queryByTestId("checks-setup-preview-tool-outcome")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "SuperPlane reads GitHub Actions logs.",
    );
    expect(screen.queryByText("CircleCI")).not.toBeInTheDocument();
    expect(screen.queryByText("Semaphore")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-tools-no-access")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-finish")).toBeEnabled();

    await user.click(screen.getByTestId("checks-setup-finish"));
    await waitFor(() => expect(mocks.createHandler).toHaveBeenCalled());
    expect(onCreated).toHaveBeenCalledWith("handler-1");
  });

  it("keeps suggested tools when GitHub Actions is mixed with another CI tool", async () => {
    const user = userEvent.setup();
    mocks.catalog.splice(
      0,
      mocks.catalog.length,
      {
        type: "status_check",
        id: "build",
        name: "build",
        url: "https://github.com/acme/api/actions/runs/99",
      },
      {
        type: "status_check",
        id: "e2e",
        name: "e2e",
        url: "https://app.circleci.com/pipelines/github/acme/api/1",
      },
    );
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByText("CircleCI")).toBeInTheDocument();
    expect(screen.queryByText("Semaphore")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-tools-github-actions")).toHaveTextContent(
      "GitHub Actions does not need additional access.",
    );
  });

  it("finishes without a handler and minimizes next steps when no checks exist", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onCreated = vi.fn();
    mocks.catalog.splice(0);
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("checks-setup-empty")).toBeInTheDocument());
    expect(screen.getByTestId("checks-setup-preview")).toHaveTextContent("No status checks yet.");
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "This repository has no status checks yet.",
    );
    expect(screen.queryByTestId("checks-setup-preview-attempts")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-empty")).toHaveTextContent(
      "This repository does not have status checks yet. That is OK.",
    );
    expect(screen.getByTestId("checks-setup-empty")).toHaveTextContent(
      "After you add them, you can configure SuperPlane to fix them automatically.",
    );
    expect(screen.queryByTestId("pr-feedback-check-names-picker")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-maximum-attempts")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-continue")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("checks-setup-finish"));

    expect(mocks.createHandler).not.toHaveBeenCalled();
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
    expect(readDeferredWorkspaceNextStep("factory-1")).toBe("pr-checks-handler");
  });

  it("preselects a connected integration and hides Connect", async () => {
    const user = userEvent.setup();
    mocks.connected.push({
      metadata: { id: "int-cci", name: "circleci-prod", integrationName: "circleci" },
      status: { state: "ready" },
    });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    const row = screen.getByTestId("checks-setup-integration-int-cci");
    await waitFor(() => expect(row).toHaveAttribute("aria-selected", "true"));
    expect(row).toHaveTextContent("circleci-prod");
    expect(screen.queryByTestId("checks-setup-connect-circleci")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-connect-semaphore")).not.toBeInTheDocument();
    expect(screen.queryByText("Semaphore")).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-tools-unselected")).not.toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-preview-tool-outcome")).toHaveTextContent("Reading CircleCI logs.");
    expect(screen.getByTestId("checks-setup-preview-caption")).toHaveTextContent(
      "SuperPlane reads CI logs from the tools you grant.",
    );

    await user.click(row);
    expect(row).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "Some of the checks you selected require additional access that you are not granting.",
    );
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "That prevents the agent from having enough context to fix issues correctly.",
    );
    expect(screen.getByTestId("checks-setup-preview-tool-outcome")).toHaveTextContent("Not enough access");
    expect(screen.getByTestId("checks-setup-finish")).toBeEnabled();

    await user.click(screen.getByTestId("checks-setup-finish"));
    await waitFor(() => {
      expect(mocks.createHandler).toHaveBeenCalledWith(
        expect.objectContaining({
          settings: expect.objectContaining({
            checks: expect.objectContaining({ runnerIntegrationIds: [] }),
          }),
        }),
      );
    });
  });
});
