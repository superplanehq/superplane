import type { CanvasesCanvas, CanvasesCanvasRun } from "@/api-client";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import { http, HttpResponse } from "msw";

import { stageFromPhase } from "./redesign/automationsViewModel";
import type { SplitRunPhase } from "./splitRunMocks";

const SECOND = 1_000;
const MINUTE = 60 * SECOND;
const LIFECYCLE_MS = 28 * MINUTE + 20 * SECOND;
const COMMANDS_MS = 5 * MINUTE + 30 * SECOND;

function createCommandLogRecords(checkoutMs: number, waitMs: number, finishedMs?: number) {
  const entries: Record<string, unknown>[] = [
    { type: "cmd_start", index: 0, kind: "bash", text: "Checkout", started_at: checkoutMs },
    { type: "cmd_end", index: 0, status: "passed", duration_ms: SECOND },
    { type: "cmd_start", index: 1, kind: "prompt", text: "Prompt", started_at: checkoutMs + SECOND },
    { type: "cmd_end", index: 1, status: "passed", duration_ms: 29 * SECOND },
    { type: "cmd_start", index: 2, kind: "bash", text: "Wait", started_at: waitMs },
  ];
  if (finishedMs !== undefined) {
    entries.push({ type: "cmd_end", index: 2, status: "passed", duration_ms: finishedMs - waitMs });
  }
  return entries;
}

/** Story-only API records. The displayed steps still use the real log parser. */
export function createConsoleDurationSimulation(initiallyRunning: boolean) {
  const name = initiallyRunning ? "live" : "completed";
  const organizationId = `duration-simulation-${name}`;
  const canvasId = `duration-canvas-${name}`;
  const runId = `duration-run-${name}`;
  const executionId = `duration-execution-${name}`;
  const nodeId = "simulate-agent";
  const rootEventId = `duration-event-${name}`;
  const referenceMs = initiallyRunning ? Date.now() : Date.parse("2026-10-24T12:28:20.000Z");
  const createdMs = referenceMs - LIFECYCLE_MS;
  const checkoutMs = referenceMs - COMMANDS_MS;
  const waitMs = referenceMs - 5 * MINUTE;
  let finishedMs: number | undefined = initiallyRunning ? undefined : referenceMs;
  const iso = (value: number) => new Date(value).toISOString();

  const canvas: CanvasesCanvas = {
    metadata: { id: canvasId, name: "Task duration simulation", organizationId },
    spec: {
      nodes: [{ id: nodeId, name: "Task agent", type: "TYPE_ACTION", component: "runnerClaudeCode" }],
      edges: [],
    },
  };

  function run(): CanvasesCanvasRun {
    const finished = finishedMs !== undefined;
    return {
      id: runId,
      canvasId,
      state: finished ? "STATE_FINISHED" : "STATE_STARTED",
      result: finished ? "RESULT_PASSED" : "RESULT_UNKNOWN",
      createdAt: iso(createdMs),
      updatedAt: iso(finishedMs ?? referenceMs),
      finishedAt: finishedMs === undefined ? undefined : iso(finishedMs),
      rootEvent: { id: rootEventId, createdAt: iso(createdMs) },
      executions: [
        {
          id: executionId,
          nodeId,
          state: finished ? "STATE_FINISHED" : "STATE_STARTED",
          result: finished ? "RESULT_PASSED" : "RESULT_UNKNOWN",
          createdAt: iso(checkoutMs),
          updatedAt: iso(finishedMs ?? referenceMs),
        },
      ],
    };
  }

  function phase(): SplitRunPhase {
    return {
      id: "implement",
      name: "Task agent",
      componentName: "Task agent",
      status: finishedMs === undefined ? "running" : "passed",
      duration: formatMinutesSecondsDuration((finishedMs ?? Date.now()) - createdMs),
      startedAt: iso(createdMs),
      endedAt: finishedMs === undefined ? undefined : iso(finishedMs),
      appId: canvasId,
      runId,
      artifacts: [],
      canvasSteps: [],
      stream: [
        {
          id: nodeId,
          nodeId,
          executionId,
          at: "12:00:00",
          component: "runnerClaudeCode",
          componentName: "Task agent",
          status: finishedMs === undefined ? "running" : "passed",
        },
      ],
    };
  }

  const logsPath = `/storybook/duration-simulation/${name}/logs`;
  const handlers = [
    http.get(`*/api/v1/canvases/${canvasId}`, () => HttpResponse.json({ canvas })),
    http.get(`*/api/v1/canvases/${canvasId}/runs/${runId}`, () => HttpResponse.json({ run: run() })),
    http.get(`*/api/v1/canvases/${canvasId}/events/${rootEventId}/executions`, () =>
      HttpResponse.json({ executions: run().executions }),
    ),
    http.get(`*/api/v1/canvases/${canvasId}/node-executions/${executionId}/runner-live-logs/session`, () =>
      HttpResponse.json({ backend: "integrated", stream_url: logsPath }),
    ),
    http.get(`*${logsPath}`, ({ request }) => {
      const entries = createCommandLogRecords(checkoutMs, waitMs, finishedMs);
      const after = Number(new URL(request.url).searchParams.get("after") ?? 0);
      return new HttpResponse(
        entries
          .slice(after)
          .map((entry) => JSON.stringify(entry))
          .join("\n") + "\n",
        {
          headers: {
            "Content-Type": "application/x-ndjson",
            "X-SuperPlane-Log-Cursor": String(entries.length),
            "X-SuperPlane-Log-State": finishedMs === undefined ? "active" : "archived",
          },
        },
      );
    }),
  ];

  return {
    organizationId,
    initiallyRunning,
    handlers,
    phase,
    automation() {
      const latest = stageFromPhase(phase());
      return { id: "duration-simulation", name: "Task agent", latest, runs: [latest] };
    },
    isFinished: () => finishedMs !== undefined,
    finish() {
      finishedMs ??= Date.now();
    },
  };
}

export type ConsoleDurationSimulation = ReturnType<typeof createConsoleDurationSimulation>;
