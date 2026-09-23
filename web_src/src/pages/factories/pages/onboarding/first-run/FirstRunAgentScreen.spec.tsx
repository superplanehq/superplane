import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { JIRA_COMPLETION_COLUMN_COPY } from "../../jiraCompletionColumnCopy";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAgentScreen } from "./FirstRunAgentScreen";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
  }),
}));

describe("FirstRunAgentScreen", () => {
  it("shows the completion column outside the agent card when a Jira project is set", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(
      <FirstRunAgentScreen
        saving={false}
        loading={false}
        agentReady
        intro="Connect a coding agent."
        jiraCompletion={{
          organizationId: "org-1",
          integrationId: "jira-1",
          projectId: "PAY",
          value: { jiraMoveOnComplete: true, jiraCompletionColumn: "Done" },
          onChange,
        }}
        onContinue={vi.fn()}
      >
        <div data-testid="agent-step">Agent options</div>
      </FirstRunAgentScreen>,
    );

    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.agent.headline })).toBeInTheDocument();
    expect(screen.getByTestId("agent-step")).toBeInTheDocument();
    const completion = screen.getByTestId("jira-completion-column");
    expect(completion).toBeInTheDocument();
    expect(screen.getByText(JIRA_COMPLETION_COLUMN_COPY.section)).toBeInTheDocument();
    expect(screen.getByTestId("jira-move-on-complete").parentElement).not.toHaveClass("rounded-lg");
    await user.click(screen.getByTestId("jira-move-on-complete"));
    expect(onChange).toHaveBeenCalledWith({ jiraMoveOnComplete: false, jiraCompletionColumn: "Done" });
  });

  it("hides the completion column when the ticket source is not Jira", () => {
    render(
      <FirstRunAgentScreen
        saving={false}
        loading={false}
        agentReady
        intro="Connect a coding agent."
        onContinue={vi.fn()}
      >
        <div data-testid="agent-step">Agent options</div>
      </FirstRunAgentScreen>,
    );

    expect(screen.queryByTestId("jira-completion-column")).not.toBeInTheDocument();
  });
});
