import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "bun:test";
import { AgentRunsPage } from "./consoleAgentRuns";
import { implementPhase, implementStage } from "./LiveAgentSteps.testHelpers";

vi.mock("../useSplitRunLiveCanvas", () => ({
  useSplitRunLiveCanvas: () => ({
    enabled: true,
    isLoading: false,
    isError: false,
    stream: [
      {
        id: "attach-pr",
        executionId: "attach-execution",
        componentName: "Attach PR",
        status: "failed",
        detail: "factory pull request already exists",
      },
    ],
  }),
}));

it("opens the failed agent-run activity in read-only run details", async () => {
  const stage = { ...implementStage(), status: "failed" as const, agentSteps: [] };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  try {
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter>
          <AgentRunsPage
            runs={[stage]}
            phases={[{ ...implementPhase(), id: stage.id, status: "failed", appId: "implement", runId: "failed-run" }]}
            automationName="Implement"
            organizationId="org"
            factoryKey="sp"
            orderNumber="106"
          />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    const link = await screen.findByRole("link", { name: "View run details" });
    expect(link).toHaveAttribute(
      "href",
      "/org/workspaces/sp/automations/implement/runs/failed-run?from=task&orderNumber=106&activity=attach-execution",
    );
  } finally {
    client.clear();
  }
});
