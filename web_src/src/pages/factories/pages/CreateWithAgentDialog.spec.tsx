import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  emptyCreateWithAgentView,
  runningCreateWithAgentView,
  waitingCreateWithAgentView,
} from "./createWithAgentDemo";
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
      onSubmitSurvey={noop}
      onDraftTitleChange={noop}
      onDraftDescriptionChange={noop}
      onCreateDraft={noop}
      onSkipDraft={noop}
      onOpenCreated={noop}
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
    expect(screen.getByTestId("create-with-agent-stream")).toHaveTextContent(CREATE_WITH_AGENT_COPY.greeting);
    expect(screen.queryByTestId("create-with-agent-message-greet")).not.toBeInTheDocument();
    expect(screen.queryByTestId("create-with-agent-chat")).not.toBeInTheDocument();
  });

  it("shows Waiting for you when the machine is on and SuperPlane waits", () => {
    renderDialog(waitingCreateWithAgentView());

    expect(screen.getByTestId("create-with-agent-machine")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineWaiting);
    expect(screen.getByTestId("create-with-agent-machine")).toHaveTextContent("acme/payments");
  });

  it("shows the Automations stream while the machine is starting", () => {
    renderDialog(emptyCreateWithAgentView());

    expect(screen.getByTestId("create-with-agent-machine")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineStarting);
    expect(screen.getByTestId(`split-run-phase-${PLANNING_SESSION_PHASE_ID}`)).toBeInTheDocument();
    expect(screen.getByTestId("create-with-agent-composer")).toBeEnabled();
    expect(screen.queryByTestId("create-with-agent-activity-starting")).not.toBeInTheDocument();
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
        onSubmitSurvey={noop}
        onDraftTitleChange={noop}
        onDraftDescriptionChange={noop}
        onCreateDraft={noop}
        onSkipDraft={noop}
        onOpenCreated={noop}
        onRequestClose={noop}
        onCancelEnd={noop}
        onConfirmEnd={onConfirmEnd}
      />,
    );

    expect(screen.getByTestId("create-with-agent-end-confirm")).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.endSessionAsk)).toBeInTheDocument();
  });

  it("shows a survey form above the chat", () => {
    renderDialog(
      runningCreateWithAgentView({
        survey: {
          questions: [{ prompt: "What is the priority?", options: ["High", "Low"] }],
        },
      }),
    );

    expect(screen.getByTestId("create-with-agent-survey")).toBeInTheDocument();
    expect(screen.getByText("What is the priority?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.skipSurvey })).toBeInTheDocument();
  });
});
