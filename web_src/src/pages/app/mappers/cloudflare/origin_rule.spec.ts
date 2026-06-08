import { describe, expect, it } from "vitest";

import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";
import { eventStateRegistry } from "./index";
import { originRuleMapper } from "./origin_rule";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Origin Rule",
    componentName: "cloudflare.createOriginRule",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "execution-1",
    createdAt: "2026-05-06T12:00:00Z",
    updatedAt: "2026-05-06T12:01:00Z",
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

function buildOutput(data: unknown, type = "cloudflare.createOriginRule"): OutputPayload {
  return {
    type,
    timestamp: "2026-05-06T12:00:00Z",
    data,
  };
}

function buildComponentContext(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(nodeOverrides);

  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: node.componentName,
      label: "Create Origin Rule",
      description: "",
      icon: "cloud",
      color: "orange",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function buildExecutionContext(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);

  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides?.execution),
  };
}

describe("cloudflare origin rule mapper", () => {
  it("renders a maximum of three metadata items and uses zone name instead of zone id", () => {
    const props = originRuleMapper.props(
      buildComponentContext({
        configuration: {
          zone: "zone123",
          matchMode: "custom",
          matchRules: [{ field: "uriPath", operator: "wildcard", value: "/api/*" }],
          originHost: "origin.example.com",
          enabled: true,
        },
        metadata: {
          zoneName: "example.com",
          rewrites: ["DNS Record", "Host Header"],
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
    expect(props.metadata?.[0]).toEqual({ icon: "globe", label: "example.com" });
    expect(props.metadata?.some((item) => item.label === "zone123")).toBe(false);
  });

  it("puts the compulsory execution timestamp first on the details tab", () => {
    const details = originRuleMapper.getExecutionDetails(
      buildExecutionContext({
        node: {
          configuration: {
            rule: "zone123/rule123",
            originHost: "configured-origin.example.com",
            enabled: false,
          },
          metadata: {
            zoneName: "example.com",
          },
        },
        execution: {
          outputs: {
            default: [
              buildOutput(
                {
                  zoneId: "zone123",
                  rule: {
                    id: "rule123",
                    expression: "true",
                    enabled: true,
                    action_parameters: {
                      host_header: "api.example.com",
                      origin: {
                        host: "api-origin.example.com",
                        port: 8443,
                      },
                      sni: {
                        value: "tls.example.com",
                      },
                    },
                  },
                },
                "cloudflare.createOriginRule",
              ),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(details["Executed At"]).toBe(new Date("2026-05-06T12:00:00Z").toLocaleString());
    expect(details["Zone"]).toBe("example.com");
    expect(details["DNS Record"]).toBe("api-origin.example.com");
    expect(details["Host Header"]).toBe("api.example.com");
    expect(details["SNI"]).toBe("tls.example.com");
    expect(details["Destination Port"]).toBe("8443");
    expect(details["Enabled"]).toBe("Yes");
  });

  it("uses wrapped example-output style payload data for details", () => {
    const details = originRuleMapper.getExecutionDetails(
      buildExecutionContext({
        execution: {
          outputs: {
            default: [
              buildOutput(
                {
                  zoneId: "zone123",
                  rule: {
                    id: "rule123",
                    expression: `starts_with(http.request.uri.path, "/api/")`,
                    enabled: true,
                    action_parameters: {
                      host_header: "api.example.com",
                    },
                  },
                },
                "cloudflare.updateOriginRule",
              ),
            ],
          },
        },
      }),
    );

    expect(details["Rule ID"]).toBe("rule123");
    expect(details["Match"]).toBe(`starts_with(http.request.uri.path, "/api/")`);
    expect(details["Host Header"]).toBe("api.example.com");
  });
});

describe("cloudflare origin rule event states", () => {
  it("labels successful origin rule actions by action type", () => {
    const execution = buildExecution();

    expect(eventStateRegistry.createOriginRule.getState(execution)).toBe("created");
    expect(eventStateRegistry.updateOriginRule.getState(execution)).toBe("updated");
    expect(eventStateRegistry.deleteOriginRule.getState(execution)).toBe("deleted");
  });

  it("keeps non-successful origin rule states unchanged", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNKNOWN",
      resultReason: "RESULT_REASON_OK",
    });

    expect(eventStateRegistry.createOriginRule.getState(execution)).toBe("running");
    expect(eventStateRegistry.updateOriginRule.getState(execution)).toBe("running");
    expect(eventStateRegistry.deleteOriginRule.getState(execution)).toBe("running");
  });
});
