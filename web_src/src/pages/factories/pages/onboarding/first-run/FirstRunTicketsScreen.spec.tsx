import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";
import { JIRA_COMPLETION_COLUMN_COPY } from "../../jiraCompletionColumnCopy";

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
  }),
}));

describe("FirstRunTicketsScreen", () => {
  it("keeps analysis stopped until a ticket system is selected", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();
    const onAnalyzeTickets = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={onAnalyzeTickets}
      />,
    );

    expect(screen.getByRole("heading", { name: FIRST_RUN_COPY.tickets.headline })).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.intro)).toBeInTheDocument();
    expect(screen.queryByText(/The analysis starts when you choose/)).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("github-issues");
    expect(onAnalyzeTickets).not.toHaveBeenCalled();
  });

  it("starts analysis from the button after a ticket system is selected", async () => {
    const user = userEvent.setup();
    const onAnalyzeTickets = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={onAnalyzeTickets}
      />,
    );

    const analyze = screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.analyze });
    expect(analyze).toBeEnabled();
    await user.click(analyze);
    expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
  });

  it("lets the user select Jira and keeps Linear as coming soon", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();
    const onConnectJira = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={vi.fn()}
        onConnectJira={onConnectJira}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Connect Jira" }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("jira");
    expect(onConnectJira).toHaveBeenCalledTimes(1);

    expect(screen.getByText("Coming soon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Linear/ })).toBeDisabled();
  });

  it("keeps scan stopped until Jira is connected and a project is chosen", async () => {
    const user = userEvent.setup();
    const onSelectJiraProject = vi.fn();
    const onAnalyzeTickets = vi.fn();

    const { rerender } = render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={onAnalyzeTickets}
        onConnectJira={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    expect(screen.getByRole("button", { name: "Connect Jira" })).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-jira-projects")).not.toBeInTheDocument();

    rerender(
      <FirstRunTicketsScreen
        ticketSource="jira"
        jiraConnected
        jiraProjects={[{ id: "PAY", name: "Payments" }]}
        jiraProjectId=""
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={onAnalyzeTickets}
        onSelectJiraProject={onSelectJiraProject}
      />,
    );

    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jiraProjectHeading)).toBeInTheDocument();
    await user.click(screen.getByTestId("jira-project-PAY"));
    expect(onSelectJiraProject).toHaveBeenCalledWith("PAY");
    expect(onAnalyzeTickets).not.toHaveBeenCalled();

    rerender(
      <FirstRunTicketsScreen
        ticketSource="jira"
        jiraConnected
        jiraProjects={[{ id: "PAY", name: "Payments" }]}
        jiraProjectId="PAY"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={onAnalyzeTickets}
        onSelectJiraProject={onSelectJiraProject}
      />,
    );

    const analyze = screen.getByTestId("first-run-analyze-tickets");
    expect(analyze).toBeEnabled();
    await user.click(analyze);
    expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
  });

  it("shows finish progress while the screen provisions the workspace", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        saving
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    const analyze = screen.getByTestId("first-run-analyze-tickets");
    expect(analyze).toHaveTextContent(FIRST_RUN_COPY.finish.saving);
    expect(analyze).toBeDisabled();
  });

  it("locks the Jira project picker while setup is saving", async () => {
    const user = userEvent.setup();
    const onSelectJiraProject = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        saving
        jiraConnected
        jiraProjects={[
          { id: "PAY", name: "Payments" },
          { id: "CORE", name: "Core" },
        ]}
        jiraProjectId="PAY"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
        onSelectJiraProject={onSelectJiraProject}
      />,
    );

    await user.click(screen.getByTestId("jira-project-CORE"));
    expect(onSelectJiraProject).not.toHaveBeenCalled();
  });

  it("uses the next-step label when setup names the coding agent step", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="github-issues"
        continueLabel={FIRST_RUN_COPY.tickets.continue}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: FIRST_RUN_COPY.tickets.continue })).toBeEnabled();
  });

  it("hides completion column settings when Jira has a Done status", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        jiraConnected
        jiraProjects={[{ id: "PAY", name: "Payments" }]}
        jiraProjectId="PAY"
        jiraCompletionNeedsManualColumn={false}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("jira-completion-column")).not.toBeInTheDocument();
  });

  it("shows completion column settings when Jira has no Done status", async () => {
    const user = userEvent.setup();
    const onJiraCompletionChange = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        jiraConnected
        organizationId="org-1"
        jiraIntegrationId="jira-1"
        jiraProjects={[{ id: "PAY", name: "Payments" }]}
        jiraProjectId="PAY"
        jiraCompletionNeedsManualColumn
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
        onJiraCompletionChange={onJiraCompletionChange}
      />,
    );

    expect(screen.getByTestId("jira-completion-column")).toBeInTheDocument();
    expect(screen.getByText(JIRA_COMPLETION_COLUMN_COPY.section)).toBeInTheDocument();
    expect(screen.getByTestId("jira-move-on-complete")).toBeChecked();
    await user.click(screen.getByTestId("jira-move-on-complete"));
    expect(onJiraCompletionChange).toHaveBeenCalledWith({
      jiraMoveOnComplete: false,
      jiraCompletionColumn: "",
    });
  });
});
