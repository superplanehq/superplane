import { describe, expect, it } from "vitest";

import type { FactoriesWorkOrderEvent } from "@/api-client";

import { buildWorkOrderTimelineView } from "./workOrderTimelineEvents";
import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

function stepExecutionEvent(
  type: "step.execution.created" | "step.execution.finished",
  timestamp: string,
  runState: string,
  runResult?: string,
  overrides: { stepName?: string; runId?: string } = {},
): FactoriesWorkOrderEvent {
  return {
    timestamp,
    type,
    event: {
      stepName: overrides.stepName ?? "Build",
      line: { id: "line-1", name: "CI" },
      run: { id: overrides.runId ?? "run-1", state: runState, result: runResult },
      app: { id: "app-1" },
    },
  };
}

describe("buildWorkOrderTimelineViewFromEvents", () => {
  it("hydrates timeline steps with execution usage", () => {
    const view = buildWorkOrderTimelineView(
      [stepExecutionEvent("step.execution.finished", "2026-08-04T12:00:00.000Z", "finished", "passed")],
      undefined,
      [{ id: "execution-1", run: { id: "run-1" }, totalTokens: "1200", costCents: "45" }],
    );

    expect(view.events[0]?.steps?.[0]?.execution).toMatchObject({
      id: "execution-1",
      run: { id: "run-1" },
      totalTokens: "1200",
      costCents: "45",
    });
  });

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

  it("renders a draft→open transition as an open with source-run enrichment", () => {
    // `order.status.updated` is the sole lifecycle event; the reader derives
    // the visual kind + title from the transition.
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          fromState: "draft",
          toState: "open",
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
      statusChange: { fromState: "draft", toState: "open" },
    });
    expect(view.events[0]?.actorUserId).toBeUndefined();
  });

  it("includes the closing user on close transitions", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          user: { id: "user-1" },
          fromState: "open",
          toState: "closed",
          toResult: "completed",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "closed",
      actorUserId: "user-1",
      title: "closed as completed",
      statusChange: { fromState: "open", toState: "closed", toResult: "completed" },
    });
  });

  it("labels a close-as-failed result", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: { fromState: "open", toState: "closed", toResult: "failed" },
      },
    ]);

    expect(view.events[0]?.title).toBe("closed as failed");
  });

  it("labels an abandon (draft→closed as rejected) with the same close styling", () => {
    // `draft → closed` (allowed only with `rejected`) surfaces the same
    // `closed` kind as a normal open→closed, so the timeline shows a single
    // consistent entry per termination.
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.status.updated",
        event: {
          user: { id: "user-1" },
          fromState: "draft",
          toState: "closed",
          toResult: "rejected",
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "closed",
      title: "closed as rejected",
      statusChange: { fromState: "draft", toState: "closed", toResult: "rejected" },
    });
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
      title: "moved this work order back to Draft",
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
      title: "reopened this work order",
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
    expect(view.events[0]?.sourceRunId).toBeUndefined();
    expect(view.events[0]?.sourceAppId).toBeUndefined();
  });

  it("carries the source run onto a top-level automation comment", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Investigating the failure.",
          author: {
            kind: "automation",
            automation: { nodeName: "review-payload", appId: "app-1", appName: "Refund Diagnostics" },
          },
          run: { id: "run-99" },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "commented",
      sourceRunId: "run-99",
      sourceAppId: "app-1",
    });
  });

  it("does not attach a source run to a user comment", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Looks good to me.",
          author: { kind: "user", userId: "user-1" },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({ kind: "commented" });
    expect(view.events[0]?.sourceRunId).toBeUndefined();
    expect(view.events[0]?.sourceAppId).toBeUndefined();
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
            data: {
              url: "https://github.com/example/repo/pull/1",
              title: "Add checkout",
            },
          },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "artifactAdded",
      artifact: {
        id: "art-1",
        type: "pr",
        data: {
          url: "https://github.com/example/repo/pull/1",
          title: "Add checkout",
        },
      },
      title: "attached PR: Add checkout",
    });
  });

  it("falls back to data.url when data.title is absent", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.artifact.added",
        event: {
          artifact: {
            id: "art-2",
            type: "pr",
            data: { url: "https://github.com/example/repo/pull/2" },
          },
        },
      },
    ]);

    expect(view.events[0]?.title).toBe("attached PR: https://github.com/example/repo/pull/2");
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
        type: "order.status.updated",
        event: {
          automation: {
            nodeName: "node-comment",
            appName: "Factory-App",
            lineName: "Plan",
            stepName: "step-01",
          },
          fromState: "open",
          toState: "closed",
          toResult: "completed",
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
          artifact: { id: "art-1", type: "pr", data: { url: "https://example.com/pull/1", title: "PR" } },
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
    expect(view.events[0]?.sourceRunId).toBeUndefined();
    expect(view.events[0]?.sourceAppId).toBeUndefined();
  });

  it("groups an automation artifact into its dispatch step", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.artifact.added",
        event: {
          automation: {
            lineId: "line-1",
            lineName: "CI",
            stepName: "Build",
          },
          artifact: {
            id: "artifact-1",
            type: "pr",
            data: { number: 42, url: "https://github.com/example/repo/pull/42" },
          },
        },
      },
    ]);

    expect(view.events).toHaveLength(1);
    expect(view.events[0]).toMatchObject({
      kind: "dispatched",
      steps: [
        {
          stepName: "Build",
          artifacts: [
            {
              id: "artifact-1",
              type: "pr",
              data: { number: 42, url: "https://github.com/example/repo/pull/42" },
            },
          ],
        },
      ],
    });
  });

  it("groups automation artifacts by explicit step index (including index 0)", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      stepExecutionEvent("step.execution.created", "2026-08-04T12:01:00.000Z", "started", undefined, {
        stepName: "Verify",
        runId: "run-2",
      }),
      {
        timestamp: "2026-08-04T12:02:00.000Z",
        type: "order.artifact.added",
        event: {
          automation: { lineId: "line-1", lineName: "CI", stepIndex: 0 },
          artifact: { id: "artifact-1", type: "markdown", data: { title: "Plan" } },
        },
      },
    ]);

    const dispatched = view.events.find((event) => event.kind === "dispatched");
    expect(dispatched?.steps?.[0]?.artifacts).toEqual([
      { id: "artifact-1", type: "markdown", data: { title: "Plan" } },
    ]);
    expect(dispatched?.steps?.[1]?.artifacts).toBeUndefined();
  });

  it("leaves an unmatchable automation artifact as a top-level event", () => {
    // Regression: previously fell back to the last dispatch step, which
    // misattributed step-0 refs whose JSON `stepIndex` was dropped by
    // `omitempty` on the Go int.
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      stepExecutionEvent("step.execution.created", "2026-08-04T12:01:00.000Z", "started", undefined, {
        stepName: "Verify",
        runId: "run-2",
      }),
      {
        timestamp: "2026-08-04T12:02:00.000Z",
        type: "order.artifact.added",
        event: {
          automation: { lineId: "line-1", lineName: "CI" },
          artifact: { id: "artifact-1", type: "pr", data: { url: "https://example.com/pr/1" } },
        },
      },
    ]);

    const dispatched = view.events.find((event) => event.kind === "dispatched");
    for (const step of dispatched?.steps ?? []) {
      expect(step.artifacts).toBeUndefined();
    }
    expect(view.events.find((event) => event.kind === "artifactAdded")?.artifact?.id).toBe("artifact-1");
  });

  it("groups an automation comment into its dispatch step, labelled with the line name and linked to its run", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Ready for review",
          author: {
            kind: "automation",
            automation: {
              lineId: "line-1",
              lineName: "CI",
              stepName: "Build",
              appId: "app-comment",
            },
          },
          run: { id: "run-comment-1" },
        },
      },
    ]);

    expect(view.events).toHaveLength(1);
    expect(view.events[0]).toMatchObject({
      kind: "dispatched",
      steps: [
        {
          stepName: "Build",
          comments: [
            {
              body: "Ready for review",
              label: "CI",
              sourceRunId: "run-comment-1",
              sourceAppId: "app-comment",
            },
          ],
        },
      ],
    });
  });

  it("labels a step comment with the line name even when the node name differs (regression)", () => {
    // Guards against reintroducing the "node name" bug: the label shown in
    // the timeline must be the automation (line) name, never the canvas
    // node name, even though both are present on the author ref.
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Ready for review",
          author: {
            kind: "automation",
            automation: {
              nodeId: "node-42",
              nodeName: "review-payload",
              lineId: "line-1",
              lineName: "CI",
              stepName: "Build",
            },
          },
        },
      },
    ]);

    const step = view.events[0]?.steps?.[0];
    expect(step?.comments?.[0]?.label).toBe("CI");
  });

  it("falls back to a node/app label for a step comment with no resolvable line name", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Ready for review",
          author: {
            kind: "automation",
            automation: {
              lineId: "line-1",
              nodeName: "review-payload",
              appName: "Refund Diagnostics",
              stepName: "Build",
            },
          },
        },
      },
    ]);

    const step = view.events[0]?.steps?.[0];
    expect(step?.comments?.[0]?.label).toBe("review-payload · Refund Diagnostics");
  });
});
