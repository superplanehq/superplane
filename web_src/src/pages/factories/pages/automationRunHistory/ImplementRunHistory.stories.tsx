import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider, useQueryClient } from "@tanstack/react-query";
import { delay, http, HttpResponse } from "msw";
import { useMemo, useRef, useState } from "react";
import { MemoryRouter } from "react-router";
import type { CanvasesCanvasRun, CanvasesCanvasNodeExecution } from "@/api-client";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { FactoryAutomationRunHistory } from "./FactoryAutomationRunHistory";
import { useAutomationRunNavigation } from "../useAutomationRunNavigation";
import { canvasKeys } from "@/hooks/useCanvasData";
import { MergeConfidenceConfigShell } from "../MergeConfidenceConfigShell";
import { DEFAULT_SPLIT_CONFIG_DELETE_COPY } from "../splitConfigDeleteCopy";
import { prepareData } from "@/pages/app/workflowPageHelpers";

const canvasId = "storybook-implement";
const runId = "f13d08a6-7b41-4d2e-9c83-6a0295dce108";
const execution: CanvasesCanvasNodeExecution = {
  id: "implementation-execution",
  nodeId: "implementation-agent-no-issue",
  runId,
  canvasId,
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  resultMessage: "Agent process exited with code 1.",
  createdAt: "2026-10-19T14:06:02Z",
  updatedAt: "2026-10-19T14:18:42Z",
  metadata: { runner_broker_task_id: "runner-task" },
  outputs: { failed: [{ message: "MCP connection failed" }] },
};
const run: CanvasesCanvasRun = {
  id: runId,
  canvasId,
  state: "STATE_FINISHED",
  result: "RESULT_FAILED",
  versionId: "historical-version",
  createdAt: "2026-10-19T14:06:00Z",
  finishedAt: "2026-10-19T14:18:42Z",
  errors: ["Agent process exited with code 1."],
  executions: [execution],
  rootEvent: { id: "start-event", data: { taskId: "task-1" } },
};
const olderRun = {
  ...run,
  id: "a8c19e42-7b41-4d2e-9c83-6a0295dce108",
  result: "RESULT_PASSED" as const,
  errors: [],
  executions: [],
  rootEvent: { id: "older-event" },
};
const api = `/api/v1/canvases/${canvasId}`;
const historyRuns = Array.from({ length: 50 }, (_, index) => ({
  ...(index === 0 ? run : olderRun),
  id: index === 0 ? runId : `${index.toString(16).padStart(8, "0")}-7b41-4d2e-9c83-6a0295dce108`,
  createdAt: new Date(Date.parse(run.createdAt!) - index * 30 * 60 * 1000).toISOString(),
  task: {
    id: `task-${index + 1}`,
    key: `SUPER-${971 - index}`,
    title: index === 0 ? "Keep MCP tool timeouts as retries" : "Use the default branch when starting a task",
  },
}));
const orders = historyRuns.map((item, index) => ({
  id: `task-${index + 1}`,
  key: `SUPER-${971 - index}`,
  title: index === 0 ? "Keep MCP tool timeouts as retries" : "Use the default branch when starting a task",
  sourceRunId: item.id,
}));
function runsPage(request: Request) {
  const offset = new URL(request.url).searchParams.has("before") ? 25 : 0;
  const runs = historyRuns.slice(offset, offset + 25);
  return HttpResponse.json({ runs, totalCount: historyRuns.length, lastTimestamp: runs.at(-1)?.createdAt });
}
const handlers = [
  http.get("/api/v1/organizations/storybook-org", () =>
    HttpResponse.json({
      organization: {
        metadata: { id: "storybook-org", name: "SuperPlane Prod" },
        spec: { enabledExperimentalFeatures: ["factory_run_history"] },
      },
    }),
  ),
  http.get("/account/experimental-features", () =>
    HttpResponse.json({ features: [{ id: "factory_run_history", label: "Factory Run History", released: false }] }),
  ),
  http.get(`${api}/runs`, ({ request }) => runsPage(request)),
  http.get(`${api}/runs/:runId`, ({ params }) =>
    HttpResponse.json({ run: historyRuns.find((item) => item.id === params.runId) }),
  ),
  http.get(`${api}/events/:eventId/executions`, ({ params }) =>
    HttpResponse.json({ executions: params.eventId === "start-event" ? [execution] : [] }),
  ),
  http.get(`${api}/versions/historical-version`, () =>
    HttpResponse.json({
      version: { spec: { nodes: [{ id: execution.nodeId, name: "Implement task", component: "runnerClaudeCode" }] } },
    }),
  ),
  http.get("/api/v1/factories/storybook-factory/orders", () => HttpResponse.json({ orders, hasNextPage: false })),
  http.get(`${api}/node-executions/${execution.id}/runner-live-logs/session`, () =>
    HttpResponse.json({ backend: "integrated", stream_url: `${api}/runner-logs` }),
  ),
  http.get(
    `${api}/runner-logs`,
    () =>
      new HttpResponse(
        '{"type":"line","text":"Preparing repository…"}\n{"type":"line","text":"Error: could not list MCP tools"}\n{"type":"line","text":"Agent process exited with code 1."}\n',
        { headers: { "Content-Type": "application/x-ndjson", "X-SuperPlane-Log-State": "archived" } },
      ),
  ),
];

