import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { createWithAgentSurveyMessage, emptyCreateWithAgentView, runningCreateWithAgentView } from "./createWithAgentDemo";
import { CreateWithAgentDialog, type CreateWithAgentDialogProps } from "./CreateWithAgentDialog";
import { PLANNING_SESSION_PHASE_ID } from "./planningSessionActivity";

vi.mock("@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream", () => ({
  useLiveLogStream: () => ({
    sections: [],
    orphanLines: [],
    error: null,
    isStreaming: false,
    toggleSection: () => undefined,
    scrollRef: { current: null },
  }),
  terminalCommandStatusForExecution: () => null,
  terminalTimeMsForExecution: () => null,
}));

const noop = () => undefined;

function renderDialog(view: CreateWithAgentDialogProps["view"]) {
  return render(
    <CreateWithAgentDialog
      open
      workspaceName="Refunds"
      view={view}
      onComposerChange={noop}
      onSend={noop}
      onAnswerSurvey={noop}
      onDraftTitleChange={noop}
      onDraftDescriptionChange={noop}
      onCreateDraft={noop}
      onSkipDraft={noop}
      onWorkOnNew={noop}
      onSelectCreated={noop}
      onOpenCreated={noop}
      onBackToList={noop}
      onRequestClose={noop}
      onCancelEnd={noop}
      onConfirmEnd={noop}
    />,
  );
}

describe("CreateWithAgentDialog", () => {
  it("shows the Automations stream instead of chat while the agent waits", () => {
    renderDialog(runningCreateWithAgentView());

    expect(screen.getByTestId("create-with-agent-dialog")).toBeInTheDocument();
    expect(screen.getByTestId(`split-run-phase-${PLANNING_SESSION_PHASE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId("create-with-agent-stream")).toHaveTextContent(CREATE_WITH_AGENT_COPY.menu);
    expect(screen.getByTestId("create-with-agent-stream")).toHaveTextContent("Agent");
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.emptyHeadline)).toBeInTheDocument();
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.greeting)).not.toBeInTheDocument();
    expect(screen.queryByTestId("create-with-agent-message-greet")).not.toBeInTheDocument();
    expect(screen.queryByTestId("create-with-agent-chat")).not.toBeInTheDocument();
  });

  it("shows the Automations stream while the machine is starting", () => {
    renderDialog(emptyCreateWithAgentView());

    expect(screen.getByTestId("create-with-agent-machine")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineStarting);
    expect(screen.getByTestId(`split-run-phase-${PLANNING_SESSION_PHASE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId("create-with-agent-composer")).toBeEnabled();
    expect(screen.queryByTestId("create-with-agent-activity-starting")).not.toBeInTheDocument();
  });

  it("shows a survey above the composer without chat bubbles", () => {
    renderDialog(
      runningCreateWithAgentView({
        messages: [
          { id: "user-1", kind: "text", role: "user", text: "Add a health check for refunds." },
          createWithAgentSurveyMessage(),
        ],
      }),
    );

    expect(screen.getByTestId("work-order-survey-card")).toBeInTheDocument();
    expect(screen.queryByTestId("create-with-agent-message-user-1")).not.toBeInTheDocument();
    expect(screen.queryByText("Add a health check for refunds.")).not.toBeInTheDocument();
  });

  it("asks before the session ends", () => {
    const onConfirmEnd = vi.fn();
    render(
      <CreateWithAgentDialog
        open
        workspaceName="Refunds"
        view={runningCreateWithAgentView({ endConfirmOpen: true })}
        onComposerChange={noop}
        onSend={noop}
        onAnswerSurvey={noop}
        onDraftTitleChange={noop}
        onDraftDescriptionChange={noop}
        onCreateDraft={noop}
        onSkipDraft={noop}
        onWorkOnNew={noop}
        onSelectCreated={noop}
        onOpenCreated={noop}
        onBackToList={noop}
        onRequestClose={noop}
        onCancelEnd={noop}
        onConfirmEnd={onConfirmEnd}
      />,
    );

    expect(screen.getByTestId("create-with-agent-end-confirm")).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.endSessionAsk)).toBeInTheDocument();
  });
});
