import { describe, expect, it } from "vitest";

import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";
import { stepExecutionEvent } from "./workOrderTimelineFromEvents.testHelpers";

describe("buildWorkOrderTimelineViewFromEvents: dispatch-step grouping", () => {
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
