import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onIncidentTriggerRenderer } from "./on_incident";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-07-17T09:30:00Z").toISOString(),
    nodeId: "node-1",
    type: "jira.incident",
    data,
  };
}

const incidentData = {
  action: "created",
  issue: {
    id: "10001",
    key: "DEMO-42",
    self: "https://your-domain.atlassian.net/rest/api/3/issue/10001",
    fields: {
      summary: "Payments API returning 500s",
      issuetype: { name: "Incident" },
      status: { name: "To Do" },
      priority: { name: "High" },
    },
  },
  user: { displayName: "Bob Jones" },
};

describe("onIncidentTriggerRenderer.getTitleAndSubtitle", () => {
  it("derives an incident title from the event", () => {
    const context: TriggerEventContext = { event: event(incidentData) };
    expect(onIncidentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("DEMO-42 - Payments API returning 500s");
  });

  it("falls back to a generic title when no issue is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onIncidentTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Incident event");
  });
});

describe("onIncidentTriggerRenderer.getRootEventValues", () => {
  it("maps the issue fields to root event values", () => {
    const context: TriggerEventContext = { event: event(incidentData) };
    const values = onIncidentTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Action"]).toBe("Created");
    expect(values["Key"]).toBe("DEMO-42");
    expect(values["Summary"]).toBe("Payments API returning 500s");
    expect(values["Status"]).toBe("To Do");
    expect(values["Priority"]).toBe("High");
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Incident",
    componentName: "jira.onIncident",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildTriggerContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastEvent?: EventInfo;
  definition?: Partial<ComponentDefinition>;
}): TriggerRendererContext {
  return {
    node: buildNode(overrides?.node),
    definition: {
      name: "jira.onIncident",
      label: "On Incident",
      description: "",
      icon: "jira",
      color: "orange",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onIncidentTriggerRenderer.getTriggerProps", () => {
  it("renders the service desk, request type count, and selected events", () => {
    const props = onIncidentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { serviceDesk: "2", requestTypes: ["5", "7"], events: ["created", "updated"] },
          metadata: { serviceDesk: { projectKey: "DEMO", projectName: "Demo service space" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "layers", label: "Demo service space (DEMO)" });
    expect(props.metadata?.[1]).toEqual({ icon: "tag", label: "2 request types" });
    expect(props.metadata?.[2]).toEqual({ icon: "funnel", label: "Created, Updated" });
  });

  it("uses the singular form for a single request type", () => {
    const props = onIncidentTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: { configuration: { serviceDesk: "2", requestTypes: ["5"], events: ["created"] } },
      }),
    );

    expect(props.metadata?.[1]).toEqual({ icon: "tag", label: "1 request type" });
  });

  it("falls back to the configured service desk id when metadata is missing", () => {
    const props = onIncidentTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { serviceDesk: "2", events: ["created"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "layers", label: "2" });
  });
});
