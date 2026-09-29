import { describe, expect, it } from "bun:test";

import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";
import { stepExecutionEvent } from "./workOrderTimelineFromEvents.testHelpers";

describe("buildWorkOrderTimelineViewFromEvents: content broadcasts", () => {
  it("folds a line-step broadcast into the dispatch step", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.content.broadcast",
        event: {
          summary: "Preview environment is ready",
          body: "Open the preview environment.",
          url: "https://preview.example.com/orders/12",
          urlLabel: "Preview",
          automation: {
            lineId: "line-1",
            lineName: "CI",
            stepName: "Build",
          },
        },
      },
    ]);

    const dispatched = view.events.find((event) => event.kind === "dispatched");
    expect(dispatched?.steps?.[0]?.broadcasts).toEqual([
      {
        summary: "Preview environment is ready",
        body: "Open the preview environment.",
        url: "https://preview.example.com/orders/12",
        urlLabel: "Preview",
      },
    ]);
    expect(view.events.some((event) => event.kind === "contentBroadcast")).toBe(false);
  });

  it("keeps a step broadcast when the step later finishes", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      stepExecutionEvent("step.execution.created", "2026-08-04T12:00:00.000Z", "started"),
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.content.broadcast",
        event: {
          summary: "Preview environment is ready",
          url: "https://preview.example.com/orders/12",
          automation: { lineId: "line-1", lineName: "CI", stepName: "Build" },
        },
      },
      stepExecutionEvent("step.execution.finished", "2026-08-04T12:02:00.000Z", "finished", "passed"),
    ]);

    expect(view.events.find((event) => event.kind === "dispatched")?.steps?.[0]?.broadcasts?.[0]?.summary).toBe(
      "Preview environment is ready",
    );
  });

  it("keeps a broadcast without a matching step as its own activity item", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.content.broadcast",
        event: {
          summary: "Preview environment is ready",
          url: "https://preview.example.com",
          automation: { nodeName: "create-preview", appId: "app-1", appName: "Preview" },
          run: { id: "run-9" },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "contentBroadcast",
      sourceRunId: "run-9",
      sourceAppId: "app-1",
      broadcast: {
        summary: "Preview environment is ready",
        url: "https://preview.example.com",
      },
    });
  });

  it("drops a broadcast that has no summary", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:01:00.000Z",
        type: "order.content.broadcast",
        event: { url: "https://preview.example.com" },
      },
    ]);

    expect(view.events).toEqual([]);
  });
});
