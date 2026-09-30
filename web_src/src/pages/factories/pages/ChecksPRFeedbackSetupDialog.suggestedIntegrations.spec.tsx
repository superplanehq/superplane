import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";

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

function renderDialog() {
  return render(
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
}

describe("ChecksPRFeedbackSetupDialog suggested integrations", () => {
  beforeEach(() => {
    mocks.createHandler.mockReset();
    mocks.createHandler.mockResolvedValue({ id: "handler-1" });
    mocks.fetching = false;
    mocks.catalog.splice(0, mocks.catalog.length, ...defaultCatalog);
    mocks.connected.splice(0);
  });

  it("auto-selects a suggested integration that becomes ready after checks are selected", async () => {
    const user = userEvent.setup();
    renderDialog();

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e"));

    // No CI tool is ready yet, so the "select all ready tools" seed on mount cannot have
    // picked up "int-cci". It only becomes ready after checks are already selected, which
    // isolates the suggestion effect this test exercises.
    mocks.connected.push({
      metadata: { id: "int-cci", name: "circleci-prod", integrationName: "circleci" },
      status: { state: "ready" },
    });

    // Toggling a check re-runs the suggestion effect against the now-ready tool.
    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));
    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));

    await user.click(screen.getByTestId("checks-setup-continue"));

    const row = screen.getByTestId("checks-setup-integration-int-cci");
    await waitFor(() => expect(row).toHaveAttribute("aria-selected", "true"));
    expect(row).toHaveTextContent("circleci-prod");
    expect(screen.getByTestId("checks-setup-preview-tool-outcome")).toHaveTextContent("Reading CircleCI logs.");
  });

  it("keeps a suggested integration deselected after the user removes it", async () => {
    const user = userEvent.setup();
    mocks.connected.push({
      metadata: { id: "int-cci", name: "circleci-prod", integrationName: "circleci" },
      status: { state: "ready" },
    });
    renderDialog();

    await waitFor(() => expect(screen.getByTestId("pr-feedback-check-names-list")).toHaveTextContent("e2e"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    const row = screen.getByTestId("checks-setup-integration-int-cci");
    await waitFor(() => expect(row).toHaveAttribute("aria-selected", "true"));
    await user.click(row);
    expect(row).toHaveAttribute("aria-selected", "false");

    // Changing the selected checks must not re-add the tool the user just deselected.
    await user.click(screen.getByTestId("checks-setup-back"));
    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));
    await user.click(screen.getByTestId("pr-feedback-check-option-lint"));
    await user.click(screen.getByTestId("checks-setup-continue"));

    expect(screen.getByTestId("checks-setup-integration-int-cci")).toHaveAttribute("aria-selected", "false");
  });
});