function ConnectedPreview({
  initialRun = false,
  cachedExecutions = false,
  navigation = false,
}: {
  initialRun?: boolean;
  cachedExecutions?: boolean;
  navigation?: boolean;
}) {
  const [client] = useState(() => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    if (cachedExecutions)
      queryClient.setQueryData(canvasKeys.eventExecution(canvasId, "start-event"), { executions: [] });
    return queryClient;
  });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter
        initialEntries={[
          `/storybook?automationView=${canvasId}&automationTab=runs${initialRun ? `&automationRun=${runId}` : ""}`,
        ]}
      >
        {navigation ? <ConnectedAutomationNavigation /> : <ConnectedRunPage />}
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function ConnectedAutomationNavigation() {
  const { tab, setTab, selectedRunId, setSelectedRunId } = useAutomationRunNavigation();
  const queryClient = useQueryClient();
  const containerRef = useRef<HTMLDivElement>(null);
  const graph = useMemo(() => {
    const specNodes = [
      { id: "onrun-implement", name: "Start implementation", type: "TYPE_TRIGGER" as const, component: "onRun" },
      {
        id: "implementation-agent-no-issue",
        name: "Implement task",
        type: "TYPE_ACTION" as const,
        component: "runnerClaudeCode",
      },
    ];
    const prepared = prepareData({
      workflow: {
        metadata: { id: canvasId },
        spec: {
          nodes: specNodes,
          edges: [{ sourceId: "onrun-implement", targetId: "implementation-agent-no-issue", channel: "default" }],
        },
      },
      triggers: [{ name: "onRun", label: "On Run" }],
      components: [{ name: "runnerClaudeCode", label: "Run Claude Code" }],
      nodeEventsMap: {},
      nodeExecutionsMap: {},
      nodeQueueItemsMap: {},
      workflowId: canvasId,
      queryClient,
      user: null,
      canvasMode: "live",
    });
    return {
      nodes: prepared.nodes,
      edges: prepared.edges,
      specNodes,
      organizationId: "storybook-org",
      factoryId: "storybook-factory",
    };
  }, [queryClient]);
  return (
    <main className="h-screen bg-background text-foreground">
      <MergeConfidenceConfigShell
        title="Implement"
        page
        onClose={() => setTab("automation")}
        deleteCopy={DEFAULT_SPLIT_CONFIG_DELETE_COPY}
        deletePending={false}
        confirmDelete={false}
        onConfirmDelete={() => {}}
        onConfirmDeleteChange={() => {}}
        tab={tab}
        showRunsTab
        onTabChange={setTab}
        containerRef={containerRef}
        organizationId="storybook-org"
        factoryId="storybook-factory"
        widthPercent={50}
        isResizing={false}
        onResizeStart={() => {}}
        onSaveNode={() => {}}
        graph={graph}
        loading={false}
        error={false}
        canvasId={canvasId}
        selectedRunId={selectedRunId}
        onSelectedRunIdChange={setSelectedRunId}
        onNodeSelect={() => {}}
        selectedNodeId={null}
        focusNonce={0}
        layoutFitNonce={null}
      />
    </main>
  );
}
function ConnectedRunPage() {
  const { selectedRunId, setSelectedRunId: onSelectRun } = useAutomationRunNavigation();
  return (
    <main className="flex h-screen flex-col bg-background text-foreground">
      <header className="border-b p-6">
        <h1 className="text-xl font-semibold">Implement</h1>
        <p className="mt-3 text-sm">Runs</p>
      </header>
      <FactoryAutomationRunHistory
        canvasId={canvasId}
        organizationId="storybook-org"
        factoryId="storybook-factory"
        selectedRunId={selectedRunId}
        onSelectRun={onSelectRun}
      />
    </main>
  );
}
const meta = {
  title: "Factories/Pages/Automation run history/Connected Implement",
  component: ConnectedPreview,
  decorators: [withFactoriesTheme],
  parameters: { layout: "fullscreen", msw: { handlers } },
} satisfies Meta<typeof ConnectedPreview>;
export default meta;
type Story = StoryObj<typeof meta>;
export const HistoryAndArchivedLogs: Story = {};
export const UnavailableLogs: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get(`${api}/runner-logs`, () => new HttpResponse("Saved logs are unavailable", { status: 404 })),
        ...handlers,
      ],
    },
  },
};

