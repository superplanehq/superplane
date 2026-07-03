import { useRef, type ReactNode } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import type {
  CanvasesCanvasNodeExecution,
  CanvasesCanvasRun,
  CanvasesCanvasRunResult,
  CanvasesCanvasRunState,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { canvasKeys } from "@/hooks/useCanvasData";

export const RUNS_STORY_CANVAS_ID = "canvas-1";

export const TRIGGER_NODE_ID = "trigger-1";
export const NOTIFY_NODE_ID = "action-notify";
export const DEPLOY_NODE_ID = "action-deploy";

const minutesAgo = (minutes: number) => new Date(Date.now() - minutes * 60 * 1000).toISOString();
const secondsAfter = (iso: string, seconds: number) => new Date(new Date(iso).getTime() + seconds * 1000).toISOString();

type NodeState = "success" | "running" | "error";
type RunCategory = "passed" | "failed" | "running";

interface ActionNodeSpec {
  id: string;
  name: string;
  component: string;
}

const ACTION_NODE_SPECS: ActionNodeSpec[] = [
  { id: NOTIFY_NODE_ID, name: "Notify team", component: "notify" },
  { id: DEPLOY_NODE_ID, name: "Deploy to staging", component: "deploy" },
  { id: "action-test", name: "Run tests", component: "test" },
  { id: "action-build", name: "Build image", component: "build" },
  { id: "action-lint", name: "Lint code", component: "lint" },
  { id: "action-scan", name: "Security scan", component: "scan" },
  { id: "action-migrate", name: "Run migrations", component: "migrate" },
  { id: "action-approve", name: "Manual approval", component: "approval" },
  { id: "action-deploy-prod", name: "Deploy to production", component: "deploy" },
  { id: "action-smoke", name: "Smoke tests", component: "test" },
  { id: "action-slack", name: "Post to Slack", component: "notify" },
  { id: "action-cache", name: "Warm cache", component: "cache" },
  { id: "action-backup", name: "Backup database", component: "backup" },
  { id: "action-report", name: "Generate report", component: "report" },
  { id: "action-cleanup", name: "Cleanup artifacts", component: "cleanup" },
];

export const mockWorkflowNodes: ComponentsNode[] = [
  {
    id: TRIGGER_NODE_ID,
    name: "Push to main",
    type: "TYPE_TRIGGER",
    component: "github",
  },
  ...ACTION_NODE_SPECS.map<ComponentsNode>((spec) => ({
    id: spec.id,
    name: spec.name,
    type: "TYPE_ACTION",
    component: spec.component,
    configuration: { action: spec.component },
  })),
];

function pickActionSpecs(executionCount: number, offset: number): ActionNodeSpec[] {
  return Array.from(
    { length: executionCount },
    (_, index) => ACTION_NODE_SPECS[(offset + index) % ACTION_NODE_SPECS.length],
  );
}

function nodeStatesForRun(category: RunCategory, count: number): NodeState[] {
  const states: NodeState[] = Array.from({ length: count }, () => "success");
  if (count === 0) {
    return states;
  }

  // Each category's steps reflect its status: passed runs are all success, failed
  // runs carry a single errored step, and running runs have their latest step
  // still executing (plus any approval step left waiting - handled in buildRun).
  if (category === "failed") {
    states[Math.min(count - 1, Math.floor(count / 2))] = "error";
  }

  if (category === "running") {
    states[count - 1] = "running";
  }

  return states;
}

function buildExecution(config: {
  runId: string;
  spec: ActionNodeSpec;
  order: number;
  state: NodeState;
  createdAt: string;
  previousExecutionId?: string;
}): CanvasesCanvasNodeExecution {
  const { runId, spec, order, state, createdAt, previousExecutionId } = config;
  const base: CanvasesCanvasNodeExecution = {
    id: `exec-${runId}-${spec.id}`,
    canvasId: RUNS_STORY_CANVAS_ID,
    nodeId: spec.id,
    previousExecutionId,
    createdAt,
    updatedAt: createdAt,
    configuration: { action: spec.component },
    rootEvent: {
      id: `input-${runId}-${spec.id}`,
      canvasId: RUNS_STORY_CANVAS_ID,
      createdAt,
      data: {
        source: "push",
        ref: "refs/heads/main",
        params: { environment: "staging", service: spec.component },
      },
    },
  };

  if (state === "running") {
    return {
      ...base,
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      metadata: { phase: "in_progress", startedAt: createdAt },
    };
  }

  if (state === "error") {
    return {
      ...base,
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: `${spec.name} failed: exited with code 1`,
      outputs: {},
      metadata: { attempts: "3 / 3", exitCode: 1 },
    };
  }

  return {
    ...base,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    outputs: {
      statusCode: 200,
      durationMs: 1200 + order * 137,
      url: "https://staging.example.com",
    },
    metadata: { environment: "staging" },
  };
}

interface GeneratedRun {
  run: CanvasesCanvasRun;
  executions: CanvasesCanvasNodeExecution[];
}

function buildRun(config: {
  id: string;
  name: string;
  category: RunCategory;
  executionCount: number;
  createdMinutesAgo: number;
  offset: number;
}): GeneratedRun {
  const { id, name, category, executionCount, createdMinutesAgo, offset } = config;
  const createdAt = minutesAgo(createdMinutesAgo);
  const specs = pickActionSpecs(executionCount, offset);
  const states = nodeStatesForRun(category, executionCount);

  // A still-running run that routes through an approval step leaves that step in a
  // STATE_STARTED execution, which the approval mapper reports as "waiting".
  if (category === "running") {
    const approvalIndex = specs.findIndex((spec) => spec.component === "approval");
    if (approvalIndex !== -1) states[approvalIndex] = "running";
  }

  const executions: CanvasesCanvasNodeExecution[] = specs.map((spec, index) =>
    buildExecution({
      runId: id,
      spec,
      order: index,
      state: states[index],
      createdAt: secondsAfter(createdAt, (index + 1) * 45),
      previousExecutionId: index === 0 ? undefined : `exec-${id}-${specs[index - 1].id}`,
    }),
  );

  const state: CanvasesCanvasRunState = category === "running" ? "STATE_STARTED" : "STATE_FINISHED";
  const result: CanvasesCanvasRunResult =
    category === "running" ? "RESULT_UNKNOWN" : category === "failed" ? "RESULT_FAILED" : "RESULT_PASSED";

  const run: CanvasesCanvasRun = {
    id,
    canvasId: RUNS_STORY_CANVAS_ID,
    state,
    result,
    versionId: `v-${String((offset % 8) + 1).padStart(2, "0")}`,
    createdAt,
    finishedAt: category === "running" ? undefined : secondsAfter(createdAt, (executionCount + 1) * 45),
    rootEvent: {
      id: `event-${id}`,
      canvasId: RUNS_STORY_CANVAS_ID,
      nodeId: TRIGGER_NODE_ID,
      channel: "push",
      customName: name,
      createdAt,
      root: true,
      data: {
        ref: "refs/heads/main",
        commit: id.slice(-7),
        author: "octocat",
      },
    },
  };

  return { run, executions };
}

const RUN_NAMES = [
  "Deploy main",
  "Hotfix release",
  "Feature branch",
  "Nightly build",
  "Release candidate",
  "Dependency bump",
  "Config update",
  "Schema migration",
  "Rollback attempt",
  "Canary deploy",
  "Docs update",
  "Perf regression fix",
  "Security patch",
  "Infra change",
  "Data backfill",
  "Cache warmup",
  "Integration sync",
  "Cleanup job",
  "Smoke test run",
  "Manual trigger",
];

const CATEGORIES: RunCategory[] = ["passed", "failed", "running"];

const featuredRuns: GeneratedRun[] = [
  buildRun({
    id: "run-passed",
    name: "Deploy main",
    category: "passed",
    executionCount: 7,
    createdMinutesAgo: 8,
    offset: 0,
  }),
  buildRun({
    id: "run-failed",
    name: "Hotfix release",
    category: "failed",
    executionCount: 5,
    createdMinutesAgo: 32,
    offset: 2,
  }),
  buildRun({
    id: "run-running",
    name: "Feature branch",
    category: "running",
    executionCount: 8,
    createdMinutesAgo: 1,
    offset: 1,
  }),
];

const generatedRuns: GeneratedRun[] = Array.from({ length: 17 }, (_, index) => {
  const runNumber = index + 4;
  return buildRun({
    id: `run-${String(runNumber).padStart(2, "0")}`,
    name: RUN_NAMES[(index + 3) % RUN_NAMES.length],
    category: CATEGORIES[index % CATEGORIES.length],
    executionCount: 2 + (index % 13),
    createdMinutesAgo: 12 + index * 17,
    offset: (index * 3) % ACTION_NODE_SPECS.length,
  });
});

const allRuns: GeneratedRun[] = [...featuredRuns, ...generatedRuns];

export const mockPassedRun = featuredRuns[0].run;
export const mockPassedExecutions = featuredRuns[0].executions;
export const mockFailedRun = featuredRuns[1].run;
export const mockRunningRun = featuredRuns[2].run;

export const mockRuns: CanvasesCanvasRun[] = allRuns.map((generated) => generated.run);

export function getRunExecutions(runId: string): CanvasesCanvasNodeExecution[] {
  return allRuns.find((generated) => generated.run.id === runId)?.executions ?? [];
}

export function seedRunExecutionsCache(queryClient: QueryClient) {
  for (const { run, executions } of allRuns) {
    queryClient.setQueryData(canvasKeys.eventExecution(RUNS_STORY_CANVAS_ID, run.rootEvent!.id!), { executions });
  }
}

export function RunsStorySeed({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const hasSeeded = useRef(false);

  if (!hasSeeded.current) {
    seedRunExecutionsCache(queryClient);
    hasSeeded.current = true;
  }

  return <>{children}</>;
}
