import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { NextStepsPanel, WorkspaceNextStepsHeaderBadge } from "./NextStepsPanel";
import { workspaceNextSteps } from "./workspaceNextStepCatalog";

const planningSteps = workspaceNextSteps({
  onboardingComplete: true,
  canConfigure: true,
  takenPRFeedbackSources: [],
  prFeedbackHandlersReady: true,
  planningSetupReady: true,
  planningSetupCompleted: false,
});

const commentsSteps = workspaceNextSteps({
  onboardingComplete: true,
  canConfigure: true,
  takenPRFeedbackSources: [],
  prFeedbackHandlersReady: true,
  planningSetupReady: true,
  planningSetupCompleted: true,
});

const checksSteps = workspaceNextSteps({
  onboardingComplete: true,
  canConfigure: true,
  takenPRFeedbackSources: ["discussion"],
  prFeedbackHandlersReady: true,
  planningSetupReady: true,
  planningSetupCompleted: true,
});

describe("NextStepsPanel", () => {
  it("does not offer Later for the required Planning step", () => {
    render(<NextStepsPanel steps={planningSteps} onContinue={vi.fn()} onDefer={vi.fn()} />);

    expect(screen.getByTestId("workspace-next-step-cta-planning-setup")).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-next-step-later")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps")).toHaveClass("workspace-next-steps-reveal");
  });

  it("does not offer Later for the required comments step", () => {
    render(<NextStepsPanel steps={commentsSteps} onContinue={vi.fn()} onDefer={vi.fn()} />);

    expect(screen.getByTestId("workspace-next-step-cta-pr-comments-handler")).toBeInTheDocument();
    expect(screen.queryByTestId("workspace-next-step-later")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-next-steps")).toHaveClass("workspace-next-steps-reveal");
  });

  it("offers Later for the optional status-checks step", async () => {
    const user = userEvent.setup();
    const onDefer = vi.fn();
    render(<NextStepsPanel steps={checksSteps} onContinue={vi.fn()} onDefer={onDefer} />);

    const later = screen.getByTestId("workspace-next-step-later");
    const configure = screen.getByTestId("workspace-next-step-cta-pr-checks-handler");
    expect(configure).toHaveTextContent("Configure");
    expect(later.compareDocumentPosition(configure) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    await user.click(later);
    expect(onDefer).toHaveBeenCalledOnce();
  });

  it("hides the banner when the optional step is deferred", () => {
    render(<NextStepsPanel steps={checksSteps} collapsed onContinue={vi.fn()} onDefer={vi.fn()} />);

    expect(screen.queryByTestId("workspace-next-steps")).not.toBeInTheDocument();
  });
});

describe("WorkspaceNextStepsHeaderBadge", () => {
  it("keeps the call to action next to the progress count", async () => {
    const user = userEvent.setup();
    const onOpen = vi.fn();
    render(<WorkspaceNextStepsHeaderBadge progress="4/5" title="Configure status checks" onOpen={onOpen} />);

    const restore = screen.getByTestId("workspace-next-steps-restore");
    expect(restore).toHaveTextContent("4/5");
    expect(restore).toHaveTextContent("Configure status checks");
    expect(restore).toHaveAccessibleName("Show next steps. Configure status checks");
    await user.click(restore);
    expect(onOpen).toHaveBeenCalledOnce();
  });
});
