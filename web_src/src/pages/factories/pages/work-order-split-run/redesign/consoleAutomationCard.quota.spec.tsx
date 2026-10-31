import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { expect, it, mock } from "bun:test";
import { MemoryRouter } from "react-router";
import { OPEN_WORK_ORDER } from "../../../__fixtures__/factoryPageResponses";
import { splitRunFixtureForWorkOrder, type SplitRunPhase } from "../splitRunMocks";
import { stageFromPhase } from "./automationsViewModel";

mock.module("../useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({ enabled: false, isError: false, isLoading: false, stream: [] }),
}));
const { ConsoleAutomationCard } = await import("./consoleAutomationCard");

const failure = { reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" };
const phase: SplitRunPhase = {
  id: "analysis",
  name: "Backlog",
  componentName: "Backlog",
  status: "failed",
  duration: "1m",
  artifacts: [],
  stream: [],
  canvasSteps: [],
  subscriptionFailure: failure,
};

it("shows the quota notice on the default console summary while its logs are closed", () => {
  const latest = stageFromPhase(phase);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <ConsoleAutomationCard
          automation={{ id: "backlog", name: "Backlog", latest, runs: [latest] }}
          phase={phase}
          phases={[phase]}
          expandIdle={false}
          canStopRun={false}
          actionBusy={false}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  expect(screen.getByRole("button", { name: "Toggle Backlog details" })).toHaveAttribute("aria-expanded", "false");
  expect(screen.getByRole("alert", { name: "ChatGPT limit reached" })).toHaveTextContent(
    "This run stopped because the ChatGPT subscription reached its usage limit.",
  );
});

it("keeps an earlier planning quota failure in history after another attempt starts", () => {
  const resumed = {
    ...phase,
    status: "running" as const,
    subscriptionFailure: undefined,
    subscriptionFailureHistory: [{ runId: "old-attempt", failure }],
  };
  const latest = stageFromPhase(resumed);
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <ConsoleAutomationCard
          automation={{ id: "backlog", name: "Backlog", latest, runs: [latest] }}
          phase={resumed}
          phases={[resumed]}
          expandIdle={false}
          canStopRun={false}
          actionBusy={false}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  const history = screen.getByRole("region", { name: "Earlier attempt" });
  expect(within(history).getByRole("alert", { name: "ChatGPT limit reached" })).toBeInTheDocument();
  expect(screen.getAllByRole("alert", { name: "ChatGPT limit reached" })).toHaveLength(1);
});

it.each(["PR feedback", "linked column app", "described column app"] as const)(
  "shows a saved %s quota failure in the task execution summary",
  (source) => {
    const run = {
      id: "quota-run",
      canvasId: "quota-app",
      state: "STATE_FINISHED" as const,
      result: "RESULT_PASSED" as const,
      createdAt: new Date(Date.now() - 60000).toISOString(),
      subscriptionFailure: failure,
      usageState: "partial",
    };
    const fixture = splitRunFixtureForWorkOrder(
      {
        ...OPEN_WORK_ORDER,
        pullRequests: source === "linked column app" ? [{ runs: [{ run }] }] : [],
      },
      {
        demoArtifacts: false,
        prFeedbackRuns: source === "PR feedback" ? [{ canvasId: "quota-app", run }] : [],
        columnApps: [{ id: "quota-app", name: "Check pull request", columnKey: "verify" }],
        checks:
          source === "PR feedback"
            ? []
            : [{ name: "Check", runId: run.id, automation: { appId: "quota-app" }, score: 3 }],
        columnAppRuns:
          source === "described column app" ? { runsById: new Map([[run.id, run]]), loadingIds: new Set() } : undefined,
      },
    );
    const saved = fixture.phases.find((entry) => entry.runId === run.id);
    if (!saved) throw new Error("The linked activity must appear in the task summary");
    expect(saved.status).toBe("failed");
    expect(saved.usageState).toBe("partial");
    const latest = stageFromPhase(saved);
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <ConsoleAutomationCard
            automation={{ id: "quota-app", name: "Check pull request", latest, runs: [latest] }}
            phase={saved}
            phases={[saved]}
            expandIdle={false}
            canStopRun={false}
            actionBusy={false}
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    expect(screen.getByRole("alert", { name: "ChatGPT limit reached" })).toBeInTheDocument();
  },
);
