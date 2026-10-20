import { http, HttpResponse } from "msw";
import type { CanvasesCanvas, CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
import { storybookSelectableLlmModels } from "@/pages/home/__fixtures__/hostedLlmModels";
import { PLANNING_REVIEW_RUNNER_FIELDS } from "../planningReviewRunnerFields";

export const backlogStoryOrganization = "backlog-history-org";
export const backlogStoryFactory = "backlog-history-factory";
export const backlogStoryRunId = "de79e14a-5c4a-4ce9-b283-cdb01acbb923";
export const backlogStorySources = ["github-issues", "sentry-exceptions", "dependabot-alerts", "planning"] as const;
export type BacklogStorySource = (typeof backlogStorySources)[number];

export const backlogStoryNames: Record<BacklogStorySource, string> = {
  "github-issues": "GitHub issues",
  "sentry-exceptions": "Sentry exceptions",
  "dependabot-alerts": "Dependabot alerts",
  planning: "Planning",
};
const taskTitles: Record<BacklogStorySource, string> = {
  "github-issues": "Keep MCP tool timeouts as retries",
  "sentry-exceptions": "Handle missing project settings",
  "dependabot-alerts": "Update the vulnerable vite dependency",
  planning: "Keep task links while loading run history",
};
const action = {
  name: "runnerSuperPlane",
  label: "Run SuperPlane Agent",
  configuration: PLANNING_REVIEW_RUNNER_FIELDS.filter((field) => field.name !== "credentials").map((field) =>
    field.name === "model"
      ? { ...field, type: "hosted-model", typeOptions: { hostedModel: { provider: HOSTED_MODEL_ALL_PROVIDERS } } }
      : field,
  ),
};

export function backlogStoryCanvasId(source: BacklogStorySource) {
  return `backlog-${source}`;
}

export function backlogStoryCanvas(source: BacklogStorySource): CanvasesCanvas {
  return {
    metadata: { id: backlogStoryCanvasId(source), name: backlogStoryNames[source], factoryId: backlogStoryFactory },
    spec: {
      nodes: [
        { id: "trigger", name: "Task received", type: "TYPE_TRIGGER", component: "onRun" },
        {
          id: "refine-task",
          name: source === "planning" ? "Plan task" : "Evaluate incoming task",
          type: "TYPE_ACTION",
          component: "runnerSuperPlane",
          configuration: {
            model: "",
            executionTimeoutSeconds: 3600,
            steps: [{ name: "Evaluate task", type: "prompt", prompt: "Evaluate this task using repository context." }],
          },
          concurrency: { max: 3 },
        },
      ],
      edges: [{ sourceId: "trigger", targetId: "refine-task", channel: "default" }],
    },
  };
}

function storyRun(source: BacklogStorySource): CanvasesCanvasRun {
  return {
    id: backlogStoryRunId,
    canvasId: backlogStoryCanvasId(source),
    versionId: "history-version",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: new Date(Date.now() - 125_000).toISOString(),
    finishedAt: new Date(Date.now() - 5_000).toISOString(),
    rootEvent: {
      id: "history-event",
      customName: taskTitles[source],
      data: source === "planning" ? { workOrder: { id: "task-1" } } : { source, title: taskTitles[source] },
    },
    executions: [{ id: "history-execution", nodeId: "refine-task", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
  };
}

export function backlogHistoryHandlers(source: BacklogStorySource, enabled = true) {
  const canvas = backlogStoryCanvas(source);
  const run = storyRun(source);
  const execution: CanvasesCanvasNodeExecution = {
    ...run.executions![0],
    runId: run.id,
    createdAt: new Date(Date.parse(run.createdAt!) + 1000).toISOString(),
    updatedAt: run.finishedAt,
    metadata: { runner_broker_task_id: "history-runner-task" },
    outputs: { default: [{ accepted: true, taskId: "task-1" }] },
  };
  const api = `/api/v1/canvases/${backlogStoryCanvasId(source)}`;
  const rejectedRun: CanvasesCanvasRun = {
    ...run,
    id: "24bb170c-fd98-41d2-b9b5-13057c247b2a",
    result: "RESULT_FAILED",
    rootEvent: { id: "rejected-event", data: { source } },
    executions: [],
    errors: ["Could not evaluate the incoming event."],
  };
  const runs = [run, rejectedRun];
  return [
    http.get(`/api/v1/organizations/${backlogStoryOrganization}`, () =>
      HttpResponse.json({
        organization: {
          metadata: { id: backlogStoryOrganization, name: "SuperPlane" },
          spec: { enabledExperimentalFeatures: enabled ? ["factory_run_history"] : [] },
        },
      }),
    ),
    http.get("/account/experimental-features", () =>
      HttpResponse.json({ features: [{ id: "factory_run_history", released: false }] }),
    ),
    http.get("/api/v1/actions", () => HttpResponse.json({ actions: [action] })),
    http.get(`/api/v1/actions/${action.name}`, () => HttpResponse.json({ action })),
    http.get("/api/v1/triggers", () => HttpResponse.json({ triggers: [{ name: "onRun", label: "On Run" }] })),
    http.get("/api/v1/widgets", () => HttpResponse.json({ widgets: [] })),
    http.get("/api/v1/integrations", () => HttpResponse.json({ integrations: [] })),
    http.get(`/api/v1/organizations/${backlogStoryOrganization}/selectable-llm-models`, () =>
      HttpResponse.json({ models: storybookSelectableLlmModels() }),
    ),
    http.get(`/api/v1/organizations/${backlogStoryOrganization}/workspace-usage`, () =>
      HttpResponse.json({ defaultHostedProvider: "openrouter", defaultHostedModel: "anthropic/claude-sonnet-4-6" }),
    ),
    http.get(api, () => HttpResponse.json({ canvas })),
    http.get(`${api}/staging`, () => HttpResponse.json({})),
    http.get(`${api}/runs`, () =>
      HttpResponse.json({
        runs,
        totalCount: 2,
      }),
    ),
    http.get(`${api}/runs/:runId`, ({ params }) =>
      HttpResponse.json({ run: runs.find((item) => item.id === params.runId) }),
    ),
    http.get(`${api}/events/:eventId/executions`, ({ params }) =>
      HttpResponse.json({ executions: params.eventId === "history-event" ? [execution] : [] }),
    ),
    http.get(`${api}/versions/history-version`, () => HttpResponse.json({ version: { spec: canvas.spec } })),
    http.get(`${api}/node-executions/history-execution/runner-live-logs/session`, () =>
      HttpResponse.json({ backend: "integrated", stream_url: `${api}/saved-output` }),
    ),
    http.get(
      `${api}/saved-output`,
      () =>
        new HttpResponse(
          '{"type":"line","text":"Loaded repository context."}\n{"type":"line","text":"Task evaluation complete."}\n',
          { headers: { "Content-Type": "application/x-ndjson", "X-SuperPlane-Log-State": "archived" } },
        ),
    ),
    http.get(`/api/v1/factories/${backlogStoryFactory}/orders`, () =>
      HttpResponse.json({
        orders: [
          {
            id: "task-1",
            key: "SUPER-982",
            title: taskTitles[source],
            sourceRunId: source === "planning" ? undefined : run.id,
          },
        ],
        hasNextPage: false,
      }),
    ),
    http.get(`/api/v1/organizations/${backlogStoryOrganization}/integrations/:integrationId/resources`, () =>
      HttpResponse.json({ resources: [{ id: "payments", name: "Payments" }] }),
    ),
    http.patch(`/api/v1/factories/${backlogStoryFactory}/intakes/:intakeId`, () => HttpResponse.json({})),
  ];
}
