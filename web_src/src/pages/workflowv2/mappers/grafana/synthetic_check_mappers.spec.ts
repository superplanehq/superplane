import { describe, expect, it } from "vitest";

import { createHttpSyntheticCheckMapper } from "./create_http_synthetic_check";
import { deleteHttpSyntheticCheckMapper } from "./delete_http_synthetic_check";
import { getHttpSyntheticCheckMapper } from "./get_http_synthetic_check";
import { getGrafanaSyntheticCheckFlatView } from "./synthetic_check_shared";
import { updateHttpSyntheticCheckMapper } from "./update_http_synthetic_check";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function buildNode(componentName: string, overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Grafana Synthetic Check",
    componentName,
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-04-15T10:20:30Z",
    updatedAt: "2026-04-15T10:20:30Z",
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
    type: "grafana.syntheticCheck",
    timestamp: "2026-04-15T10:20:30Z",
    data,
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
      icon: "activity",
      color: "blue",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
  };
}

function buildExecutionContext(
  componentName: string,
  overrides?: { node?: Partial<NodeInfo>; execution?: Partial<ExecutionInfo> },
): ExecutionDetailsContext {
  const node = buildNode(componentName, overrides?.node);

  return {
    nodes: [node],
    node,
    execution: buildExecution(overrides?.execution),
  };
}

