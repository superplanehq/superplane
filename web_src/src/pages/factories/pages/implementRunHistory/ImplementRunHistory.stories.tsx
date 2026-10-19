import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import type { CanvasesCanvasRun, CanvasesCanvasNodeExecution } from "@/api-client";
import { withFactoriesTheme } from "../../__fixtures__/factoriesStoryTheme";
import { ImplementRunHistory } from "./ImplementRunHistory";

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
const handlers = [
  http.get(`${api}/runs`, ({ request }) =>
    HttpResponse.json({
      runs: new URL(request.url).searchParams.has("before") ? [olderRun] : [run],
      totalCount: 2,
      lastTimestamp: run.createdAt,
    }),
  ),
  http.get(`${api}/runs/:runId`, ({ params }) => HttpResponse.json({ run: params.runId === runId ? run : olderRun })),
  http.get(`${api}/events/:eventId/executions`, ({ params }) =>
    HttpResponse.json({ executions: params.eventId === "start-event" ? [execution] : [] }),
  ),
  http.get(`${api}/versions/historical-version`, () =>
    HttpResponse.json({
      version: { spec: { nodes: [{ id: execution.nodeId, name: "Implement task", component: "runnerClaudeCode" }] } },
    }),
  ),
  http.get("/api/v1/factories/storybook-factory/orders", () =>
    HttpResponse.json({
      orders: [{ id: "task-1", key: "SP-8578", title: "Keep MCP tool timeouts as retries", sourceRunId: runId }],
      hasNextPage: false,
    }),
  ),
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

function ConnectedPreview() {
  const [client] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));
  const [selectedRunId, onSelectRun] = useState<string | null>(null);
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <main className="flex h-screen flex-col bg-background text-foreground">
          <header className="border-b p-6">
            <h1 className="text-xl font-semibold">Implement</h1>
            <p className="mt-3 text-sm">Runs</p>
          </header>
          <ImplementRunHistory
            canvasId={canvasId}
            organizationId="storybook-org"
            factoryId="storybook-factory"
            selectedRunId={selectedRunId}
            onSelectRun={onSelectRun}
          />
        </main>
      </MemoryRouter>
    </QueryClientProvider>
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
