import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

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

function renderDialog(overrides: { githubIntegrationId?: string; onCreated?: () => void } = {}) {
  const onCreated = overrides.onCreated ?? vi.fn();
  render(
    <RiskScoreSetupDialog
      organizationId="org-1"
      factoryId="factory-1"
      githubIntegrationId={overrides.githubIntegrationId ?? "github-1"}
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

  it("explains the 1 to 5 scale and previews the task check", () => {
    renderDialog();

    expect(screen.getByRole("heading", { name: RISK_SCORE_SETUP_COPY.title })).toBeInTheDocument();
    expect(screen.getByTestId("risk-score-setup-scale-1")).toHaveTextContent("Minimal Risk");
    expect(screen.getByTestId("risk-score-setup-scale-2")).toHaveTextContent("Low Risk");
    expect(screen.getByTestId("risk-score-setup-scale-3")).toHaveTextContent("Moderate Risk");
    expect(screen.getByTestId("risk-score-setup-scale-4")).toHaveTextContent("High Risk");
    expect(screen.getByTestId("risk-score-setup-scale-5")).toHaveTextContent("Severe Risk");
    expect(screen.queryByTestId("risk-score-setup-categories")).not.toBeInTheDocument();
    expect(screen.getByTestId("risk-score-setup-preview-card")).toHaveTextContent("Low Risk");
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
        installParams: {
          appRepository: "acme/app",
          backlogRepository: "acme/app",
          defaultBranch: "main",
          riskRules: formatRiskScoreRules(defaultRiskScoreCategories()),
        },
      }),
    );
    expect(mocks.showSuccessToast).toHaveBeenCalledWith(RISK_SCORE_SETUP_COPY.created);
  });

  it("asks for GitHub before it installs", async () => {
    const user = userEvent.setup();
    const { onCreated } = renderDialog({ githubIntegrationId: "" });

    await user.click(screen.getByTestId("risk-score-setup-finish"));

    expect(mocks.installFactory).not.toHaveBeenCalled();
    expect(mocks.showErrorToast).toHaveBeenCalledWith(RISK_SCORE_SETUP_COPY.missingGitHub);
    expect(onCreated).not.toHaveBeenCalled();
  });
});
