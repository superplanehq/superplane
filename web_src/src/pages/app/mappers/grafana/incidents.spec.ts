import { describe, expect, it } from "vitest";

import { addIncidentActivityMapper } from "./add_incident_activity";
import { declareDrillMapper, declareIncidentMapper } from "./declare_incident";
import { getIncidentMapper } from "./get_incident";
import { resolveIncidentMapper } from "./resolve_incident";
import { updateIncidentMapper } from "./update_incident";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana Incident",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  const now = new Date().toISOString();

  return {
    id: "exec-1",
    createdAt: now,
    updatedAt: now,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "grafana.incident",
    timestamp: "2026-04-20T10:00:00Z",
    data,
  };
}

function buildExecutionContext(
  componentName: string,
  data: unknown,
  nodeOverrides?: Partial<NodeInfo>,
): ExecutionDetailsContext {
  const node = buildNode(componentName, nodeOverrides);

  return {
    nodes: [node],
    node,
    execution: buildExecution({ outputs: { default: [buildOutput(data)] } }),
  };
}

function buildComponentContext(componentName: string, nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(componentName, nodeOverrides);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: componentName,
      label: componentName,
      description: "",
      icon: "alert-triangle",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

const incident = {
  incidentID: "incident-123",
  title: "API latency",
  severity: "critical",
  status: "active",
  summary: "A long summary that should not be dumped into details",
  labels: [{ label: "prod" }, { label: "api" }, { label: "checkout" }, { label: "database" }],
  createdTime: "2026-04-20T09:55:00Z",
  modifiedTime: "2026-04-20T10:01:00Z",
  closedTime: "2026-04-20T10:30:00Z",
  incidentUrl: "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123",
};

describe("Grafana incident execution details", () => {
  it("declareIncident shows a curated set of fields", () => {
    const details = declareIncidentMapper.getExecutionDetails(buildExecutionContext("declareIncident", incident));

    expect(Object.keys(details)).toEqual(["Declared At", "Title", "Severity", "Status", "Labels", "Incident URL"]);
    expect(details.Labels).toBe("prod, api, checkout +1");
    expect(details.Summary).toBeUndefined();
  });

  it("getIncident does not dump the full incident object", () => {
    const details = getIncidentMapper.getExecutionDetails(buildExecutionContext("getIncident", incident));

    expect(Object.keys(details)).toEqual(["Fetched At", "Title", "Severity", "Status", "Labels", "Incident URL"]);
    expect(details.Labels).toBe("prod, api, checkout +1");
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("updateIncident focuses on updated incident state", () => {
    const details = updateIncidentMapper.getExecutionDetails(buildExecutionContext("updateIncident", incident));

    expect(Object.keys(details)).toEqual(["Updated At", "Title", "Severity", "Status", "Labels", "Incident URL"]);
    expect(details.Labels).toBe("prod, api, checkout +1");
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("resolveIncident shows labels and resolved status", () => {
    const details = resolveIncidentMapper.getExecutionDetails(
      buildExecutionContext("resolveIncident", { ...incident, status: "resolved" }),
    );

    expect(Object.keys(details)).toEqual(["Resolved At", "Title", "Severity", "Status", "Labels", "Incident URL"]);
    expect(details.Labels).toBe("prod, api, checkout +1");
    expect(details.Status).toBe("resolved");
  });

  it("addIncidentActivity limits activity details", () => {
    const details = addIncidentActivityMapper.getExecutionDetails(
      buildExecutionContext("addIncidentActivity", {
        activityItemID: "activity-123",
        incidentID: "incident-123",
        activityKind: "userNote",
        body: "Root cause identified",
        createdTime: "2026-04-20T10:10:00Z",
        fieldValues: { ignored: true },
      }),
    );

    expect(Object.keys(details)).toEqual(["Added At", "Incident ID", "Activity ID", "Body", "Created At"]);
    expect(details.fieldValues).toBeUndefined();
  });
});

describe("Grafana incident card metadata", () => {
  it("uses concise declare incident metadata", () => {
    const props = declareIncidentMapper.props(
      buildComponentContext("declareIncident", {
        configuration: { title: "API latency", severity: "critical", status: "resolved" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ label: "API latency" }),
        expect.objectContaining({ label: "Severity: critical" }),
        expect.objectContaining({ label: "Status: resolved" }),
      ]),
    );
  });

  it("always marks declare drill metadata as drill", () => {
    const props = declareDrillMapper.props(
      buildComponentContext("declareDrill", {
        configuration: { title: "Game day", severity: "major" },
      }),
    );

    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ label: "Drill" })]));
  });

  it("uses resolved incident metadata label for selected incidents", () => {
    const props = getIncidentMapper.props(
      buildComponentContext("getIncident", {
        configuration: { incident: "incident-123" },
        metadata: { label: "API latency [active] (incident-123)" },
      }),
    );

    expect(props.metadata).toEqual([
      expect.objectContaining({ label: "Incident: API latency [active] (incident-123)" }),
    ]);
  });

  it("shows labels in update incident metadata", () => {
    const props = updateIncidentMapper.props(
      buildComponentContext("updateIncident", {
        configuration: { incident: "incident-123", labels: ["prod", "api"] },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ label: "Updating: Labels (2)" })]),
    );
  });
});