export const SlowTaskLinks: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/api/v1/factories/storybook-factory/orders", async () => {
          await delay(3000);
          return HttpResponse.json({ orders, hasNextPage: false });
        }),
        ...handlers,
      ],
    },
  },
};
export const SlowPagination: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get(`${api}/runs`, async ({ request }) => {
          if (new URL(request.url).searchParams.has("before")) await delay(2500);
          return runsPage(request);
        }),
        ...handlers,
      ],
    },
  },
};
export const PaginationError: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get(`${api}/runs`, ({ request }) =>
          new URL(request.url).searchParams.has("before") ? new HttpResponse(null, { status: 500 }) : runsPage(request),
        ),
        ...handlers,
      ],
    },
  },
};
export const RunPermalink: Story = { args: { initialRun: true } };
export const AutomationNavigation: Story = { args: { navigation: true } };
export const LoadingExecutionMetadata: Story = {
  args: { initialRun: true, cachedExecutions: true },
  parameters: {
    msw: {
      handlers: [
        http.get(`${api}/events/:eventId/executions`, async () => {
          await delay(2500);
          return HttpResponse.json({ executions: [execution] });
        }),
        ...handlers,
      ],
    },
  },
};
export const LoadingRunnerLogs: Story = {
  args: { initialRun: true },
  parameters: {
    msw: {
      handlers: [
        http.get(
          `${api}/runner-logs`,
          () =>
            new HttpResponse(null, {
              headers: {
                "X-SuperPlane-Log-State": "archived",
                "X-SuperPlane-Log-URL": `${api}/archived-runner-logs`,
              },
            }),
        ),
        http.get(`${api}/archived-runner-logs`, async () => {
          await delay(2500);
          return new HttpResponse('{"type":"line","text":"Archived runner output loaded."}\n', {
            headers: { "Content-Type": "application/x-ndjson", "X-SuperPlane-Log-State": "archived" },
          });
        }),
        ...handlers,
      ],
    },
  },
};
