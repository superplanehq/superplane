import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useLocation, useNavigate } from "react-router";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { FactoryAutomationRunPage } from "./FactoryAutomationRunPage";
import { SplitRunReview } from "./work-order-split-run/SplitRunReview";
import { splitRunFixtureForWorkOrder } from "./work-order-split-run/splitRunMocks";
import { BOARD_IMPLEMENT_FAILED_ORDER } from "../__fixtures__/lineMetricsBoardOrders";
import { FailedNodeAlerts } from "./work-order-split-run/redesign/failedNodeAlerts";
import { splitRunPhaseRunHref } from "./work-order-split-run/splitRunPopupModel";

const api = vi.hoisted(() => ({
  canvasesListRuns: vi.fn(),
  canvasesDescribeRun: vi.fn(),
  canvasesListEventExecutions: vi.fn(),
  canvasesDescribeCanvasVersion: vi.fn(),
}));
vi.mock("@/api-client/sdk.gen", () => api);
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));
vi.mock("@/ui/Runs/RunInspectorRunnerLogs", () => ({ RunnerLogsTerminal: () => <p>Saved runner log</p> }));
vi.mock("./useFactoryAppCanvasRoute", () => ({
  useFactoryAppCanvasRoute: () => ({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "sp",
    appId: "app",
    canvas: { metadata: { name: "Implement" } },
    canvasLoading: false,
    shouldRedirect: false,
    from: "task",
    orderNumber: "106",
    lineId: "line",
    back: { href: "/org/workspaces/sp/task/106", label: "Back" },
  }),
}));

const runHref = "/org/workspaces/sp/automations/app/runs/run?from=task&lineId=line&orderNumber=106";
const execution = {
  id: "attach",
  runId: "run",
  nodeId: "attach-pr",
  componentName: "addWorkOrderPullRequest",
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  resultMessage: "factory pull request already exists",
};
const runner = {
  id: "agent",
  runId: "run",
  nodeId: "implement",
  componentName: "runnerSuperPlane",
  state: "STATE_FINISHED",
  result: "RESULT_PASSED",
  metadata: { runner_broker_task_id: "broker-task" },
};
const otherFailure = { ...execution, id: "earlier-failure", nodeId: "earlier", resultMessage: "An earlier failure" };
const run = {
  id: "run",
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  versionId: "version",
  rootEvent: { id: "event" },
  executions: [otherFailure, runner, execution],
};

function FailureActions() {
  const fixture = splitRunFixtureForWorkOrder({
    ...BOARD_IMPLEMENT_FAILED_ORDER,
    state: "STATE_OPEN",
    result: undefined,
  });
  const phase = { ...fixture.phases[0], appId: "app", runId: "run" };
  return (
    <>
      <div data-testid="summary">
        <SplitRunReview
          footer={{ ...fixture.footer, run: { appId: "app", runId: "run" } }}
          organizationId="org"
          factoryId="factory"
          factoryKey="sp"
          orderNumber="106"
        />
      </div>
      <FailedNodeAlerts
        nodes={[{ id: "attach-pr", executionId: "attach", name: "Attach PR", message: execution.resultMessage }]}
        runHref={splitRunPhaseRunHref({
          organizationId: "org",
          factoryKey: "sp",
          lineId: "line",
          orderNumber: "106",
          phase,
        })}
      />
    </>
  );
}
function BrowserControls() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="location">
        {location.pathname}
        {location.search}
      </output>
      <button onClick={() => navigate(-1)}>Browser back</button>
    </>
  );
}

describe("Failure run details navigation", () => {
  beforeEach(() => {
    api.canvasesListRuns.mockResolvedValue({ data: { runs: [run], totalCount: 1 } });
    api.canvasesDescribeRun.mockResolvedValue({ data: { run } });
    api.canvasesListEventExecutions.mockResolvedValue({ data: { executions: [otherFailure, runner, execution] } });
    api.canvasesDescribeCanvasVersion.mockResolvedValue({
      data: {
        version: {
          spec: {
            nodes: [
              { id: "earlier", name: "Earlier step" },
              { id: "attach-pr", name: "Attach PR", component: "addWorkOrderPullRequest" },
              { id: "implement", name: "Implement task", component: "runnerSuperPlane" },
            ],
          },
        },
      },
    });
  });

  it.each(["summary", "step"])("opens %s failure details without exposing the canvas", async (source) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    try {
      render(
        <QueryClientProvider client={client}>
          <ThemeProvider>
            <TooltipProvider>
              <MemoryRouter initialEntries={["/task"]}>
                <BrowserControls />
                <Routes>
                  <Route path="/task" element={<FailureActions />} />
                  <Route
                    path="/org/workspaces/sp/automations/:appId/runs/:runId?"
                    element={<FactoryAutomationRunPage />}
                  />
                </Routes>
              </MemoryRouter>
            </TooltipProvider>
          </ThemeProvider>
        </QueryClientProvider>,
      );
      const links = screen.getAllByRole("link", { name: "View run details" });
      const link =
        source === "summary"
          ? within(screen.getByTestId("summary")).getByRole("link", { name: "View run details" })
          : links[1];
      expect(link).toHaveAttribute(
        "href",
        source === "step"
          ? `${runHref}&activity=attach`
          : "/org/workspaces/sp/automations/app/runs/run?from=task&orderNumber=106",
      );
      fireEvent.click(link);
      await screen.findByTestId("factory-automation-run-page");
      await screen.findByRole("heading", { name: source === "step" ? "Attach PR" : "Earlier step" });
      expect(screen.queryByTestId("factory-app-canvas-page")).not.toBeInTheDocument();
      expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: /Implement task/ }));
      await screen.findByText("Saved runner log");
      expect(screen.getByTestId("location")).toHaveTextContent("activity=agent");
      fireEvent.click(screen.getByRole("button", { name: "Browser back" }));
      await screen.findByRole("heading", { name: source === "step" ? "Attach PR" : "Earlier step" });
      fireEvent.click(screen.getByRole("button", { name: "Run history" }));
      await screen.findByRole("table", { name: "Runs" });
      expect(screen.getByRole("link", { name: "Back" })).toHaveAttribute("href", "/org/workspaces/sp/task/106");
      expect(screen.getByTestId("location").textContent).not.toContain("activity=");
      fireEvent.click(screen.getByRole("button", { name: "Browser back" }));
      await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/runs/run"));
    } finally {
      client.clear();
    }
  });
});
