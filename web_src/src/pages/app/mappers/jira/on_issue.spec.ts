import { describe, expect, it } from "vitest";
import type { EventInfo, TriggerEventContext } from "../types";
import { onIssueTriggerRenderer } from "./on_issue";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-07-17T09:30:00Z").toISOString(),
    nodeId: "node-1",
    type: "jira.issue",
    data,
  };
}

const issueData = {
  action: "created",
  issue: {
    id: "10001",
    key: "ENG-42",
    self: "https://your-domain.atlassian.net/rest/api/3/issue/10001",
    fields: {
      summary: "Login page returns 500 on invalid password",
      issuetype: { name: "Bug" },
      status: { name: "To Do" },
      priority: { name: "High" },
      assignee: { displayName: "Alice Smith" },
      reporter: { displayName: "Bob Jones" },
    },
  },
  user: { displayName: "Bob Jones" },
};

describe("onIssueTriggerRenderer", () => {
  it("derives an issue title from the event", () => {
    const context: TriggerEventContext = { event: event(issueData) };
    expect(onIssueTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "ENG-42 - Login page returns 500 on invalid password",
    );
  });

  it("falls back to a generic title when no issue is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIssueTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Issue event");
  });

  it("maps the issue fields to root event values", () => {
    const context: TriggerEventContext = { event: event(issueData) };
    const values = onIssueTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Action"]).toBe("Created");
    expect(values["Key"]).toBe("ENG-42");
    expect(values["Summary"]).toBe("Login page returns 500 on invalid password");
    expect(values["Status"]).toBe("To Do");
    expect(values["Priority"]).toBe("High");
    expect(values["Issue Type"]).toBe("Bug");
    expect(values["Assignee"]).toBe("Alice Smith");
    expect(values["Reporter"]).toBe("Bob Jones");
  });
});
