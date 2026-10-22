import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { ColumnAutomationViewHost } from "./ColumnAutomationViewPopup";
import { MergeConfidenceConfigModal } from "./MergeConfidenceConfigModal";
import { PRFeedbackSettingsPopup } from "./PRFeedbackSettingsPopup";
import { resetHostMocks, useCanvas, useInfiniteCanvasRuns } from "./IntakeSettingsHost.spec.fixtures";
import type { PRFeedbackDraftSettings } from "./prFeedbackSettingsModel";

vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryWorkOrders: () => ({
    data: [{ id: "task", key: "SP-123", title: "Linked task", sourceRunId: "run-intake-1" }],
    isLoading: false,
  }),
}));
vi.mock("@/hooks/useCanvasWebsocket", () => ({ useCanvasRuntimeWebsocket: () => {} }));

const settings: PRFeedbackDraftSettings = {
  source: "discussion",
  name: "Address PR feedback",
  repository: "acme/payments",
  mention: "@superplane",
  ignoreBots: true,
  addressAllBots: false,
  allowedBots: [],
  checkNames: [],
  maximumAttempts: 3,
  runnerIntegrationIds: [],
};

function BrowserNavigation() {
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <>
      <output data-testid="location">{location.search}</output>
      <button onClick={() => navigate(-1)}>Browser back</button>
    </>
  );
}

function FeedbackPage() {
  useLocation();
  return (
    <PRFeedbackSettingsPopup
      organizationId="org"
      factoryId="factory"
      canvasId="canvas"
      settings={{ ...settings, source: "checks" }}
      healthy
      onSave={() => {}}
      onClose={() => {}}
    />
  );
}

