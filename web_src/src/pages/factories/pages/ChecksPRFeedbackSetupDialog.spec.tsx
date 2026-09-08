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
    data: [],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({
    data: [
      { name: "circleci", label: "CircleCI" },
      { name: "semaphore", label: "Semaphore" },
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
  });

  it("waits for the catalog before it offers continue", () => {
    mocks.fetching = true;
    render(
      <ChecksPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    expect(screen.getByTestId("pr-feedback-check-names-loading")).toHaveTextContent("Loading status checks...");
    expect(screen.getByTestId("checks-setup-continue")).toBeDisabled();
  });

  it("preselects required checks and creates the handler after the tools step", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <ChecksPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/api"
        source={checksSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    expect(screen.getByTestId("checks-pr-feedback-setup")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("lint");
    });
    expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e");

    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByText("Suggested from the selected checks")).toBeInTheDocument();
    await user.click(screen.getByTestId("checks-setup-connect-circleci"));
    expect(screen.getByTestId("integration-create-dialog")).toBeInTheDocument();
  });

  it("preselects observed checks when none are required", async () => {
    mocks.catalog.splice(0, mocks.catalog.length, { name: "e2e", required: false, suggestedIntegration: "circleci" });
    render(
      <ChecksPRFeedbackSetupDialog
        open
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
        open
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
        open
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
});
