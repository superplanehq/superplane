import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";

const mocks = vi.hoisted(() => ({
  createHandler: vi.fn(),
  fetching: false,
  catalog: [
    { name: "lint", required: true },
    { name: "e2e", required: false, suggestedIntegration: "circleci" },
  ],
  connected: [] as Array<{
    metadata: { id: string; name: string; integrationName: string };
    status: { state: string };
  }>,
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryRepositoryStatusChecks: () => ({
    data: mocks.catalog,
    isPending: false,
    isFetching: mocks.fetching,
    isError: false,
  }),
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: mocks.createHandler, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
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
  { name: "lint", required: true },
  { name: "e2e", required: false, suggestedIntegration: "circleci" },
];

describe("ChecksPRFeedbackSetupDialog", () => {
  beforeEach(() => {
    mocks.createHandler.mockReset();
    mocks.createHandler.mockResolvedValue({ id: "handler-1" });
    mocks.fetching = false;
    mocks.catalog.splice(0, mocks.catalog.length, ...defaultCatalog);
    mocks.connected.splice(0);
  });

  it("waits for the catalog before it offers continue", () => {
    mocks.fetching = true;
    mocks.catalog.splice(0);
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
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
  });

  it("preselects required checks and creates the handler after the tools step", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => {
      expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint");
    });
    expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e");

    expect(screen.getByTestId("checks-setup-back")).toHaveTextContent("Back to board");
    expect(screen.getByRole("heading", { name: "Which status checks should be fixed?" })).toBeInTheDocument();

    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByRole("heading", { name: "Which tools report these checks?" })).toBeInTheDocument();
    expect(screen.getByText("Suggested from the selected checks")).toBeInTheDocument();
    expect(screen.getByText("CircleCI")).toBeInTheDocument();
    expect(screen.getByText("Semaphore")).toBeInTheDocument();
    expect(screen.queryByText("Slack")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("checks-setup-connect-circleci"));
    expect(screen.getByTestId("integration-create-dialog")).toBeInTheDocument();
  });

  it("preselects observed checks when none are required", async () => {
    mocks.catalog.splice(0, mocks.catalog.length, { name: "e2e", required: false, suggestedIntegration: "circleci" });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
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
  });

  it("creates the handler with selected check names", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));
    expect(screen.getByTestId("checks-setup-finish")).toHaveTextContent("Finish");
    expect(screen.getByText(/Connect the external tools reporting the status checks/)).toBeInTheDocument();
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "Some of the checks you selected require additional access that you are not granting.",
    );
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "That prevents the agent from having enough context to fix issues correctly.",
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

  it("warns when no CI tool is suggested and still lets the user finish", async () => {
    const user = userEvent.setup();
    mocks.catalog.splice(0, mocks.catalog.length, { name: "lint", required: true });
    render(
      <ChecksPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByTestId("checks-setup-tools-unknown")).toHaveTextContent(
      "The agent may not have enough context to fix them",
    );
    expect(screen.getByTestId("checks-setup-finish")).toBeEnabled();
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
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
    expect(screen.getByTestId("checks-setup-connect-semaphore")).toHaveTextContent("Connect");
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(screen.queryByTestId("checks-setup-tools-unselected")).not.toBeInTheDocument();

    await user.click(row);
    expect(row).toHaveAttribute("aria-selected", "false");
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "Some of the checks you selected require additional access that you are not granting.",
    );
    expect(screen.getByTestId("checks-setup-tools-unselected")).toHaveTextContent(
      "That prevents the agent from having enough context to fix issues correctly.",
    );
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
