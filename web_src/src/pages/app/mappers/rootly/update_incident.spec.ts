import { describe, expect, it } from "bun:test";
import type { MetadataItem } from "@/ui/metadataList";
import type { ComponentBaseContext, NodeInfo } from "../types";
import { updateIncidentMapper } from "./update_incident";

function buildComponentContext(nodeOverrides: Partial<NodeInfo>): ComponentBaseContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Update Incident",
    componentName: "rootly.updateIncident",
    isCollapsed: false,
    ...nodeOverrides,
  };

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "rootly.updateIncident",
      label: "Update Incident",
      description: "",
      icon: "rootly",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

const metadataCases: { name: string; node: Partial<NodeInfo>; expected: MetadataItem[] }[] = [
  { name: "omitted configuration", node: {}, expected: [] },
  { name: "null configuration", node: { configuration: null }, expected: [] },
  { name: "empty configuration", node: { configuration: {} }, expected: [] },
  {
    name: "incident without updates",
    node: { configuration: { incidentId: "incident-42" } },
    expected: [{ icon: "alert-triangle", label: "Incident: incident-42" }],
  },
  {
    name: "scalar updates in order",
    node: {
      configuration: {
        status: "in_progress",
        subStatus: "investigating",
        severity: "critical",
        title: "API outage",
        summary: "Requests fail",
      },
    },
    expected: [{ icon: "funnel", label: "Updating: Status: in_progress, Sub-Status, Severity, Title, Summary" }],
  },
  {
    name: "empty collections",
    node: { configuration: { services: [], teams: [], labels: [] } },
    expected: [],
  },
  {
    name: "non-array collections",
    node: { configuration: { services: "service-1", teams: { length: 2 }, labels: true } },
    expected: [],
  },
  {
    name: "collection counts in order",
    node: { configuration: { services: ["service-1"], teams: ["team-1", "team-2"], labels: ["a", "b", "c"] } },
    expected: [{ icon: "funnel", label: "Updating: Services (1), Teams (2), Labels (3)" }],
  },
  {
    name: "incident before all updates",
    node: {
      configuration: {
        incidentId: "incident-42",
        status: "resolved",
        subStatus: "closed",
        severity: "minor",
        title: "API recovered",
        summary: "Requests succeed",
        services: ["service-1", "service-2"],
        teams: ["team-1"],
        labels: ["a", "b", "c"],
      },
    },
    expected: [
      { icon: "alert-triangle", label: "Incident: incident-42" },
      {
        icon: "funnel",
        label: "Updating: Status: resolved, Sub-Status, Severity, Title, Summary, Services (2), Teams (1), Labels (3)",
      },
    ],
  },
];

describe("updateIncidentMapper.props", () => {
  it.each(metadataCases)("preserves metadata for $name", ({ node, expected }) => {
    expect(updateIncidentMapper.props(buildComponentContext(node)).metadata).toEqual(expected);
  });

  it.each([undefined, null, "", false, 0])("omits falsy scalar fields (%p)", (value) => {
    const context = buildComponentContext({
      configuration: {
        incidentId: value,
        status: value,
        subStatus: value,
        severity: value,
        title: value,
        summary: value,
      },
    });

    expect(updateIncidentMapper.props(context).metadata).toEqual([]);
  });
});