function renderPage(view: React.ReactNode, path = "/") {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={[path]}>
        <ThemeProvider>
          <TooltipProvider>
            <BrowserNavigation />
            {view}
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Verify and Done run history", () => {
  beforeEach(() => {
    resetHostMocks();
  });

  it.each(["on-pr-risk", "on-pr-visual-evidence", "on-pr-closed"])("shows %s history by default", (nodeId) => {
    renderPage(
      <MergeConfidenceConfigModal
        title="Automation"
        canvasId="canvas"
        factoryId="factory"
        graph={{
          organizationId: "org",
          factoryId: "factory",
          nodes: [],
          edges: [],
          specNodes: [{ id: nodeId, type: "TYPE_TRIGGER" }],
        }}
        onClose={() => {}}
        onSaveNode={() => {}}
      />,
      "/?automationTab=runs",
    );
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByText("SP-123")).toBeInTheDocument();
    expect(screen.queryByTestId("merge-confidence-config-canvas")).not.toBeInTheDocument();
  });

  it.each(["discussion", "checks"] as const)("shares %s settings and run selection through the URL", async (source) => {
    const user = userEvent.setup();
    renderPage(
      <PRFeedbackSettingsPopup
        organizationId="org"
        settings={{ ...settings, source }}
        healthy
        automationGraph={{ organizationId: "org", factoryId: "factory", nodes: [], edges: [] }}
        canvasId="canvas"
        onSave={() => {}}
        onClose={() => {}}
      />,
      "/?prFeedback=1&keep=1",
    );
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    expect(screen.getByTestId("location")).toHaveTextContent("prFeedbackSettings=runs");
    await user.click(screen.getByRole("button", { name: "Open run run-intake-1" }));
    expect(screen.getByTestId("location")).toHaveTextContent("automationRun=run-intake-1");
    expect(screen.getByTestId("location")).toHaveTextContent("keep=1");
    await user.click(screen.getByRole("button", { name: "Browser back" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Browser back" }));
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
  });

  it.each(["verify", "done"])("shows custom %s history by default", async (columnKey) => {
    useCanvas.mockReturnValue({
      data: {
        metadata: { id: "custom", factoryId: "factory" },
        spec: {
          nodes: [{ id: "agent", type: "TYPE_ACTION", component: "runnerSuperPlane", configuration: { steps: [] } }],
          edges: [],
        },
      },
      isPending: false,
      isError: false,
      refetch: vi.fn(),
    });
    renderPage(
      <ColumnAutomationViewHost
        organizationId="org"
        factoryId="factory"
        factoryKey="SP"
        canvasId="custom"
        columnKey={columnKey}
        title="Custom automation"
        onClose={() => {}}
      />,
    );
    expect(screen.getByRole("table")).toBeInTheDocument();
    expect(screen.getByTestId("column-automation-view-edit")).toHaveAttribute(
      "href",
      "/org/workspaces/sp/automations/custom?configure=1&agent=1&from=lines",
    );
    expect(screen.queryByTestId("column-automation-view-canvas")).not.toBeInTheDocument();
  });

  it("shows scheduled Backlog history by default", () => {
    useCanvas.mockReturnValue({
      data: {
        metadata: { id: "scheduled", factoryId: "factory" },
        spec: {
          nodes: [
            {
              id: "schedule",
              type: "TYPE_TRIGGER",
              component: "schedule",
              configuration: { type: "days", hour: 9, minute: 0 },
            },
            { id: "scheduled-agent", type: "TYPE_ACTION", component: "runnerSuperPlane", configuration: { steps: [] } },
            {
              id: "has-proposal",
              type: "TYPE_ACTION",
              component: "if",
              configuration: { expression: '{{ $["Agent"].data.result.title != nil }}' },
            },
          ],
          edges: [],
        },
      },
      isPending: false,
      isError: false,
      refetch: vi.fn(),
    });
    useInfiniteCanvasRuns.mockReturnValue({
      data: {
        pages: [
          {
            runs: [
              {
                id: "scheduled-run",
                state: "STATE_FINISHED",
                result: "RESULT_PASSED",
                rootEvent: { id: "tick", data: { type: "schedule.tick", data: {} } },
                executions: [
                  {
                    id: "proposal",
                    nodeId: "has-proposal",
                    componentName: "if",
                    result: "RESULT_PASSED",
                    outputChannels: ["false"],
                  },
                ],
              },
            ],
          },
        ],
      },
      isLoading: false,
      isError: false,
      hasNextPage: false,
    });
    renderPage(
      <ColumnAutomationViewHost
        organizationId="org"
        factoryId="factory"
        factoryKey="SP"
        canvasId="scheduled"
        columnKey="backlog"
        title="Scheduled intake"
        initialTab="runs"
        onClose={() => {}}
      />,
      "/?automationTab=runs",
    );
    expect(screen.getByRole("table", { name: "Runs" })).toBeInTheDocument();
    expect(screen.getByText("No task proposed")).toBeInTheDocument();
    expect(screen.queryByTestId("column-automation-view-canvas")).not.toBeInTheDocument();
  });

  it("keeps the full editor available for custom automations without an Agent form", async () => {
    useCanvas.mockReturnValue({
      data: {
        metadata: { id: "custom", factoryId: "factory" },
        spec: { nodes: [{ id: "notify", type: "TYPE_ACTION", component: "http" }], edges: [] },
      },
      isPending: false,
      isError: false,
      refetch: vi.fn(),
    });
    renderPage(
      <ColumnAutomationViewHost
        organizationId="org"
        factoryId="factory"
        factoryKey="SP"
        lineId="line-1"
        canvasId="custom"
        columnKey="done"
        title="Notify team"
        onClose={() => {}}
      />,
    );
    expect(screen.queryByRole("tab", { name: "Agent" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Open run run-intake-1" }));
    expect(screen.getByTestId("column-automation-view-edit")).toHaveAttribute(
      "href",
      "/org/workspaces/sp/automations/custom?configure=1&agent=1&from=lines&lineId=line-1",
    );
  });

  it("keeps scheduled Backlog intake on its current view", () => {
    renderPage(
      <ColumnAutomationViewHost
        organizationId="org"
        factoryId="factory"
        factoryKey="SP"
        canvasId="custom"
        columnKey="backlog"
        title="Scheduled intake"
        onClose={() => {}}
      />,
    );
    expect(screen.getByTestId("column-automation-view-canvas")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("preserves unsaved PR feedback settings while changing URL tabs", async () => {
    const user = userEvent.setup();
    renderPage(<FeedbackPage />);
    const attempts = screen.getByRole("spinbutton");
    await user.clear(attempts);
    await user.type(attempts, "5");
    await user.click(screen.getByRole("tab", { name: "Runs" }));
    await user.click(screen.getByRole("tab", { name: "General" }));
    expect(screen.getByRole("spinbutton")).toHaveValue(5);
  });
});
