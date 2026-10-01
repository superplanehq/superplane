import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { formatEnabledChecksValue } from "./mergeConfidenceChecks";
import { defaultRiskScoreCategories, formatRiskScoreRules } from "./riskScoreCategories";
import { RiskScoreSetupDialog } from "./RiskScoreSetupDialog";
import { RISK_SCORE_SETUP_COPY } from "./riskScoreSetupCopy";

const mocks = vi.hoisted(() => ({
  installFactory: vi.fn(),
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

vi.mock("@/pages/home/useInstallFactory", () => ({
  useInstallFactory: () => ({ installFactory: mocks.installFactory, isInstalling: false }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: mocks.showErrorToast,
  showSuccessToast: mocks.showSuccessToast,
}));

function renderDialog(
  overrides: { githubIntegrationId?: string; githubInstallationName?: string; onCreated?: () => void } = {},
) {
  const onCreated = overrides.onCreated ?? vi.fn();
  render(
    <RiskScoreSetupDialog
      organizationId="org-1"
      factoryId="factory-1"
      githubIntegrationId={overrides.githubIntegrationId ?? "github-1"}
      githubInstallationName={overrides.githubInstallationName ?? "github-superplanehq"}
      appRepository="acme/app"
      backlogRepository="acme/app"
      defaultBranch="main"
      onClose={vi.fn()}
      onCreated={onCreated}
    />,
  );
  return { onCreated };
}

describe("RiskScoreSetupDialog", () => {
  beforeEach(() => {
    mocks.installFactory.mockReset();
    mocks.showErrorToast.mockReset();
    mocks.showSuccessToast.mockReset();
  });

  it("lists the four checks and previews them on the task", async () => {
    const user = userEvent.setup();
    renderDialog();

    expect(screen.getByRole("heading", { name: RISK_SCORE_SETUP_COPY.title })).toBeInTheDocument();
    expect(screen.getByTestId("merge-confidence-setup-check-risk")).toHaveTextContent("Blast radius");
    expect(screen.getByTestId("merge-confidence-setup-check-performance")).toHaveTextContent("Performance");
    expect(screen.getByTestId("merge-confidence-setup-check-security")).toHaveTextContent("Security");
    expect(screen.getByTestId("merge-confidence-setup-check-drift")).toHaveTextContent("Drift from Specification");
    expect(screen.getByTestId("merge-confidence-setup-check-reversibility")).toHaveTextContent("Reversibility");
    expect(screen.queryByTestId("risk-score-setup-categories")).not.toBeInTheDocument();
    expect(screen.getByTestId("risk-score-setup-preview-card")).toHaveTextContent("Low risk");
    expect(screen.getByTestId("risk-score-setup-preview-card")).toHaveTextContent("On task");

    await user.click(screen.getByRole("switch", { name: "Performance" }));
    expect(screen.getByTestId("risk-score-setup-preview-card")).not.toHaveTextContent("Performance");
  });

  it("installs the risk score and returns to the board", async () => {
    mocks.installFactory.mockResolvedValue({ canvasId: "canvas-risk" });
    const user = userEvent.setup();
    const { onCreated } = renderDialog();

    await user.click(screen.getByTestId("risk-score-setup-finish"));

    await waitFor(() => expect(onCreated).toHaveBeenCalled());
    expect(mocks.installFactory).toHaveBeenCalledWith(
      expect.objectContaining({
        factoryId: "risk-score",
        workspaceFactoryId: "factory-1",
        integrations: { github: { id: "github-1", name: "github-superplanehq", ready: true } },
        installParams: {
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          riskRules: formatRiskScoreRules(defaultRiskScoreCategories()),
          enabledChecks: formatEnabledChecksValue(["risk", "performance", "security", "drift", "reversibility"]),
        },
      }),
    );
    expect(mocks.showSuccessToast).toHaveBeenCalledWith(RISK_SCORE_SETUP_COPY.created);
  });

  it("waits for the GitHub installation name before it installs", () => {
    renderDialog({ githubInstallationName: "" });

    expect(screen.getByTestId("risk-score-setup-finish")).toBeDisabled();
    expect(mocks.installFactory).not.toHaveBeenCalled();
  });

  it("asks for GitHub before it installs", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog({ githubIntegrationId: "", githubInstallationName: "" });

    await user.click(screen.getByTestId("risk-score-setup-finish"));

    expect(mocks.installFactory).not.toHaveBeenCalled();
    expect(mocks.showErrorToast).toHaveBeenCalledWith(RISK_SCORE_SETUP_COPY.missingGitHub);
    expect(onCreated).not.toHaveBeenCalled();
  });
});
