import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketsScreen } from "./FirstRunTicketsScreen";

const copy = FIRST_RUN_COPY.tickets;

describe("FirstRunTicketsScreen", () => {
  describe("source list", () => {
    it("lists the sources as plain choices with no Connect buttons", async () => {
      const user = userEvent.setup();
      const onSelectTicketSource = vi.fn();
      const onConnectJira = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource={null}
          githubConnected
          onSelectTicketSource={onSelectTicketSource}
          onAnalyzeTickets={vi.fn()}
          onConnectJira={onConnectJira}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.headline })).toBeInTheDocument();
      expect(screen.getByText(copy.intro)).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /^Connect/ })).not.toBeInTheDocument();
      expect(screen.queryByTestId("first-run-analyze-tickets")).not.toBeInTheDocument();
      expect(within(screen.getByRole("button", { name: /GitHub issues/ })).getByText(copy.connectedMeta)).toBeVisible();
      expect(within(screen.getByRole("button", { name: /Jira issues/ })).queryByText(copy.connectedMeta)).toBeNull();

      await user.click(screen.getByRole("button", { name: /Jira issues/ }));
      expect(onSelectTicketSource).toHaveBeenCalledWith("jira");
      expect(onConnectJira).not.toHaveBeenCalled();
    });

    it("stays on the list when the source is only a default, and opens the card on click", async () => {
      const user = userEvent.setup();
      const onSelectTicketSource = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          startOnSourceList
          githubConnected
          onSelectTicketSource={onSelectTicketSource}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.headline })).toBeInTheDocument();
      const githubIssues = screen.getByRole("button", { name: /GitHub issues/ });
      expect(githubIssues).toHaveAttribute("aria-pressed", "true");
      expect(screen.queryByTestId("first-run-analyze-tickets")).not.toBeInTheDocument();

      await user.click(githubIssues);
      expect(onSelectTicketSource).toHaveBeenCalledWith("github-issues");
      expect(screen.getByRole("heading", { name: copy.setupHeadline(copy.githubIssues) })).toBeInTheDocument();
      expect(screen.getByTestId("first-run-analyze-tickets")).toBeEnabled();
    });

    it("keeps Linear as coming soon while the feature lookup is loading and still lets the user pick Jira", async () => {
      const user = userEvent.setup();
      const onSelectTicketSource = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource={null}
          linearFeatureLoading
          onSelectTicketSource={onSelectTicketSource}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.linearLookupLoading)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /Linear issues/ })).toBeDisabled();
      expect(screen.queryByText("Coming soon")).not.toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: /Jira issues/ }));
      expect(onSelectTicketSource).toHaveBeenCalledWith("jira");
    });

    it("marks Linear as coming soon when the feature is off", () => {
      render(
        <FirstRunTicketsScreen
          ticketSource={null}
          linearAvailable={false}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.linearSoonHelper)).toBeInTheDocument();
      expect(screen.getByText(copy.linear).closest('[data-soon="true"]')).toBeInTheDocument();
    });
  });

  describe("GitHub issues card", () => {
    it("opens on intake settings with the connection and repository done", async () => {
      const user = userEvent.setup();
      const onAnalyzeTickets = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          repository="acme/payments"
          githubConnected
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={onAnalyzeTickets}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.setupHeadline(copy.githubIssues) })).toBeInTheDocument();
      expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "done");
      expect(screen.getByTestId("first-run-step-project")).toHaveAttribute("data-state", "done");
      expect(screen.getByText(copy.stepRepositoryDone("acme/payments"))).toBeInTheDocument();
      expect(screen.getByTestId("first-run-step-settings")).toHaveAttribute("data-state", "active");

      const analyze = screen.getByRole("button", { name: copy.analyze });
      expect(analyze).toBeEnabled();
      await user.click(analyze);
      expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
    });

    it("asks to connect GitHub first when the account is not connected", async () => {
      const user = userEvent.setup();
      const onConnectGitHub = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected={false}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
          onConnectGitHub={onConnectGitHub}
        />,
      );

      expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "active");
      expect(screen.queryByTestId("first-run-analyze-tickets")).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: copy.stepConnect(copy.githubIssues) }));
      expect(onConnectGitHub).toHaveBeenCalledTimes(1);
    });

    it("returns to the source list on Back, then hands Back to the shell", async () => {
      const user = userEvent.setup();
      const onBack = vi.fn();

      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          chrome={{ stepIndex: 3, onBack }}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      await user.click(screen.getByTestId("first-run-back"));
      expect(onBack).not.toHaveBeenCalled();
      expect(screen.getByRole("heading", { name: copy.headline })).toBeInTheDocument();
      expect(screen.getByRole("button", { name: /GitHub issues/ })).toHaveAttribute("aria-pressed", "true");

      await user.click(screen.getByTestId("first-run-back"));
      expect(onBack).toHaveBeenCalledTimes(1);
    });
  });

  describe("intake settings", () => {
    it("states the first workspace import and offers no way to skip it", () => {
      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          initialImportLimit={3}
          onAutomaticIntakeChange={vi.fn()}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.importFixed(3))).toBeInTheDocument();
      expect(screen.queryByRole("switch", { name: copy.importExisting })).not.toBeInTheDocument();
      expect(screen.getByRole("switch", { name: copy.automaticIntake })).toBeChecked();
    });

    it("lets later workspaces turn the import and automatic intake off and warns when both are off", async () => {
      const user = userEvent.setup();
      const onSkip = vi.fn();
      const onAutomatic = vi.fn();

      const { rerender } = render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          skipInitialImport={false}
          onSkipInitialImportChange={onSkip}
          automaticIntake
          onAutomaticIntakeChange={onAutomatic}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      const importSwitch = screen.getByRole("switch", { name: copy.importExisting });
      expect(importSwitch).toBeChecked();
      expect(screen.getByText(copy.importExistingOn(3))).toBeInTheDocument();
      await user.click(importSwitch);
      expect(onSkip).toHaveBeenCalledWith(true);

      await user.click(screen.getByRole("switch", { name: copy.automaticIntake }));
      expect(onAutomatic).toHaveBeenCalledWith(false);
      expect(screen.queryByTestId("onboarding-no-intake-hint")).not.toBeInTheDocument();

      rerender(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          skipInitialImport
          onSkipInitialImportChange={onSkip}
          automaticIntake={false}
          onAutomaticIntakeChange={onAutomatic}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByText(copy.importExistingOff)).toBeInTheDocument();
      expect(screen.getByText(copy.automaticIntakeOff)).toBeInTheDocument();
      expect(screen.getByTestId("onboarding-no-intake-hint")).toHaveTextContent(copy.nothingImportedHint);
    });
  });

  describe("Jira issues card", () => {
    it("walks connect, project, then settings before the scan", async () => {
      const user = userEvent.setup();
      const onConnectJira = vi.fn();
      const onSelectJiraProject = vi.fn();
      const onAnalyzeTickets = vi.fn();

      const { rerender } = render(
        <FirstRunTicketsScreen
          ticketSource="jira"
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={onAnalyzeTickets}
          onConnectJira={onConnectJira}
        />,
      );

      expect(screen.getByRole("heading", { name: copy.setupHeadline(copy.jira) })).toBeInTheDocument();
      expect(screen.queryByTestId("first-run-jira-projects")).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: copy.stepConnect(copy.jira) }));
      expect(onConnectJira).toHaveBeenCalledTimes(1);

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

      expect(screen.getByTestId("first-run-step-connect")).toHaveAttribute("data-state", "done");
      expect(screen.getByTestId("first-run-jira-projects")).toBeInTheDocument();
      expect(screen.getByTestId("first-run-confirm-projects")).toBeDisabled();
      await user.click(screen.getByTestId("jira-project-PAY"));
      expect(onSelectJiraProject).toHaveBeenCalledWith("PAY");

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

      expect(screen.queryByTestId("first-run-analyze-tickets")).not.toBeInTheDocument();
      await user.click(screen.getByTestId("first-run-confirm-projects"));

      expect(screen.getByText(copy.jiraProjectDone("Payments"))).toBeInTheDocument();
      expect(screen.getByTestId("first-run-step-settings")).toHaveAttribute("data-state", "active");
      expect(screen.queryByTestId("jira-completion-column")).not.toBeInTheDocument();
      const analyze = screen.getByTestId("first-run-analyze-tickets");
      expect(analyze).toBeEnabled();
      await user.click(analyze);
      expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
    });

    it("reopens the project step on Back and forgets the confirmation when the source changes", async () => {
      const user = userEvent.setup();
      const onBack = vi.fn();
      const props = {
        jiraConnected: true,
        jiraProjects: [{ id: "PAY", name: "Payments" }],
        jiraProjectId: "PAY",
        linearConnected: true,
        linearProjects: [{ id: "project-1", name: "Platform" }],
        linearProjectIds: ["project-1"],
        chrome: { stepIndex: 3, onBack },
        onSelectTicketSource: vi.fn(),
        onAnalyzeTickets: vi.fn(),
      };

      const { rerender } = render(<FirstRunTicketsScreen ticketSource="jira" {...props} />);

      await user.click(screen.getByTestId("first-run-confirm-projects"));
      expect(screen.getByTestId("first-run-step-settings")).toHaveAttribute("data-state", "active");

      await user.click(screen.getByTestId("first-run-back"));
      expect(onBack).not.toHaveBeenCalled();
      expect(screen.getByTestId("first-run-step-project")).toHaveAttribute("data-state", "active");

      await user.click(screen.getByTestId("first-run-confirm-projects"));
      rerender(<FirstRunTicketsScreen ticketSource="linear" {...props} />);
      expect(screen.getByTestId("first-run-step-project")).toHaveAttribute("data-state", "active");
      expect(screen.getByTestId("first-run-linear-projects")).toBeInTheDocument();
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
      expect(screen.getByTestId("first-run-confirm-projects")).toBeDisabled();
    });
  });

  describe("Linear issues card", () => {
    it("keeps the project step open until at least one project is chosen", async () => {
      const user = userEvent.setup();
      const onToggleLinearProject = vi.fn();
      const onAnalyzeTickets = vi.fn();

      const { rerender } = render(
        <FirstRunTicketsScreen
          ticketSource="linear"
          linearConnected
          linearProjects={[{ id: "project-1", name: "Platform" }]}
          linearProjectIds={[]}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={onAnalyzeTickets}
          onToggleLinearProject={onToggleLinearProject}
        />,
      );

      expect(screen.getByTestId("first-run-linear-projects")).toBeInTheDocument();
      expect(screen.getByTestId("first-run-confirm-projects")).toBeDisabled();
      await user.click(screen.getByTestId("linear-project-project-1"));
      expect(onToggleLinearProject).toHaveBeenCalledWith("project-1");

      rerender(
        <FirstRunTicketsScreen
          ticketSource="linear"
          linearConnected
          linearProjects={[{ id: "project-1", name: "Platform" }]}
          linearProjectIds={["project-1"]}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={onAnalyzeTickets}
          onToggleLinearProject={onToggleLinearProject}
        />,
      );

      await user.click(screen.getByTestId("first-run-confirm-projects"));
      expect(screen.getByText(copy.linearProjectsDone(1))).toBeInTheDocument();
      const analyze = screen.getByTestId("first-run-analyze-tickets");
      expect(analyze).toBeEnabled();
      await user.click(analyze);
      expect(onAnalyzeTickets).toHaveBeenCalledTimes(1);
    });

    it.each([{ linearProjectsLoading: true }, { linearProjectsError: true }])(
      "keeps the project step closed to confirmation while projects are %j",
      (state) => {
        render(
          <FirstRunTicketsScreen
            ticketSource="linear"
            linearConnected
            linearProjectIds={["project-gone"]}
            onSelectTicketSource={vi.fn()}
            onAnalyzeTickets={vi.fn()}
            {...state}
          />,
        );

        expect(screen.getByTestId("first-run-confirm-projects")).toBeDisabled();
      },
    );

    it("shows the lookup failure and keeps the scan stopped when Linear cannot be confirmed", async () => {
      const user = userEvent.setup();
      render(
        <FirstRunTicketsScreen
          ticketSource="linear"
          linearAvailable={false}
          linearChoiceBlock="lookup-failed"
          linearConnected
          linearProjects={[{ id: "project-1", name: "Platform" }]}
          linearProjectIds={["project-1"]}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByTestId("first-run-linear-choice-notice")).toHaveTextContent(copy.linearLookupFailed);
      await user.click(screen.getByTestId("first-run-confirm-projects"));
      expect(screen.getByTestId("first-run-analyze-tickets")).toBeDisabled();
    });
  });

  describe("final action", () => {
    it("shows finish progress while the screen provisions the workspace", () => {
      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          saving
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      const analyze = screen.getByTestId("first-run-analyze-tickets");
      expect(analyze).toHaveTextContent(FIRST_RUN_COPY.finish.saving);
      expect(analyze).toBeDisabled();
    });

    it("uses the next-step label when setup names the coding agent step", () => {
      render(
        <FirstRunTicketsScreen
          ticketSource="github-issues"
          githubConnected
          continueLabel={copy.continue}
          onSelectTicketSource={vi.fn()}
          onAnalyzeTickets={vi.fn()}
        />,
      );

      expect(screen.getByRole("button", { name: copy.continue })).toBeEnabled();
    });
  });
});
