import { describe, expect, it } from "bun:test";
import type { ExecutionInfo, NodeInfo } from "../types";
import { getDetailsForIncident } from "./base";
import { createIncidentMapper } from "./create_incident";
import { onIncidentTriggerRenderer } from "./on_incident";
import type { Incident } from "./types";

const createdAt = new Date().toISOString();
const updatedAt = new Date(Date.now() + 60_000).toISOString();
const completeIncident: Incident = {
  id: "incident-1",
  name: "API outage",
  summary: "Requests fail",
  reference: "INC-1",
  visibility: "public",
  severity: { name: "Critical" },
  incident_status: { name: "Investigating" },
  created_at: createdAt,
  updated_at: updatedAt,
  permalink: "https://app.incident.io/incidents/incident-1",
};
const fallbackDetails = { ID: "-", Name: "-", Summary: "-", Reference: "-", Visibility: "-" };
const completeDetails = {
  ID: "incident-1",
  Name: "API outage",
  Summary: "Requests fail",
  Reference: "INC-1",
  Visibility: "public",
  Severity: "Critical",
  Status: "Investigating",
  "Created At": new Date(createdAt).toLocaleString(),
  "Updated At": new Date(updatedAt).toLocaleString(),
  "Incident URL": "https://app.incident.io/incidents/incident-1",
};

describe("getDetailsForIncident", () => {
  it("returns ordered mandatory fallbacks for an undefined incident", () => {
    expect(Object.entries(getDetailsForIncident(undefined))).toEqual(Object.entries(fallbackDetails));
  });

  it("returns all details in order with local timestamp formatting", () => {
    expect(Object.entries(getDetailsForIncident(completeIncident))).toEqual(Object.entries(completeDetails));
  });

  it.each([
    ["empty optional strings and nested names", { severity: { name: "" }, incident_status: { name: "" } }],
    ["absent nested names", { severity: {}, incident_status: {} }],
    ["absent nested objects", {}],
  ] satisfies [string, Incident][])("omits %s and preserves empty mandatory fallbacks", (_name, nestedFields) => {
    const incident: Incident = {
      id: "",
      name: "",
      summary: "",
      reference: "",
      visibility: "",
      created_at: "",
      updated_at: "",
      permalink: "",
      ...nestedFields,
    };
    expect(Object.entries(getDetailsForIncident(incident))).toEqual(Object.entries(fallbackDetails));
  });
});

describe("incident detail callers", () => {
  it.each([
    ["complete", completeIncident, completeDetails],
    ["missing", undefined, fallbackDetails],
  ] satisfies [string, Incident | undefined, Record<string, string>][])(
    "returns ordered trigger details for a %s incident",
    (_name, incident, expected) => {
      const details = onIncidentTriggerRenderer.getRootEventValues({
        event: { id: "event-1", createdAt, nodeId: "node-1", type: "incident.created", data: { incident } },
      });
      expect(Object.entries(details)).toEqual(Object.entries(expected));
    },
  );

  const node: NodeInfo = {
    id: "node-1",
    name: "Create incident",
    componentName: "incident.createIncident",
    isCollapsed: false,
  };
  const execution: ExecutionInfo = {
    id: "execution-1",
    createdAt,
    updatedAt,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
  };

  it("appends the first default output details after the formatted execution timestamp", () => {
    const details = createIncidentMapper.getExecutionDetails({
      nodes: [node],
      node,
      execution: {
        ...execution,
        outputs: {
          default: [{ data: completeIncident }, { data: { ...completeIncident, id: "incident-2" } }],
        },
      },
    });
    expect(Object.entries(details)).toEqual([
      ["Executed at", new Date(createdAt).toLocaleString()],
      ...Object.entries(completeDetails),
    ]);
  });

  it.each([undefined, { default: [] }, { default: [{ data: undefined }] }])(
    "returns only the execution timestamp when incident output is missing (%j)",
    (outputs) => {
      const details = createIncidentMapper.getExecutionDetails({
        nodes: [node],
        node,
        execution: { ...execution, outputs },
      });
      expect(details).toEqual({ "Executed at": new Date(createdAt).toLocaleString() });
    },
  );
});
