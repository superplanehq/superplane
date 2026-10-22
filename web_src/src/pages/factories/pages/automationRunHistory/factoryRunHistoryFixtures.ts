import { http, HttpResponse } from "msw";
import type { CanvasesCanvas, CanvasesCanvasNodeExecution, CanvasesCanvasRun } from "@/api-client";
import { HOSTED_MODEL_ALL_PROVIDERS } from "@/lib/hostedLLMModels";
import { storybookSelectableLlmModels } from "@/pages/home/__fixtures__/hostedLlmModels";
import { PLANNING_REVIEW_RUNNER_FIELDS } from "../planningReviewRunnerFields";

export const factoryHistoryStoryOrganization = "backlog-history-org";
export const factoryHistoryStoryFactory = "backlog-history-factory";
export const factoryHistoryStoryRunId = "de79e14a-5c4a-4ce9-b283-cdb01acbb923";
export const factoryHistoryStorySources = [
  "github-issues",
  "sentry-exceptions",
  "dependabot-alerts",
  "planning",
] as const;
export type BacklogStorySource = (typeof factoryHistoryStorySources)[number];

export type VerificationStorySource =
  | "discussion"
  | "checks"
  | "merge-confidence"
  | "visual-evidence"
  | "pr-closure"
  | "custom-verify"
  | "custom-done";
export type FactoryHistoryStorySource = BacklogStorySource | VerificationStorySource;

