import type { CanvasesCanvasEvent, CanvasesListEventExecutionsResponse } from "@/api-client";
import { describe, expect, it } from "vitest";

import { analyzingTicketsFromIntakeRuns, automationRunsFromIntakeEvents } from "./intakeRunModel";

describe("analyzingTicketsFromIntakeRuns", () => {
  it("returns GitHub events with an active analysis execution", () => {
    const events: CanvasesCanvasEvent[] = [
      {
        id: "event-1",
        data: { data: { issue: { title: "Fix duplicate refunds" } } },
      },
      {
        id: "event-2",
        data: { data: { issue: { title: "Update documentation" } } },
      },
    ];
    const executions: Array<CanvasesListEventExecutionsResponse | undefined> = [
      {
        executions: [{ nodeId: "analysis", state: "STATE_STARTED" }],
      },
      {
        executions: [{ nodeId: "analysis", state: "STATE_FINISHED", result: "RESULT_PASSED" }],
      },
    ];

    expect(analyzingTicketsFromIntakeRuns("github-issues", "analysis", events, executions)).toEqual([
      { id: "event-1", title: "Fix duplicate refunds" },
    ]);
  });

  it("reads titles from Sentry and PagerDuty payloads", () => {
    const running = [{ executions: [{ nodeId: "analysis", state: "STATE_PENDING" as const }] }];

    expect(
      analyzingTicketsFromIntakeRuns(
        "sentry-exceptions",
        "analysis",
        [{ id: "sentry-1", data: { data: { data: { issue: { title: "Checkout failed" } } } } }],
        running,
      ),
    ).toEqual([{ id: "sentry-1", title: "Checkout failed" }]);
    expect(
      analyzingTicketsFromIntakeRuns(
        "pagerduty-incidents",
        "analysis",
        [{ id: "pd-1", data: { data: { incident: { title: "API latency" } } } }],
        running,
      ),
    ).toEqual([{ id: "pd-1", title: "API latency" }]);
  });
});

describe("automationRunsFromIntakeEvents", () => {
  it("maps completed events to scored backlog and below-threshold runs", () => {
    const events: CanvasesCanvasEvent[] = [
      {
        id: "event-1",
        runId: "run-1",
        createdAt: "2026-08-24T16:00:00Z",
        data: { data: { issue: { title: "Fix duplicate refunds" } } },
      },
      {
        id: "event-2",
        runId: "run-2",
        createdAt: "2026-08-24T16:10:00Z",
        data: { data: { issue: { title: "Update documentation" } } },
      },
    ];
    const executions: Array<CanvasesListEventExecutionsResponse | undefined> = [
      {
        executions: [
          {
            nodeId: "analysis",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            updatedAt: "2026-08-24T16:01:00Z",
            outputs: { passed: [{ data: { result: { result: "82" } } }] },
          },
          { nodeId: "create", state: "STATE_FINISHED", result: "RESULT_PASSED" },
        ],
      },
      {
        executions: [
          {
            nodeId: "analysis",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            updatedAt: "2026-08-24T16:11:00Z",
            outputs: { passed: [{ data: { result: { result: 41 } } }] },
          },
        ],
      },
    ];

    expect(
      automationRunsFromIntakeEvents(
        {
          appId: "app-1",
          sourceId: "github-issues",
          analysisNodeId: "analysis",
          createWorkOrderNodeId: "create",
        },
        events,
        executions,
        new Date("2026-08-24T16:20:00Z"),
      ),
    ).toEqual([
      {
        id: "event-1",
        appId: "app-1",
        runId: "run-1",
        title: "Fix duplicate refunds",
        confidencePct: 82,
        ranMinutesAgo: 20,
        analyzedMinutesAgo: 19,
        placement: "backlog",
      },
      {
        id: "event-2",
        appId: "app-1",
        runId: "run-2",
        title: "Update documentation",
        confidencePct: 41,
        ranMinutesAgo: 10,
        analyzedMinutesAgo: 9,
        placement: "below-threshold",
      },
    ]);
  });

  it("omits active runs and marks failed analysis as rejected", () => {
    const events: CanvasesCanvasEvent[] = [
      { id: "active", runId: "run-active", data: { data: { incident: { title: "Active incident" } } } },
      { id: "failed", runId: "run-failed", data: { data: { incident: { title: "Failed incident" } } } },
    ];
    const executions: Array<CanvasesListEventExecutionsResponse | undefined> = [
      { executions: [{ nodeId: "analysis", state: "STATE_STARTED" }] },
      {
        executions: [
          {
            nodeId: "analysis",
            state: "STATE_FINISHED",
            result: "RESULT_FAILED",
            outputs: {},
          },
        ],
      },
    ];

    expect(
      automationRunsFromIntakeEvents(
        {
          appId: "app-1",
          sourceId: "pagerduty-incidents",
          analysisNodeId: "analysis",
          createWorkOrderNodeId: "create",
        },
        events,
        executions,
        new Date("2026-08-24T16:20:00Z"),
      ),
    ).toEqual([
      expect.objectContaining({
        id: "failed",
        title: "Failed incident",
        confidencePct: 0,
        placement: "rejected",
      }),
    ]);
  });
});
