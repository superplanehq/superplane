import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "bun:test";
import type { FactoriesFactoryPullRequestActivity } from "@/api-client";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { prFeedbackLogRunsFromPullRequests } from "../../useWorkOrderPRFeedbackRunHref";
import { LiveHeaderSpendProvider, useReportLiveHeaderSpend } from "../liveHeaderSpendContext";
import { SPLIT_RUN_RUNNING, splitRunFixtureForWorkOrder } from "../splitRunMocks";
import { AutomationsConsoleVariant } from "./AutomationsConsoleVariant";
import { renderConsole } from "./consoleTestSupport";

function feedbackFooterFixture(activities: FactoriesFactoryPullRequestActivity[]) {
  return splitRunFixtureForWorkOrder(OPEN_WORK_ORDER, {
    demoArtifacts: false,
    prFeedbackRuns: prFeedbackLogRunsFromPullRequests(
      [{ number: "12", activities }],
      [{ canvasId: "checks-app", name: "Fix pull request checks" }],
    ),
  });
}

function feedbackActivity(
  id: string,
  minutesAgo: number,
  usage: Partial<FactoriesFactoryPullRequestActivity> = {},
): FactoriesFactoryPullRequestActivity {
  return {
    title: "Fix failing checks",
    run: {
      id,
      canvasId: "checks-app",
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      createdAt: new Date(Date.now() - minutesAgo * 60_000).toISOString(),
    },
    ...usage,
  };
}

describe("AutomationsConsoleVariant run footer", () => {
  it("keeps repair usage after checks pass and excludes other automations", () => {
    const passed = feedbackActivity("passed", 1, { title: "Checks passed" });
    renderConsole(
      feedbackFooterFixture([
        feedbackActivity("repair-1", 3, { costCents: "12.4", totalTokens: "1200", models: ["claude-sonnet-4-6"] }),
        feedbackActivity("repair-2", 2, {
          costCents: "23.4",
          totalTokens: "2300",
          models: ["claude-sonnet-4-6", "gpt-5"],
        }),
        passed,
        feedbackActivity("other", 1, {
          costCents: "900",
          totalTokens: "90000",
          models: ["other-model"],
          run: { ...passed.run, id: "other", canvasId: "other-app" },
        }),
      ]),
    );
    fireEvent.click(screen.getByRole("button", { name: "Toggle Fix pull request checks details" }));
    const footer = screen.getByTestId("redesign-console-run-footer-pr-feedback-passed");
    expect(footer).toHaveTextContent("$0.36");
    expect(footer).toHaveTextContent("3.5k");
    expect(footer).toHaveTextContent("sonnet 4-6 · gpt-5");
    expect(footer.textContent?.match(/sonnet 4-6/g)).toHaveLength(1);
    expect(footer).not.toHaveTextContent("other-model");
    expect(footer).toHaveTextContent(formatWorkOrderDateTime(new Date(passed.run?.createdAt ?? "")));
  });

  it.each([
    { recordedCents: "20", recordedTokens: "3000", expectedCost: "$0.55", expectedTokens: "5k" },
    { recordedCents: "50", recordedTokens: "5000", expectedCost: "$0.60", expectedTokens: "6k" },
  ])("adds earlier usage to the larger recorded or live latest values: %j", (usage) => {
    renderConsole(
      feedbackFooterFixture([
        feedbackActivity("earlier", 2, { costCents: "10", totalTokens: "1000" }),
        feedbackActivity("latest", 1, { costCents: usage.recordedCents, totalTokens: usage.recordedTokens }),
      ]),
      {},
      ["pr-feedback-latest:agent", 4000, 45],
    );
    fireEvent.click(screen.getByRole("button", { name: "Toggle Fix pull request checks details" }));
    const footer = screen.getByTestId("redesign-console-run-footer-pr-feedback-latest");
    expect(footer).toHaveTextContent(usage.expectedCost);
    expect(footer).toHaveTextContent(usage.expectedTokens);
  });

  it("omits empty usage for a single run", () => {
    renderConsole(feedbackFooterFixture([feedbackActivity("empty", 1)]));
    fireEvent.click(screen.getByRole("button", { name: "Toggle Fix pull request checks details" }));
    const footer = screen.getByTestId("redesign-console-run-footer-pr-feedback-empty");
    expect(within(footer).queryByRole("button", { name: "Show usage" })).not.toBeInTheDocument();
    expect(footer).not.toHaveTextContent("$");
  });

  it("shows recorded spend and model on the open card footer", () => {
    renderConsole({
      ...SPLIT_RUN_RUNNING,
      phases: SPLIT_RUN_RUNNING.phases.map((phase) =>
        phase.id === "implement"
          ? { ...phase, costCents: "45", totalTokens: "2700", startedAt: "2026-09-30T00:59:00.000Z" }
          : phase,
      ),
    });

    const footer = screen.getByTestId("redesign-console-run-footer-implement");
    expect(footer).toHaveTextContent(formatWorkOrderDateTime(new Date("2026-09-30T00:59:00.000Z")));
    expect(footer).not.toHaveTextContent("Started");
    expect(footer).toHaveTextContent("$0.45");
    expect(footer).toHaveTextContent("2.7k");
    expect(footer).toHaveTextContent("sonnet 4-6");
  });

  it("updates the footer spend from live telemetry", () => {
    function ReportLiveSpend() {
      useReportLiveHeaderSpend("implement:agent", 2100, 45);
      return (
        <AutomationsConsoleVariant
          fixture={SPLIT_RUN_RUNNING}
          source={SPLIT_RUN_RUNNING.source}
          organizationId="org-1"
        />
      );
    }

    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <ThemeProvider>
            <TooltipProvider>
              <LiveHeaderSpendProvider>
                <ReportLiveSpend />
              </LiveHeaderSpendProvider>
            </TooltipProvider>
          </ThemeProvider>
        </MemoryRouter>
      </QueryClientProvider>,
    );

    const footer = screen.getByTestId("redesign-console-run-footer-implement");
    expect(footer).toHaveTextContent("$0.45");
    expect(footer).toHaveTextContent("2.1k");
    expect(footer).toHaveTextContent("sonnet 4-6");
  });
});
