import { describe, expect, it } from "bun:test";
import { getDetailsForIncident } from "./base";

describe("getDetailsForIncident", () => {
  it("keeps agent fields after status-change timestamps", () => {
    const now = new Date();
    const details = getDetailsForIncident(
      {
        id: "inc-1",
        incident_key: "key-1",
        title: "API down",
        urgency: "high",
        status: "triggered",
        html_url: "https://pagerduty.example/inc-1",
        created_at: now.toISOString(),
        updated_at: new Date(now.getTime() + 60_000).toISOString(),
        incident_number: "42",
        service: { summary: "API", html_url: "https://pagerduty.example/svc" },
        escalation_policy: { summary: "On-call", html_url: "https://pagerduty.example/ep" },
        assignments: [{ assignee: { summary: "Ada" } }],
        last_status_change_at: new Date(now.getTime() + 120_000).toISOString(),
        resolved_at: new Date(now.getTime() + 180_000).toISOString(),
      },
      { summary: "Webhook", html_url: "https://pagerduty.example/agent" },
    );

    expect(Object.keys(details)).toEqual([
      "Created At",
      "Updated At",
      "ID",
      "Key",
      "Title",
      "Urgency",
      "Status",
      "Incident URL",
      "Number",
      "Service",
      "Service URL",
      "Escalation Policy",
      "Escalation Policy URL",
      "Assignments",
      "Last Status Change",
      "Resolved At",
      "Agent",
      "Agent URL",
    ]);
  });
});