export const factoryHistoryStoryNames: Record<FactoryHistoryStorySource, string> = {
  "github-issues": "GitHub issues",
  "sentry-exceptions": "Sentry exceptions",
  "dependabot-alerts": "Dependabot alerts",
  planning: "Planning",
  discussion: "Address PR feedback",
  checks: "Fix pull request checks",
  "merge-confidence": "Merge confidence",
  "visual-evidence": "Visual Evidence",
  "pr-closure": "PR Closure",
  "custom-verify": "Deploy preview",
  "custom-done": "Notify release team",
};
const taskTitles: Record<FactoryHistoryStorySource, string> = {
  "github-issues": "Keep MCP tool timeouts as retries",
  "sentry-exceptions": "Handle missing project settings",
  "dependabot-alerts": "Update the vulnerable vite dependency",
  planning: "Keep task links while loading run history",
  discussion: "Preserve keyboard focus after saving a task",
  checks: "Handle network timeouts during checkout",
  "merge-confidence": "Validate webhooks before processing events",
  "visual-evidence": "Use compact rows in the task list",
  "pr-closure": "Restore task links in intake run history",
  "custom-verify": "Preview the new settings screen",
  "custom-done": "Publish the completed release",
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

const verificationSteps: Partial<Record<FactoryHistoryStorySource, { trigger: string; id: string; name: string }>> = {
  discussion: { trigger: "on-comment", id: "address-feedback", name: "Address PR feedback" },
  checks: { trigger: "on-checks", id: "fix-checks", name: "Fix failing checks" },
  "merge-confidence": { trigger: "on-pr-risk", id: "assess-risk", name: "Assess Merge Confidence" },
  "visual-evidence": {
    trigger: "on-pr-visual-evidence",
    id: "capture-visual-evidence",
    name: "Capture Visual Evidence",
  },
  "pr-closure": { trigger: "on-pr-closed", id: "complete-work-order", name: "Complete task" },
  "custom-verify": { trigger: "on-preview", id: "deploy-preview", name: "Deploy preview" },
  "custom-done": { trigger: "on-completed", id: "notify-team", name: "Notify release team" },
};
function storyStep(source: FactoryHistoryStorySource) {
  return (
    verificationSteps[source] ?? {
      trigger: "trigger",
      id: "refine-task",
      name: source === "planning" ? "Plan task" : "Evaluate incoming task",
    }
  );
}

export function factoryHistoryStoryCanvasId(source: FactoryHistoryStorySource) {
  return `history-${source}`;
}

export function factoryHistoryStoryCanvas(source: FactoryHistoryStorySource): CanvasesCanvas {
  const step = storyStep(source);
  return {
    metadata: {
      id: factoryHistoryStoryCanvasId(source),
      name: factoryHistoryStoryNames[source],
      factoryId: factoryHistoryStoryFactory,
    },
    spec: {
      nodes: [
        {
          id: step.trigger,
          name: verificationSteps[source] ? "On Pull Request" : "Task received",
          type: "TYPE_TRIGGER",
          component: "onRun",
        },
        {
          id: step.id,
          name: step.name,
          type: "TYPE_ACTION",
          component:
            source === "pr-closure" ? "updateWorkOrderStatus" : source === "custom-done" ? "http" : "runnerSuperPlane",
          configuration: {
            model: "",
            executionTimeoutSeconds: 3600,
            steps: [{ name: "Evaluate task", type: "prompt", prompt: "Evaluate this task using repository context." }],
          },
          concurrency: { max: 3 },
        },
      ],
      edges: [{ sourceId: step.trigger, targetId: step.id, channel: "default" }],
    },
  };
}

function storyRun(source: FactoryHistoryStorySource): CanvasesCanvasRun {
  const isFeedback = source === "discussion" || source === "checks";
  return {
    id: factoryHistoryStoryRunId,
    canvasId: factoryHistoryStoryCanvasId(source),
    versionId: "history-version",
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: new Date(Date.now() - 125_000).toISOString(),
    finishedAt: new Date(Date.now() - 5_000).toISOString(),
    rootEvent: {
      id: "history-event",
      customName: isFeedback ? "Pull request activity received" : taskTitles[source],
      data: isFeedback
        ? { pull_request: { number: 182 }, repository: { full_name: "superplanehq/superplane" } }
        : source === "planning" || verificationSteps[source]
          ? { workOrder: { id: "task-1" } }
          : { source, title: taskTitles[source] },
    },
    executions: [
      { id: "history-execution", nodeId: storyStep(source).id, state: "STATE_FINISHED", result: "RESULT_PASSED" },
    ],
  };
}

export function factoryHistoryHandlers(source: FactoryHistoryStorySource) {
  const canvas = factoryHistoryStoryCanvas(source);
  const run = storyRun(source);
  const execution: CanvasesCanvasNodeExecution = {
    ...run.executions![0],
    runId: run.id,
    createdAt: new Date(Date.parse(run.createdAt!) + 1000).toISOString(),
    updatedAt: run.finishedAt,
    metadata:
      source === "pr-closure" || source === "custom-done" ? {} : { runner_broker_task_id: "history-runner-task" },
    outputs: { default: [{ accepted: true, taskId: "task-1" }] },
  };
  const api = `/api/v1/canvases/${factoryHistoryStoryCanvasId(source)}`;
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
    http.get(`/api/v1/organizations/${factoryHistoryStoryOrganization}`, () =>
      HttpResponse.json({
        organization: {
          metadata: { id: factoryHistoryStoryOrganization, name: "SuperPlane" },
          spec: { enabledExperimentalFeatures: [] },
        },
      }),
    ),
    http.get("/account/experimental-features", () => HttpResponse.json({ features: [] })),
    http.get("/api/v1/actions", () =>
      HttpResponse.json({
        actions: [
          action,
          { name: "updateWorkOrderStatus", label: "Update task status", configuration: [] },
          { name: "http", label: "Send HTTP request", configuration: [] },
        ],
      }),
    ),
    http.get(`/api/v1/actions/${action.name}`, () => HttpResponse.json({ action })),
    http.get("/api/v1/triggers", () => HttpResponse.json({ triggers: [{ name: "onRun", label: "On Run" }] })),
    http.get("/api/v1/widgets", () => HttpResponse.json({ widgets: [] })),
    http.get("/api/v1/integrations", () => HttpResponse.json({ integrations: [] })),
    http.get(`/api/v1/organizations/${factoryHistoryStoryOrganization}/selectable-llm-models`, () =>
      HttpResponse.json({ models: storybookSelectableLlmModels() }),
    ),
    http.get(`/api/v1/organizations/${factoryHistoryStoryOrganization}/workspace-usage`, () =>
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
    http.get(`/api/v1/factories/${factoryHistoryStoryFactory}/orders`, () =>
      HttpResponse.json({
        orders: [
          {
            id: "task-1",
            key: "SUPER-982",
            title: taskTitles[source],
            sourceRunId: source === "planning" || verificationSteps[source] ? undefined : run.id,
            pullRequests:
              source === "discussion" || source === "checks"
                ? [{ id: "pull-request-1", activities: [{ run: { id: run.id } }] }]
                : [],
          },
        ],
        hasNextPage: false,
      }),
    ),
    http.get(`/api/v1/organizations/${factoryHistoryStoryOrganization}/integrations/:integrationId/resources`, () =>
      HttpResponse.json({ resources: [{ id: "payments", name: "Payments" }] }),
    ),
    http.patch(`/api/v1/factories/${factoryHistoryStoryFactory}/intakes/:intakeId`, () => HttpResponse.json({})),
  ];
}
