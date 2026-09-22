import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { DEFAULT_PLANNING_SETTINGS } from "./planningSettingsModel";
import { PlanningSetupDialog } from "./PlanningSetupDialog";

const mocks = vi.hoisted(() => ({
  updateFactory: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactory: () => ({ mutateAsync: mocks.updateFactory, isPending: false }),
}));

/** A new workspace: Planning and Confidence on, Clarity off, setup not confirmed. */
const NEW_FACTORY_PLANNING = { ...DEFAULT_PLANNING_SETTINGS, setupCompleted: false };

function renderDialog(onFinished = vi.fn(), onClose = vi.fn()) {
  render(
    <PlanningSetupDialog
      organizationId="org-1"
      factoryId="factory-1"
      factory={{ id: "factory-1", planning: { ...NEW_FACTORY_PLANNING } }}
      onClose={onClose}
      onFinished={onFinished}
    />,
  );
  return { onFinished, onClose };
}

describe("PlanningSetupDialog", () => {
  beforeEach(() => {
    mocks.updateFactory.mockReset();
    mocks.updateFactory.mockResolvedValue({
      id: "factory-1",
      planning: { ...NEW_FACTORY_PLANNING, setupCompleted: true },
    });
  });

  it("shows Source and Start, then Finish writes setupCompleted when Planning is off", async () => {
    const user = userEvent.setup();
    const { onFinished } = renderDialog();

    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption,
    );
    expect(screen.getByTestId("planning-setup-preview-title")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewDraftTitle,
    );
    expect(screen.getByTestId("planning-setup-preview-plan")).toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-question")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewAgentQuestion,
    );
    expect(screen.queryByTestId("planning-setup-preview-scores")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-setup-preview-start")).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: /Keep the source description/ }));
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption,
    );
    expect(screen.getByTestId("planning-setup-preview-issue")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewIssue,
    );
    expect(screen.getByTestId("planning-setup-preview-note")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewReady,
    );
    expect(screen.getByTestId("planning-setup-preview-start")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewStart,
    );
    expect(screen.queryByTestId("planning-setup-continue")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-setup-preview-question")).not.toBeInTheDocument();
    expect(screen.queryByTestId("planning-setup-preview-scores")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("planning-setup-finish"));

    expect(mocks.updateFactory).toHaveBeenCalledWith({
      planning: { enabled: false, clarity: false, confidence: true, setupCompleted: true },
    });
    expect(onFinished).toHaveBeenCalledOnce();
  });

  it("walks Refine to Confidence to Clarity and Finish writes all flags", async () => {
    const user = userEvent.setup();
    const { onFinished } = renderDialog();

    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepRefine })).toBeInTheDocument();
    await user.click(screen.getByTestId("planning-setup-continue"));

    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepConfidence })).toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOn,
    );
    expect(screen.getByTestId("planning-setup-preview-confidence")).toHaveAttribute("data-emphasized", "true");
    expect(screen.getByTestId("planning-setup-preview-confidence")).toHaveTextContent("Confidence 2/5");
    expect(screen.getByTestId("planning-setup-preview-confidence")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceSummary,
    );
    expect(screen.getByTestId("planning-setup-preview-note")).toHaveAttribute("data-tone", "caution");
    expect(screen.getByTestId("planning-setup-preview-note")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewCautionHeadline,
    );
    expect(screen.queryByTestId("planning-setup-preview-clarity")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveAttribute("data-chat", "confidence");
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceMessage,
    );
    expect(screen.getByTestId("planning-setup-preview-question")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceQuestion,
    );

    await user.click(screen.getByRole("radio", { name: /Skip the confidence estimate/ }));
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOff,
    );
    expect(screen.queryByTestId("planning-setup-preview-confidence")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveAttribute("data-chat", "question");
    expect(screen.getByTestId("planning-setup-preview-note")).toHaveAttribute("data-tone", "ready");

    await user.click(screen.getByRole("radio", { name: /Estimate confidence/ }));

    await user.click(screen.getByTestId("planning-setup-continue"));
    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepClarity })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /Skip the clarity check/ })).toBeChecked();
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityOff,
    );
    expect(screen.queryByTestId("planning-setup-preview-clarity")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveAttribute("data-chat", "question");
    expect(screen.getByTestId("planning-setup-preview-question")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewAgentQuestion,
    );

    await user.click(screen.getByRole("radio", { name: /Check clarity/ }));
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityOn,
    );
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveAttribute("data-chat", "clarity");
    expect(screen.getByTestId("planning-setup-preview-chat")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityMessage,
    );
    expect(screen.getByTestId("planning-setup-preview-question")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewClarityQuestion,
    );
    expect(screen.getByTestId("planning-setup-preview-clarity")).toHaveAttribute("data-emphasized", "true");
    expect(screen.getByTestId("planning-setup-preview-clarity")).toHaveTextContent("Clarity 3/5");
    expect(screen.getByTestId("planning-setup-preview-clarity")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewClaritySummary,
    );
    expect(screen.getByTestId("planning-setup-preview-confidence")).toHaveAttribute("data-emphasized", "false");
    expect(screen.getByTestId("planning-setup-preview-note")).toHaveAttribute("data-tone", "ready");

    await user.click(screen.getByTestId("planning-setup-finish"));

    expect(mocks.updateFactory).toHaveBeenCalledWith({
      planning: { enabled: true, clarity: true, confidence: true, setupCompleted: true },
    });
    expect(onFinished).toHaveBeenCalledOnce();
  });

  it("returns from Clarity to Confidence to the board", async () => {
    const user = userEvent.setup();
    const { onClose } = renderDialog();

    await user.click(screen.getByTestId("planning-setup-continue"));
    await user.click(screen.getByTestId("planning-setup-continue"));
    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepClarity })).toBeInTheDocument();

    await user.click(screen.getByTestId("planning-setup-back"));
    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepConfidence })).toBeInTheDocument();

    await user.click(screen.getByTestId("planning-setup-back"));
    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepRefine })).toBeInTheDocument();

    await user.click(screen.getByTestId("planning-setup-back"));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
