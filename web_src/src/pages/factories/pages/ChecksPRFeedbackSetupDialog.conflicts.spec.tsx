import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ChecksPRFeedbackSetupDialog } from "./ChecksPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";
import { readDeferredWorkspaceNextStep, WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY } from "./workspaceNextStepDeferral";

const mocks = vi.hoisted(() => ({
  createHandler: vi.fn(),
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: mocks.createHandler, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [],
    isPending: false,
    isFetching: false,
    isError: false,
  }),
  useConnectedIntegrations: () => ({
    data: [],
    isPending: false,
    isFetching: false,
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [] }),
  useCreateIntegration: () => ({ mutateAsync: vi.fn(), reset: vi.fn() }),
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: () => null,
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: () => <span />,
}));

const checksSource = PR_FEEDBACK_SOURCES.find((source) => source.id === "checks")!;

function renderDialog(onClose = vi.fn(), onCreated = vi.fn()) {
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
  return { onClose, onCreated };
}

describe("ChecksPRFeedbackSetupDialog conflict setup", () => {
  beforeEach(() => {
    mocks.createHandler.mockReset();
    mocks.createHandler.mockResolvedValue({ id: "handler-1" });
    window.localStorage.removeItem(WORKSPACE_NEXT_STEP_DEFERRAL_STORAGE_KEY);
  });

  it("creates a conflict-only handler when no status checks exist", async () => {
    const user = userEvent.setup();
    const { onClose, onCreated } = renderDialog();

    await waitFor(() => expect(screen.getByTestId("checks-setup-empty")).toBeInTheDocument());
    expect(screen.getByTestId("checks-setup-empty")).toHaveTextContent(
      "Run CI on a recent pull request, then reopen Configure to select checks.",
    );
    expect(screen.getByRole("option", { name: /Merge conflicts/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("checks-setup-merge-conflict-hours")).toHaveValue(24);
    expect(screen.getByTestId("checks-setup-maximum-attempts")).toHaveValue(3);
    expect(screen.queryByTestId("checks-setup-continue")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("checks-setup-finish"));

    await waitFor(() => {
      expect(mocks.createHandler).toHaveBeenCalledWith({
        source: "SOURCE_PULL_REQUEST_CHECKS",
        name: "Fix pull request checks",
        settings: {
          subject: { repository: "acme/api" },
          checks: {
            names: [],
            maximumAttempts: 3,
            fixMergeConflicts: true,
            mergeConflictHours: 24,
            runnerIntegrationIds: [],
          },
        },
      });
    });
    expect(onCreated).toHaveBeenCalledWith("handler-1");
    expect(onClose).toHaveBeenCalled();
    expect(readDeferredWorkspaceNextStep("factory-1")).toBeNull();
  });

  it("closes without a handler when merge conflicts are off and no checks exist", async () => {
    const user = userEvent.setup();
    const { onClose, onCreated } = renderDialog();

    await waitFor(() => expect(screen.getByTestId("checks-setup-empty")).toBeInTheDocument());
    await user.click(screen.getByRole("option", { name: /Merge conflicts/ }));
    expect(screen.queryByTestId("checks-setup-maximum-attempts")).not.toBeInTheDocument();
    await user.click(screen.getByTestId("checks-setup-finish"));

    expect(mocks.createHandler).not.toHaveBeenCalled();
    expect(onCreated).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
    expect(readDeferredWorkspaceNextStep("factory-1")).toBe("pr-checks-handler");
  });
});