describe("grafana synthetic check mappers", () => {
  it("create mapper renders metadata from configuration", () => {
    const props = createHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.createHttpSyntheticCheck", {
        configuration: {
          request: { target: "https://api.example.com/health", method: "GET" },
          schedule: { frequency: 60, probes: ["1", "2"] },
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
    expect(props.metadata).toEqual([
      expect.objectContaining({ icon: "globe", label: "https://api.example.com/health" }),
      expect.objectContaining({ icon: "arrow-right", label: "GET" }),
      expect.objectContaining({ icon: "map-pin", label: "1, 2 · Every 1m" }),
    ]);
  });

  it("create mapper resolves nested request and schedule configuration", () => {
    const props = createHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.createHttpSyntheticCheck", {
        configuration: {
          request: { target: "https://nested.example.com", method: "POST" },
          schedule: { frequency: 120, probes: ["9"] },
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
    expect(props.metadata).toEqual([
      expect.objectContaining({ icon: "globe", label: "https://nested.example.com" }),
      expect.objectContaining({ icon: "arrow-right", label: "POST" }),
      expect.objectContaining({ icon: "map-pin", label: "9 · Every 2m" }),
    ]);
  });

  it("create mapper treats nested frequency as seconds even for large exact values", () => {
    const props = createHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.createHttpSyntheticCheck", {
        configuration: {
          request: { target: "https://nested.example.com", method: "GET" },
          schedule: { frequency: 1000, probes: ["9"] },
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "map-pin", label: "9 · Every 1000s" })]),
    );
  });

  it("create mapper keeps legacy flat millisecond frequency readable", () => {
    const props = createHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.createHttpSyntheticCheck", {
        configuration: {
          target: "https://api.example.com/health",
          method: "GET",
          probes: ["1"],
          frequency: 60000,
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "map-pin", label: "1 · Every 1m" })]),
    );
  });

  it("does not surface legacy or nested TLS configuration in the frontend flat view", () => {
    const flat = getGrafanaSyntheticCheckFlatView({
      target: "https://legacy.example.com",
      method: "GET",
      request: {
        target: "https://nested.example.com",
        method: "POST",
        tls: {
          insecureSkipVerify: true,
          serverName: "nested.example.com",
        },
      },
      tls: {
        insecureSkipVerify: true,
        serverName: "legacy.example.com",
      },
    } as unknown as Parameters<typeof getGrafanaSyntheticCheckFlatView>[0]);

    expect(flat).toEqual(
      expect.objectContaining({
        target: "https://nested.example.com",
        method: "POST",
      }),
    );
    expect(flat).not.toHaveProperty("tls");
  });

  it("create mapper prefers probe summary from node metadata over raw probe ids", () => {
    const props = createHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.createHttpSyntheticCheck", {
        configuration: {
          target: "https://api.example.com/health",
          method: "GET",
          schedule: { frequency: 60, probes: ["17"] },
        },
        metadata: {
          probeSummary: "Amsterdam (EU)",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "map-pin", label: "Amsterdam (EU) · Every 1m" })]),
    );
  });

  it("update mapper shows three metadata rows without duplicating the target URL", () => {
    const props = updateHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.updateHttpSyntheticCheck", {
        configuration: {
          syntheticCheck: "42",
          request: { target: "https://www.elffie.com", method: "GET" },
          schedule: { frequency: 60, probes: ["17"] },
        },
        metadata: {
          checkLabel: "Api health check (https://www.elffie.com)",
          probeSummary: "Ohio (AMER)",
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
    expect(props.metadata).toEqual([
      expect.objectContaining({
        icon: "activity",
        label: "Api health check (https://www.elffie.com)",
      }),
      expect.objectContaining({ icon: "arrow-right", label: "GET" }),
      expect.objectContaining({ icon: "map-pin", label: "Ohio (AMER) · Every 1m" }),
    ]);
  });

  it("create mapper shows details similar to dash0 synthetic create", () => {
    const details = createHttpSyntheticCheckMapper.getExecutionDetails(
      buildExecutionContext("grafana.createHttpSyntheticCheck", {
        node: {
          configuration: {
            request: { target: "https://api.example.com/health", method: "GET" },
            schedule: {
              frequency: 60,
              timeout: 3000,
              probes: ["1", "2"],
              enabled: true,
            },
          },
        },
        execution: {
          outputs: {
            default: [
              buildOutput({
                check: {
                  id: 101,
                  target: "https://api.example.com/health",
                  frequency: 60000,
                  timeout: 3000,
                  enabled: true,
                  probes: [1, 2],
                  settings: { http: { method: "GET" } },
                },
                checkUrl: "https://grafana.example.com/a/grafana-synthetic-monitoring-app/checks/101",
              }),
            ],
          },
        },
      }),
    );

    expect(details["Created At"]).toContain("2026");
    expect(details.Check).toContain("/a/grafana-synthetic-monitoring-app/checks/101");
    expect(details.Target).toBe("GET https://api.example.com/health");
    expect(details.Schedule).toBe("Every 1m");
    expect(details.Timeout).toBe("3s");
    expect(details.Enabled).toBe("Yes");
  });

  it("get mapper uses node metadata for selection metadata", () => {
    const props = getHttpSyntheticCheckMapper.props(
      buildComponentContext("grafana.getHttpSyntheticCheck", {
        configuration: {
          syntheticCheck: "101",
        },
        metadata: {
          checkLabel: "API health check (https://api.example.com/health)",
        },
      }),
    );

    expect(props.metadata).toEqual([expect.objectContaining({ label: expect.stringContaining("API health check") })]);
  });

  it("get mapper shows configuration and metrics in details", () => {
    const details = getHttpSyntheticCheckMapper.getExecutionDetails(
      buildExecutionContext("grafana.getHttpSyntheticCheck", {
        execution: {
          outputs: {
            up: [
              buildOutput({
                configuration: {
                  id: 101,
                  job: "API health check",
                  target: "https://api.example.com/health",
                  frequency: 60000,
                  timeout: 3000,
                  enabled: true,
                  probes: [1, 2],
                  settings: { http: { method: "GET" } },
                },
                metrics: {
                  lastOutcome: "Up",
                  uptimePercent24h: 99.9,
                  reachabilityPercent24h: 99.86,
                  totalRuns24h: 1440,
                  successRuns24h: 1438,
                  failureRuns24h: 2,
                  averageLatencySeconds24h: 0.142,
                  sslEarliestExpiryAt: "2026-05-15T10:25:00Z",
                  sslEarliestExpiryDays: 30,
                  frequencyMilliseconds: 60000,
                  lastExecutionAt: "2026-04-15T10:25:00Z",
                },
                checkUrl: "https://grafana.example.com/a/grafana-synthetic-monitoring-app/checks/101",
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)[0]).toBe("Fetched At");
    expect(details["Last Outcome"]).toBe("Up");
    expect(details.Job).toBe("API health check");
    expect(details.Target).toBe("GET https://api.example.com/health");
    expect(details.Schedule).toBe("Every 1m · 3s timeout");
    expect(details["Health (24h)"]).toBe("99.90% uptime · 99.86% reachability");
    expect(details["Runs (24h)"]).toBe("1438 succeeded · 2 failed · 1440 total");
    expect(details["SSL Expiry"]).toContain("(30d)");
    expect(details["Avg Latency (24h)"]).toBe("0.142s");
  });

  it("get mapper rounds fractional run counts in details", () => {
    const details = getHttpSyntheticCheckMapper.getExecutionDetails(
      buildExecutionContext("grafana.getHttpSyntheticCheck", {
        execution: {
          outputs: {
            up: [
              buildOutput({
                configuration: {
                  id: 101,
                  job: "API health check",
                  target: "https://api.example.com/health",
                  frequency: 60000,
                  timeout: 3000,
                  enabled: true,
                  probes: [1, 2],
                  settings: { http: { method: "GET" } },
                },
                metrics: {
                  successRuns24h: 144.20027816411684,
                  failureRuns24h: 0,
                  totalRuns24h: 144.20027816411684,
                },
              }),
            ],
          },
        },
      }),
    );

    expect(details["Runs (24h)"]).toBe("144 succeeded · 0 failed · 144 total");
  });

  it("update mapper tolerates missing outputs", () => {
    const details = updateHttpSyntheticCheckMapper.getExecutionDetails(
      buildExecutionContext("grafana.updateHttpSyntheticCheck", {
        node: {
          configuration: {
            request: { target: "https://api.example.com/health", method: "GET" },
            schedule: {
              frequency: 60,
              timeout: 3000,
              probes: ["1"],
            },
          },
        },
        execution: { outputs: undefined },
      }),
    );

    expect(details.Schedule).toBe("Every 1m");
    expect(details.Target).toBe("GET https://api.example.com/health");
  });

  it("delete mapper surfaces deletion details", () => {
    const details = deleteHttpSyntheticCheckMapper.getExecutionDetails(
      buildExecutionContext("grafana.deleteHttpSyntheticCheck", {
        execution: {
          outputs: {
            default: [
              buildOutput({
                syntheticCheck: "101",
                job: "API health check",
                target: "https://api.example.com/health",
                deleted: true,
              }),
            ],
          },
        },
      }),
    );

    expect(details["Check ID"]).toBe("101");
    expect(details.Job).toBe("API health check");
    expect(details.Target).toBe("https://api.example.com/health");
    expect(details.Status).toBe("Deleted");
  });
});
