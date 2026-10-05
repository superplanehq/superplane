import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { intakeCatalogAvailability, seededIntakeCatalog } from "@/test/intakeCatalog";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";

const jiraLive = intakeCatalogAvailability(
  seededIntakeCatalog([], { "jira-issues": { status: "ga", available: true } }),
);

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
        intakeState={jiraLive.stateOf}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={vi.fn()}
        onConnectJira={onConnectJira}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Connect Jira" }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("jira");
    expect(onConnectJira).toHaveBeenCalledTimes(1);

    expect(screen.getByText("Coming soon")).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jiraHelper)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Linear/ })).toBeDisabled();
  });

  it("shows Jira and Linear as coming soon when Jira is unavailable", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={vi.fn()}
        onConnectJira={vi.fn()}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jiraSoonHelper)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linearSoonHelper)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.jiraHelper)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Jira" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /GitHub Issues/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Linear/ })).toBeInTheDocument();
    expect(screen.getAllByText("Coming soon")).toHaveLength(2);

    await user.click(screen.getByText(FIRST_RUN_COPY.tickets.jira));
    expect(onSelectTicketSource).not.toHaveBeenCalled();
  });

  it("does not mark Jira as coming soon while the intake catalog loads", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        intakesLoading
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={vi.fn()}
        onConnectJira={vi.fn()}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jiraLookupLoading)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.jiraSoonHelper)).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Jira" })).not.toBeInTheDocument();
    expect(screen.getAllByText("Coming soon")).toHaveLength(1);
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira).closest('[data-soon="true"]')).not.toBeInTheDocument();

    await user.click(screen.getByText(FIRST_RUN_COPY.tickets.jira));
    expect(onSelectTicketSource).not.toHaveBeenCalled();
  });

  it("explains a saved Jira choice when the intake catalog lookup fails and keeps scan stopped", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();

    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        jiraChoiceBlock="lookup-failed"
        jiraConnected
        jiraProjectId="PAY"
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira).closest('[data-soon="true"]')).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-jira-projects")).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-jira-choice-notice")).toHaveTextContent(
      FIRST_RUN_COPY.tickets.jiraLookupFailed,
    );
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();

    await user.click(screen.getByRole("button", { name: /GitHub Issues/ }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("github-issues");
  });

  it("explains a saved Jira choice while the intake catalog still loads", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        intakesLoading
        jiraChoiceBlock="loading"
        jiraConnected
        jiraProjectId="PAY"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-jira-choice-notice")).toHaveTextContent(
      FIRST_RUN_COPY.tickets.jiraLookupLoading,
    );
    expect(screen.getByText(FIRST_RUN_COPY.tickets.jira).closest('[data-soon="true"]')).not.toBeInTheDocument();
    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
  });

  it("lets the user select Linear when the feature is on and keeps scan stopped until a project is chosen", async () => {
    const user = userEvent.setup();
    const onSelectTicketSource = vi.fn();
    const onConnectLinear = vi.fn();
    const onToggleLinearProject = vi.fn();
    const onAnalyzeTickets = vi.fn();

    const { rerender } = render(
      <FirstRunTicketsScreen
        ticketSource={null}
        linearAvailable
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={onAnalyzeTickets}
        onConnectLinear={onConnectLinear}
        onToggleLinearProject={onToggleLinearProject}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.tickets.linearHelper)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.tickets.linearSoonHelper)).not.toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linear).closest('[data-soon="true"]')).not.toBeInTheDocument();
    expect(screen.getAllByText("Coming soon")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Connect Linear" }));
    expect(onSelectTicketSource).toHaveBeenCalledWith("linear");
    expect(onConnectLinear).toHaveBeenCalledTimes(1);

    rerender(
      <FirstRunTicketsScreen
        ticketSource="linear"
        linearAvailable
        linearConnected
        linearProjects={[{ id: "project-1", name: "Platform" }]}
        linearProjectIds={[]}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={onAnalyzeTickets}
        onToggleLinearProject={onToggleLinearProject}
      />,
    );

    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linearProjectHeading)).toBeInTheDocument();
    await user.click(screen.getByTestId("linear-project-project-1"));
    expect(onToggleLinearProject).toHaveBeenCalledWith("project-1");

    rerender(
      <FirstRunTicketsScreen
        ticketSource="linear"
        linearAvailable
        linearConnected
        linearProjects={[{ id: "project-1", name: "Platform" }]}
        linearProjectIds={["project-1"]}
        onSelectTicketSource={onSelectTicketSource}
        onAnalyzeTickets={onAnalyzeTickets}
        onToggleLinearProject={onToggleLinearProject}
      />,
    );

    const analyze = screen.getByTestId("first-run-analyze-tickets");
    expect(analyze).toBeEnabled();
    await user.click(analyze);
    expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
  });

  it("keeps scan stopped while Linear projects are still loading", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="linear"
        linearAvailable
        linearConnected
        linearProjectsLoading
        linearProjectIds={["project-gone"]}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
  });

  it("keeps scan stopped when Linear projects fail to load", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="linear"
        linearAvailable
        linearConnected
        linearProjectsError
        linearProjectIds={["project-gone"]}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
  });

  it("keeps scan stopped until Jira is connected and a project is chosen", async () => {
    const user = userEvent.setup();
    const onSelectJiraProject = vi.fn();
    const onAnalyzeTickets = vi.fn();

    const { rerender } = render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        intakeState={jiraLive.stateOf}
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
        intakeState={jiraLive.stateOf}
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
        intakeState={jiraLive.stateOf}
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
        intakeState={jiraLive.stateOf}
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

  it("does not show the completion column setting after a Jira project is selected", () => {
    render(
      <FirstRunTicketsScreen
        ticketSource="jira"
        intakeState={jiraLive.stateOf}
        jiraConnected
        jiraProjects={[{ id: "PAY", name: "Payments" }]}
        jiraProjectId="PAY"
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.getByTestId("first-run-jira-projects")).toBeInTheDocument();
    expect(screen.queryByTestId("jira-completion-column")).not.toBeInTheDocument();
  });

  it("orders ticket intakes from the catalog and marks a Beta intake", () => {
    const catalog = intakeCatalogAvailability(seededIntakeCatalog(["jira-issues"]));

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        intakeState={catalog.stateOf}
        ticketIntakes={catalog.entriesFor("onboardingTickets")}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    const titles = screen.getAllByRole("button", { pressed: false }).map((button) => button.textContent ?? "");
    expect(titles[0]).toContain(FIRST_RUN_COPY.tickets.githubIssues);
    expect(titles[1]).toContain(FIRST_RUN_COPY.tickets.jira);
    expect(titles[1]).toContain(FIRST_RUN_COPY.tickets.beta);
    expect(screen.getByRole("button", { name: "Connect Jira" })).toBeEnabled();
  });

  it("leaves out an Internal intake that the company cannot use and lists an admin-added intake as coming soon", () => {
    const catalog = intakeCatalogAvailability(
      seededIntakeCatalog([], {
        "jira-issues": { status: "alpha", available: false },
        "azure-boards": { name: "Azure Boards", category: "issue_tracking", status: "planned" },
      }),
    );

    render(
      <FirstRunTicketsScreen
        ticketSource={null}
        intakeState={catalog.stateOf}
        ticketIntakes={catalog.entriesFor("onboardingTickets")}
        onSelectTicketSource={vi.fn()}
        onAnalyzeTickets={vi.fn()}
      />,
    );

    expect(screen.queryByText(FIRST_RUN_COPY.tickets.jira)).not.toBeInTheDocument();
    expect(screen.getByText("Azure Boards").closest('[data-soon="true"]')).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.tickets.linear).closest('[data-soon="true"]')).toBeInTheDocument();
  });
});
