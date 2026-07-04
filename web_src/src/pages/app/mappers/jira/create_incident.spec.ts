import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
} from "../types";
import { createIncidentMapper } from "./create_incident";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "JSM",
    componentName: "jira.createIncident",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
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

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "jira.createIncident",
  label: "Create Incident",
  description: "",
  icon: "jira",
  color: "orange",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode({
      configuration: { summary: "Outage", serviceDesk: "6", serviceDeskRequestType: "75" },
      metadata: { serviceDeskName: "IT (IT)", requestTypeName: "Incident" },
    }),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

describe("createIncidentMapper", () => {
  it("getExecutionDetails includes issue key and url", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            {
              type: "jira.incident.created",
              timestamp: new Date().toISOString(),
              data: {
                id: "1",
                key: "IT-2",
                self: "https://test.atlassian.net/rest/api/3/issue/1",
              },
            },
          ],
        },
      },
    });
    const details = createIncidentMapper.getExecutionDetails(ctx);
    expect(details["Issue key"]).toBe("IT-2");
    expect(details["url"]).toBe("https://test.atlassian.net/rest/api/3/issue/1");
    expect(details["Issue id"]).toBeUndefined();
  });

  it("props metadata shows at most service desk, request type, and summary", () => {
    const props = createIncidentMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {
            summary: "Outage",
            serviceDesk: "6",
            serviceDeskRequestType: "75",
            priority: "High",
            dueDate: "2026-05-14",
          },
          metadata: { serviceDeskName: "IT (IT)", requestTypeName: "Incident" },
        }),
      }),
    );
    expect(props.metadata).toHaveLength(3);
    expect(props.metadata?.map((m) => m.label)).toEqual(["IT (IT)", "Incident", "Outage"]);
  });

  it("event state registry maps success to created", () => {
    const exec = buildExecution({ result: "RESULT_PASSED" });
    expect(eventStateRegistry.createIncident.getState(exec)).toBe("created");
  });
});
