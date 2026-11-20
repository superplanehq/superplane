import { describe, expect, it } from "bun:test";
import { baseEventSections, getDetailsForIncident } from "./base";
import { defaultTriggerRenderer } from "../default";
import type { ExecutionInfo } from "../types";
import type { Incident } from "./types";

function makeExecution(): ExecutionInfo {
  const createdAt = new Date().toISOString();
  return {
    id: "exec-1",
    createdAt,
    updatedAt: createdAt,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: {
      id: "event-1",
      createdAt,
      data: {},
      nodeId: "missing-trigger",
      type: "rootly.onIncident",
    },
  };
}

describe("rootly baseEventSections", () => {
  it("uses the default trigger renderer when the root trigger node is absent", () => {
    const execution = makeExecution();
    const sections = baseEventSections([], execution, "rootly.getIncident");
    const expectedTitle = defaultTriggerRenderer.getTitleAndSubtitle({ event: execution.rootEvent }).title;

    expect(sections).toHaveLength(1);
    expect(sections[0].eventTitle).toBe(expectedTitle);
  });
});

describe("rootly getDetailsForIncident", () => {
  const fallbackDetails = { ID: "-", Title: "-", Summary: "-", Status: "-", Severity: "-" };

  it("returns all details in order with local timestamp formatting", () => {
    const startedAt = new Date().toISOString();
    const mitigatedAt = new Date(Date.now() + 60_000).toISOString();
    const resolvedAt = new Date(Date.now() + 120_000).toISOString();
    const updatedAt = new Date(Date.now() + 180_000).toISOString();
    const incident: Incident = {
      id: "incident-1",
      title: "API outage",
      summary: "Requests fail",
      status: "resolved",
      severity: "Critical",
      started_at: startedAt,
      mitigated_at: mitigatedAt,
      resolved_at: resolvedAt,
      updated_at: updatedAt,
      url: "https://app.rootly.com/incidents/incident-1",
    };

    expect(Object.entries(getDetailsForIncident(incident))).toEqual([
      ["ID", "incident-1"],
      ["Title", "API outage"],
      ["Summary", "Requests fail"],
      ["Status", "resolved"],
      ["Severity", "Critical"],
      ["Started At", new Date(startedAt).toLocaleString()],
      ["Mitigated At", new Date(mitigatedAt).toLocaleString()],
      ["Resolved At", new Date(resolvedAt).toLocaleString()],
      ["Updated At", new Date(updatedAt).toLocaleString()],
      ["Incident URL", "https://app.rootly.com/incidents/incident-1"],
    ]);
  });

  it("omits absent and empty optional values and preserves empty basic fallbacks", () => {
    const mitigatedAt = new Date().toISOString();
    const updatedAt = new Date(Date.now() + 60_000).toISOString();
    const incident: Incident = {
      id: "",
      title: "",
      summary: "",
      status: "",
      severity: "",
      started_at: "",
      mitigated_at: mitigatedAt,
      updated_at: updatedAt,
      url: "",
    };

    expect(Object.entries(getDetailsForIncident(incident))).toEqual([
      ...Object.entries(fallbackDetails),
      ["Mitigated At", new Date(mitigatedAt).toLocaleString()],
      ["Updated At", new Date(updatedAt).toLocaleString()],
    ]);
  });

  it("returns only ordered basic fallbacks for an undefined incident", () => {
    expect(Object.entries(getDetailsForIncident(undefined))).toEqual(Object.entries(fallbackDetails));
  });
});
