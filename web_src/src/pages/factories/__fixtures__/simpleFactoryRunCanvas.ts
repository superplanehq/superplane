import { parseCanvasYamlToSpec } from "@/pages/app/lib/canvas-yaml-staging";

import simpleFactoryRunCanvasYaml from "./simpleFactoryRunCanvas.yaml?raw";

const FIVE_HOURS_MS = 5 * 60 * 60 * 1000;

/** Footer timestamps on the inspected Line-run canvas (matches the compact CI graph). */
export const SIMPLE_FACTORY_RUN_EVENT_AT = new Date(Date.now() - FIVE_HOURS_MS).toISOString();

export const SIMPLE_FACTORY_RUN_NODE_IDS = ["on-run", "loop", "mark-pr-ready", "run-workflow", "run-claude"] as const;

export const SIMPLE_FACTORY_RUN_EDGES = [
  { sourceId: "on-run", targetId: "loop", channel: "default" },
  { sourceId: "loop", targetId: "mark-pr-ready", channel: "done" },
  { sourceId: "loop", targetId: "run-workflow", channel: "next" },
  { sourceId: "run-workflow", targetId: "loop", channel: "passed" },
  { sourceId: "run-workflow", targetId: "run-claude", channel: "failed" },
  { sourceId: "run-claude", targetId: "loop", channel: "passed" },
  { sourceId: "run-claude", targetId: "loop", channel: "failed" },
] as const;

export function simpleFactoryRunCanvasSpec() {
  const spec = parseCanvasYamlToSpec(simpleFactoryRunCanvasYaml);
  if (!spec?.nodes?.length) {
    throw new Error("simpleFactoryRunCanvas.yaml did not parse to a canvas spec");
  }
  return spec;
}

function outputChannels(...names: string[]) {
  return names.map((name) => ({ name, label: "", description: "" }));
}

/** Catalog entries the captured Software Factory fixture does not include. */
export function simpleFactoryRunCatalogActions() {
  return [
    {
      name: "github.markPullRequestReadyForReview",
      label: "Mark Pull Request Ready for Review",
      description: "Mark a draft pull request ready for review",
      icon: "github",
      color: "gray",
      outputChannels: outputChannels("default"),
    },
    {
      name: "semaphore.runWorkflow",
      label: "Run Workflow",
      description: "Run a Semaphore workflow",
      icon: "workflow",
      color: "gray",
      outputChannels: outputChannels("passed", "failed"),
    },
    {
      name: "runnerClaudeCode",
      label: "Run Claude Code",
      description: "Run Claude Code on a fleet runner",
      icon: "code",
      color: "gray",
      outputChannels: outputChannels("passed", "failed"),
    },
  ];
}

export function mergeSimpleFactoryRunActions(baseActions: unknown): { actions: unknown[] } {
  const existing = (baseActions as { actions?: unknown[] } | undefined)?.actions ?? [];
  return { actions: [...existing, ...simpleFactoryRunCatalogActions()] };
}

function passedExecution(id: string, nodeId: string, extras: Record<string, unknown> = {}) {
  return {
    id,
    nodeId,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: SIMPLE_FACTORY_RUN_EVENT_AT,
    updatedAt: SIMPLE_FACTORY_RUN_EVENT_AT,
    ...extras,
  };
}

/**
 * Executions for the compact CI graph. Run Claude Code is omitted so a finished
 * run shows Did not run on that node.
 */
export function simpleFactoryRunExecutions() {
  return [
    passedExecution("exec-loop", "loop", {
      outputs: {
        done: [{ timestamp: SIMPLE_FACTORY_RUN_EVENT_AT }],
        next: [{ timestamp: SIMPLE_FACTORY_RUN_EVENT_AT }],
      },
    }),
    passedExecution("exec-mark-pr-ready", "mark-pr-ready"),
    passedExecution("exec-run-workflow", "run-workflow"),
  ];
}

export function simpleFactoryRunOnRunTrigger() {
  return {
    name: "onRun",
    label: "On Run",
    description: "Start a factory run",
    icon: "play",
    color: "gray",
    outputChannels: outputChannels("default"),
  };
}
