import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DEFAULT_FACTORY_PLANNING } from "../__fixtures__/factoryPageResponses";
import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { PlanningSetupDialog } from "./PlanningSetupDialog";

const mocks = vi.hoisted(() => ({
  updateFactory: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useUpdateFactory: () => ({ mutateAsync: mocks.updateFactory, isPending: false }),
}));

describe("PlanningSetupDialog", () => {
  beforeEach(() => {
    mocks.updateFactory.mockReset();
    mocks.updateFactory.mockResolvedValue({
      id: "factory-1",
      planning: { ...DEFAULT_FACTORY_PLANNING, setupCompleted: true },
    });
  });

  it("shows the refine preview and skips scores when Planning is off", async () => {
    const user = userEvent.setup();
    const onFinished = vi.fn();
    render(
      <PlanningSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        factory={{ id: "factory-1", planning: { ...DEFAULT_FACTORY_PLANNING } }}
        onClose={vi.fn()}
        onFinished={onFinished}
      />,
    );

    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewRefineCaption,
    );
    await user.click(screen.getByRole("radio", { name: /Keep the source description/ }));
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewSourceCaption,
    );
    expect(screen.queryByTestId("planning-setup-continue")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("planning-setup-finish"));

    expect(mocks.updateFactory).toHaveBeenCalledWith({
      planning: { enabled: false, clarity: true, confidence: true, setupCompleted: true },
    });
    expect(onFinished).toHaveBeenCalledOnce();
  });

  it("asks for scores when Planning stays on and Finish writes setupCompleted", async () => {
    const user = userEvent.setup();
    const onFinished = vi.fn();
    render(
      <PlanningSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        factory={{ id: "factory-1", planning: { ...DEFAULT_FACTORY_PLANNING } }}
        onClose={vi.fn()}
        onFinished={onFinished}
      />,
    );

    await user.click(screen.getByTestId("planning-setup-continue"));
    expect(screen.getByRole("heading", { name: PLANNING_SETTINGS_COPY.wizardStepScores })).toBeInTheDocument();
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewScoresBoth,
    );
    expect(screen.getByTestId("planning-setup-preview-scores")).toHaveTextContent("Clarity");
    expect(screen.getByTestId("planning-setup-preview-scores")).toHaveTextContent("Confidence");

    await user.click(screen.getByRole("radio", { name: /Hide Clarity/ }));
    expect(screen.getByTestId("planning-setup-preview-caption")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.wizardPreviewConfidenceOnly,
    );

    await user.click(screen.getByTestId("planning-setup-finish"));

    expect(mocks.updateFactory).toHaveBeenCalledWith({
      planning: { enabled: true, clarity: false, confidence: true, setupCompleted: true },
    });
    expect(onFinished).toHaveBeenCalledOnce();
  });
});
