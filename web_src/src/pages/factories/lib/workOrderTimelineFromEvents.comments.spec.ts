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

  it("captures pull request metadata", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.pull_request.added",
        event: {
          pullRequest: {
            id: "pr-1",
            url: "https://github.com/example/repo/pull/1",
            title: "Add checkout",
            number: 1,
            state: "open",
          },
        },
      },
    ]);

    expect(view.events[0]).toMatchObject({
      kind: "pullRequestAdded",
      pullRequest: {
        id: "pr-1",
        url: "https://github.com/example/repo/pull/1",
        title: "Add checkout",
        number: "1",
        state: "open",
      },
      title: "added pull request #1",
    });
  });

  it("falls back to the pull request title when the number is absent", () => {
    const view = buildWorkOrderTimelineViewFromEvents([
      {
        timestamp: "2026-08-04T12:00:00.000Z",
        type: "order.pull_request.added",
        event: {
          pullRequest: {
            id: "pr-2",
            url: "https://github.com/example/repo/pull/2",
            title: "Add checkout",
          },
        },
      },
    ]);

    expect(view.events[0]?.title).toBe("added pull request Add checkout");
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
          artifact: { id: "art-1", type: "markdown", data: { title: "plan.md" } },
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
