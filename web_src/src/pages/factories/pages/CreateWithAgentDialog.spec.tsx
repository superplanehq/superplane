import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import {
  emptyCreateWithAgentView,
  failedCreateWithAgentView,
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
      onSelectCreated={noop}
      onRefineCreated={noop}
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

  it("shows that the machine stopped and turns the composer off", () => {
    renderDialog(failedCreateWithAgentView({ composer: "hey" }));

    expect(screen.getByTestId("create-with-agent-machine")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineStopped);
    expect(screen.getByTestId("split-run-attention-note")).toHaveTextContent(CREATE_WITH_AGENT_COPY.machineFailedBody);
    expect(screen.getByTestId("create-with-agent-composer")).toBeDisabled();
    expect(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.send })).toBeDisabled();
    expect(screen.queryByText("Waiting for logs…")).not.toBeInTheDocument();
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
        onSelectCreated={noop}
        onRefineCreated={noop}
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

  it("hides the session list until a task exists", () => {
    renderDialog(runningCreateWithAgentView());

    expect(screen.queryByTestId("create-with-agent-created")).not.toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.emptyHeadline)).toBeInTheDocument();
    expect(screen.queryByText(/work order/i)).not.toBeInTheDocument();
  });

  it("lists created tasks and opens one read-only from the title", () => {
    const onSelectCreated = vi.fn();
    const order = {
      id: "wo-1",
      key: "NEW-1",
      title: "Retry refunds",
      description: "Stop double charges.",
    };
    render(
      <CreateWithAgentDialog
        open
        workspaceName="Refunds"
        view={runningCreateWithAgentView({ created: [order] })}
        onComposerChange={noop}
        onSend={noop}
        onSubmitSurvey={noop}
        onDraftTitleChange={noop}
        onDraftDescriptionChange={noop}
        onCreateDraft={noop}
        onSkipDraft={noop}
        onSelectCreated={onSelectCreated}
        onRefineCreated={noop}
        onRequestClose={noop}
        onCancelEnd={noop}
        onConfirmEnd={noop}
      />,
    );

    expect(screen.getByTestId("create-with-agent-created")).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.sessionList)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "NEW-1 Retry refunds" })).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: CREATE_WITH_AGENT_COPY.refineFurther })).toHaveLength(1);
    expect(screen.queryByRole("button", { name: CREATE_WITH_AGENT_COPY.openTask })).not.toBeInTheDocument();

    screen.getByRole("button", { name: "NEW-1 Retry refunds" }).click();
    expect(onSelectCreated).toHaveBeenCalledWith(order);
  });

  it("shows a read-only task without edit fields", () => {
    const order = {
      id: "wo-1",
      key: "NEW-1",
      title: "Retry refunds",
      description: "Stop double charges.",
    };
    renderDialog(
      runningCreateWithAgentView({
        created: [order],
        right: { kind: "preview", order },
      }),
    );

    expect(screen.getByTestId("create-with-agent-preview")).toBeInTheDocument();
    expect(screen.getByText("Stop double charges.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Task title")).not.toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: CREATE_WITH_AGENT_COPY.refineFurther }).length).toBeGreaterThan(0);
  });

  it("hides the older-messages bar while the log follows the latest line", () => {
    renderDialog(runningCreateWithAgentView());

    expect(screen.getByTestId("create-with-agent-log")).toBeInTheDocument();
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("shows jump to latest after the user scrolls up, then hides it at the bottom", async () => {
    const user = userEvent.setup();
    renderDialog(runningCreateWithAgentView());
    const scroller = screen.getByTestId("create-with-agent-log");
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: CREATE_WITH_AGENT_COPY.jumpToLatest }));
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });

  it("turns following back on when the user scrolls to the latest line", async () => {
    renderDialog(runningCreateWithAgentView());
    const scroller = screen.getByTestId("create-with-agent-log");
    Object.defineProperty(scroller, "scrollHeight", { configurable: true, get: () => 400 });
    Object.defineProperty(scroller, "clientHeight", { configurable: true, get: () => 100 });
    await new Promise<void>((resolve) => {
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
    });

    scroller.scrollTop = 0;
    fireEvent.scroll(scroller);
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.viewingOlder)).toBeInTheDocument();

    scroller.scrollTop = 300;
    fireEvent.scroll(scroller);
    expect(screen.queryByText(CREATE_WITH_AGENT_COPY.viewingOlder)).not.toBeInTheDocument();
  });
});
