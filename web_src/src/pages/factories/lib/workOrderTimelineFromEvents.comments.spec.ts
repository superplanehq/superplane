import { describe, expect, it } from "vitest";

import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

describe("buildWorkOrderTimelineViewFromEvents: comments, artifacts, and attribution", () => {
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

  it("uses the API event's own id as the comment id, and passes through reactions", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        id: "event-42",
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Nice work!",
          author: { kind: "user", userId: "user-1" },
          reactions: [
            { emoji: "+1", count: 2, reactedByMe: true },
            { emoji: "eyes", count: 1, reactedByMe: false },
          ],
        },
      },
    ]);

    expect(view.events[0]?.comment).toMatchObject({
      id: "event-42",
      reactions: [
        { emoji: "+1", count: 2, reactedByMe: true },
        { emoji: "eyes", count: 1, reactedByMe: false },
      ],
    });
  });

  it("defaults to an empty reactions array when the API omits the field", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        id: "event-43",
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Looks good.",
          author: { kind: "user", userId: "user-1" },
        },
      },
    ]);

    expect(view.events[0]?.comment?.reactions).toEqual([]);
  });

  it("falls back to a synthetic id when the API event has none", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.comment.added",
        event: {
          body: "Looks good.",
          author: { kind: "user", userId: "user-1" },
        },
      },
    ]);

    expect(view.events[0]?.comment?.id).toBe("comment-0");
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
});
