import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import { runningCreateWithAgentView } from "./createWithAgentDemo";
import { CreateWithAgentDialog } from "./CreateWithAgentDialog";

const noop = () => undefined;

describe("CreateWithAgentDialog", () => {
  it("shows the empty work pane while the agent waits for the first task", () => {
    render(
      <CreateWithAgentDialog
        open
        workspaceName="Refunds"
        view={runningCreateWithAgentView()}
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

    expect(screen.getByTestId("create-with-agent-dialog")).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.emptyHeadline)).toBeInTheDocument();
    expect(screen.getByText(CREATE_WITH_AGENT_COPY.greeting)).toBeInTheDocument();
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
