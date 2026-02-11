import { describe, expect, it } from "vitest";

import { onIssueEventTriggerRenderer } from "./on_issue_event";

describe("sentry.onIssueEvent mapper", () => {
  it("normalizes wrapped event data", () => {
    const titleAndSubtitle = onIssueEventTriggerRenderer.getTitleAndSubtitle({
      event: {
        id: "evt_1",
        nodeId: "node_1",
        type: "sentry.issue.created",
        createdAt: new Date().toISOString(),
        data: {
          data: {
            action: "created",
            issue: {
              id: "123",
              title: "Boom",
            },
          },
        },
      },
    });

    expect(titleAndSubtitle.title).toBe("Boom");
    expect(titleAndSubtitle.subtitle).toContain("Created");
  });

  it("normalizes stringified event data", () => {
    const raw = JSON.stringify({
      action: "resolved",
      issue: {
        id: "123",
        title: "Fixed",
      },
    });

    const titleAndSubtitle = onIssueEventTriggerRenderer.getTitleAndSubtitle({
      event: {
        id: "evt_2",
        nodeId: "node_2",
        type: "sentry.issue.resolved",
        createdAt: new Date().toISOString(),
        data: raw,
      },
    });

    expect(titleAndSubtitle.title).toBe("Fixed");
    expect(titleAndSubtitle.subtitle).toContain("Resolved");
  });
});
