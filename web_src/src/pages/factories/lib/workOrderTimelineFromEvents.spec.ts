import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderEvent } from "@/api-client";

import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

function stepExecutionEvent(
  type: "step.execution.created" | "step.execution.finished",
  timestamp: string,
  runState: string,
  runResult?: string,
): FactoriesWorkOrderEvent {
  return {
    timestamp,
    type,
    event: {
      stepName: "Build",
      line: { id: "line-1", name: "CI" },
      run: { id: "run-1", state: runState, result: runResult },
      app: { id: "app-1" },
    },
  };
}

describe("buildWorkOrderTimelineViewFromEvents", () => {
  it("keeps finished step state when created and finished share a timestamp", () => {
    const timestamp = "2026-08-04T12:00:00.000Z";
    const apiEvents = [
      stepExecutionEvent("step.execution.finished", timestamp, "finished", "passed"),
      stepExecutionEvent("step.execution.created", timestamp, "pending"),
    ];

    const view = buildWorkOrderTimelineViewFromEvents(apiEvents);
    const step = view.events[0]?.steps?.[0];

    expect(step?.finishedAt).toBe(timestamp);
    expect(step?.execution?.state).toBe("STATE_FINISHED");
    expect(step?.execution?.result).toBe("RESULT_PASSED");
  });

  it("describes self-assignment when the actor assigns only themselves", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.assignees.updated",
        event: {
          user: { id: "user-1" },
          assigned: [{ id: "user-1" }],
        },
      },
    ]);

    expect(view.events[0]?.title).toBe("self-assigned");
  });

  it("renders order.opened (draft→open) as a status change and links it to the source run", () => {
    // Under the new FSM `order.opened` fires on the first `draft → open`
    // transition, not at creation. It must not duplicate the `created` entry
    // produced by the initial `status.updated ("" → draft)`.
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.opened",
        event: {
          run: { id: "run-1", state: "started" },
          app: { id: "app-1" },
          order: { id: "order-1", title: "Fix bug" },
        },
      },
    ]);

    expect(view.events).toHaveLength(1);
    expect(view.events[0]).toMatchObject({
      kind: "statusChanged",
      sourceRunId: "run-1",
      sourceAppId: "app-1",
      title: "opened this work order",
      statusChange: { fromState: "draft", toState: "open", fromResult: "", toResult: "" },
    });
    expect(view.events[0]?.actorUserId).toBeUndefined();
  });

  it("includes the closing user on closed events", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.closed",
        event: {
          user: { id: "user-1" },
          result: "completed",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "closed",
      actorUserId: "user-1",
      title: "closed as completed",
    });
  });

  it("labels a close-as-failed result", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.closed",
        event: { result: "failed" },
      },
    ]);

    expect(view.events[0]?.title).toBe("closed as failed");
  });

  it("skips status transitions already covered by opened/closed events", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: { fromState: "draft", toState: "open" },
      },
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: { fromState: "open", toState: "closed", toResult: "completed" },
      },
    ]);

    expect(view.events).toHaveLength(0);
  });

  it("renders the initial creation (empty fromState → draft) as a created entry", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          user: { id: "user-1" },
          fromState: "",
          toState: "draft",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "created",
      actorUserId: "user-1",
      title: "created this work order",
    });
  });

  it("renders an open→draft transition as a status change", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          user: { id: "user-1" },
          fromState: "open",
          toState: "draft",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "statusChanged",
      actorUserId: "user-1",
      statusChange: { fromState: "open", toState: "draft" },
      title: "moved Open → Draft",
    });
  });

  it("renders a reopen (closed→open) as a reopen, not a fresh open", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          user: { id: "user-1" },
          fromState: "closed",
          toState: "open",
          fromResult: "completed",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "statusChanged",
      actorUserId: "user-1",
      statusChange: { fromState: "closed", toState: "open", fromResult: "completed" },
      title: "reopened as Open",
    });
  });

  it("carries a comment body and automation ref into the timeline", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Please double-check the payload shape.",
          author: {
            kind: "automation",
            automation: { nodeName: "review-payload", appName: "Refund Diagnostics" },
          },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "commented",
      comment: {
        body: "Please double-check the payload shape.",
        authorKind: "automation",
        automation: { nodeName: "review-payload", appName: "Refund Diagnostics" },
      },
    });
  });

  it("captures PR artifact metadata", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.artifact.added",
        event: {
          artifact: {
            id: "art-1",
            type: "pr",
            url: "https://github.com/example/repo/pull/1",
            title: "Add checkout",
          },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "artifactAdded",
      artifact: {
        id: "art-1",
        type: "pr",
        url: "https://github.com/example/repo/pull/1",
        title: "Add checkout",
      },
    });
  });

  it("attributes automation-driven status changes with the factory line", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          automation: {
            nodeName: "node-comment",
            appName: "Factory-App",
            lineName: "Plan",
            stepName: "step-01",
          },
          fromState: "open",
          toState: "draft",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "statusChanged",
      actorAutomation: {
        lineName: "Plan",
        stepName: "step-01",
        nodeName: "node-comment",
      },
    });
  });

  it("attributes automation-driven closes with the factory line", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.closed",
        event: {
          automation: {
            nodeName: "node-comment",
            appName: "Factory-App",
            lineName: "Plan",
            stepName: "step-01",
          },
          result: "completed",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "closed",
      actorAutomation: {
        lineName: "Plan",
        stepName: "step-01",
        nodeName: "node-comment",
      },
      title: "closed as completed",
    });
  });

  it("attributes automation-driven artifacts with the factory line", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.artifact.added",
        event: {
          automation: {
            nodeName: "attach-artifact",
            appName: "Factory-App",
            lineName: "Plan",
            stepName: "step-01",
          },
          artifact: { id: "art-1", type: "pr", url: "https://example.com/pull/1", title: "PR" },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "artifactAdded",
      actorAutomation: {
        lineName: "Plan",
        stepName: "step-01",
        nodeName: "attach-artifact",
      },
    });
  });

  it("propagates comment automation into actorAutomation for the timeline", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Ready for review",
          author: {
            kind: "automation",
            automation: {
              nodeName: "node-comment",
              appName: "Factory-App",
              lineName: "Plan",
              stepName: "step-01",
            },
          },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "commented",
      actorAutomation: {
        lineName: "Plan",
        stepName: "step-01",
        nodeName: "node-comment",
      },
      comment: {
        automation: {
          lineName: "Plan",
          stepName: "step-01",
          nodeName: "node-comment",
        },
      },
    });
  });
});
