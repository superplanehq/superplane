import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import { onAlertReceivedTriggerRenderer } from "./on_alert_received";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
  TriggerEventContext,
  TriggerRendererContext,
  ComponentDefinition,
  EventInfo,
} from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Query Logfire",
    componentName: "logfire.queryLogfire",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "logfire.result",
    timestamp: new Date().toISOString(),
    data,
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

function buildComponentBaseContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "queryLogfire",
      label: "Query Logfire",
      description: "",
      icon: "logfire",
      color: "orange",
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: {
      id: "user-1",
      name: "Test User",
      email: "test@example.com",
      roles: ["admin"],
      groups: ["developers"],
    },
    actions: {
      invokeNodeExecutionAction: async () => {},
    },
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onAlertReceived",
    label: "On Alert Received",
    description: "",
    icon: "logfire",
    color: "orange",
    ...overrides,
  };
}

function buildEvent(overrides?: Partial<NonNullable<EventInfo>>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    data: {},
    nodeId: "node-1",
    type: "logfire.onAlertReceived",
    ...overrides,
  };
}

// ── baseMapper.props ────────────────────────────────────────────────

describe("baseMapper.props", () => {
  it("returns props with correct title from node name", () => {
    const ctx = buildComponentBaseContext({ node: { name: "My Query" } });
    const props = baseMapper.props(ctx);
    expect(props.title).toBe("My Query");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx = buildComponentBaseContext({ node: { name: "" } });
    const props = baseMapper.props(ctx);
    expect(props.title).toBe("Query Logfire");
  });

  it("includes metadata for project, sql, and time window", () => {
    const ctx = buildComponentBaseContext({
      node: {
        configuration: { sql: "SELECT * FROM logs", project: "my-proj", timeWindow: "1h" },
        metadata: { project: { id: "p1", name: "My Project" } },
      },
    });
    const props = baseMapper.props(ctx);
    expect(props.metadata).toBeDefined();
    expect(props.metadata!.length).toBeGreaterThan(0);
    expect(props.metadata!.length).toBeLessThanOrEqual(3);
  });

  it("includes empty state when there are no executions", () => {
    const ctx = buildComponentBaseContext({ lastExecutions: [] });
    const props = baseMapper.props(ctx);
    expect(props.includeEmptyState).toBe(true);
    expect(props.eventSections).toBeUndefined();
  });

  it("does not include empty state when executions exist", () => {
    const ctx = buildComponentBaseContext({
      lastExecutions: [buildExecution()],
    });
    const props = baseMapper.props(ctx);
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── baseMapper.getExecutionDetails ──────────────────────────────────

describe("baseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => baseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => baseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when node configuration and metadata are undefined", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: undefined, metadata: undefined },
    });
    expect(() => baseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts project name from node metadata", () => {
    const ctx = buildDetailsCtx({
      node: { metadata: { project: { id: "p1", name: "My Project" } } },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Project"]).toBe("My Project");
  });

  it("extracts SQL from configuration", () => {
    const ctx = buildDetailsCtx({
      execution: { configuration: { sql: "SELECT * FROM logs WHERE level = 'error'" } },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["SQL"]).toBe("SELECT * FROM logs WHERE level = 'error'");
  });

  it("truncates long SQL queries", () => {
    const longSql = "SELECT " + "a".repeat(200) + " FROM logs";
    const ctx = buildDetailsCtx({
      execution: { configuration: { sql: longSql } },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["SQL"].length).toBeLessThanOrEqual(123); // 120 + "..."
    expect(details["SQL"]).toContain("...");
  });

  it("counts rows from array-based data", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ rows: [{ a: 1 }, { a: 2 }, { a: 3 }] })],
        },
      },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Rows Returned"]).toBe("3");
  });

  it("counts rows from columnar data", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ col1: [1, 2, 3, 4, 5] })],
        },
      },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Rows Returned"]).toBe("5");
  });

  it("returns 0 rows when data is empty", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Rows Returned"]).toBe("0");
  });

  it("returns 0 rows when data is null", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(null)] } },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Rows Returned"]).toBe("0");
  });

  it("includes Executed At from updatedAt", () => {
    const now = new Date().toISOString();
    const ctx = buildDetailsCtx({
      execution: { updatedAt: now },
    });
    const details = baseMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
  });
});

// ── baseMapper.subtitle ─────────────────────────────────────────────

describe("baseMapper.subtitle", () => {
  it("returns a non-empty value when execution has updatedAt", () => {
    const ctx: SubtitleContext = {
      node: buildNode(),
      execution: buildExecution({ updatedAt: new Date().toISOString() }),
    };
    const subtitle = baseMapper.subtitle(ctx);
    expect(subtitle).not.toBe("");
  });

  it("returns empty string when no timestamp is available", () => {
    const ctx: SubtitleContext = {
      node: buildNode(),
      execution: buildExecution({ updatedAt: "", createdAt: "" }),
    };
    const subtitle = baseMapper.subtitle(ctx);
    expect(subtitle).toBe("");
  });
});

// ── baseMapper metadata helpers ─────────────────────────────────────

describe("baseMapper.props metadata", () => {
  it("shows project metadata from node metadata name", () => {
    const ctx = buildComponentBaseContext({
      node: {
        metadata: { project: { id: "p1", name: "My Project" } },
        configuration: {},
      },
    });
    const props = baseMapper.props(ctx);
    const projectMeta = props.metadata?.find((m) => String(m.label).includes("Project"));
    expect(projectMeta).toBeDefined();
    expect(projectMeta!.label).toContain("My Project");
  });

  it("falls back to project from configuration when metadata is absent", () => {
    const ctx = buildComponentBaseContext({
      node: {
        metadata: {},
        configuration: { project: "proj-from-config" },
      },
    });
    const props = baseMapper.props(ctx);
    const projectMeta = props.metadata?.find((m) => String(m.label).includes("Project"));
    expect(projectMeta).toBeDefined();
    expect(projectMeta!.label).toContain("proj-from-config");
  });

  it("shows SQL metadata", () => {
    const ctx = buildComponentBaseContext({
      node: { configuration: { sql: "SELECT count(*) FROM traces" } },
    });
    const props = baseMapper.props(ctx);
    const sqlMeta = props.metadata?.find((m) => String(m.label).includes("SQL"));
    expect(sqlMeta).toBeDefined();
  });

  it("shows time window metadata for preset windows", () => {
    const ctx = buildComponentBaseContext({
      node: { configuration: { timeWindow: "1h" } },
    });
    const props = baseMapper.props(ctx);
    const twMeta = props.metadata?.find((m) => String(m.label).includes("Last 1 hour"));
    expect(twMeta).toBeDefined();
  });

  it("shows custom time window with min/max timestamps", () => {
    const ctx = buildComponentBaseContext({
      node: {
        configuration: {
          timeWindow: "custom",
          minTimestamp: "2024-01-01T00:00:00Z",
          maxTimestamp: "2024-01-02T00:00:00Z",
        },
      },
    });
    const props = baseMapper.props(ctx);
    const twMeta = props.metadata?.find((m) => String(m.label).includes("Window"));
    expect(twMeta).toBeDefined();
    expect(twMeta!.label).toContain("from");
    expect(twMeta!.label).toContain("to");
  });

  it("omits time window when set to none", () => {
    const ctx = buildComponentBaseContext({
      node: { configuration: { timeWindow: "none" } },
    });
    const props = baseMapper.props(ctx);
    const twMeta = props.metadata?.find((m) => String(m.label).includes("Window") || String(m.label).includes("Last"));
    expect(twMeta).toBeUndefined();
  });

  it("limits metadata to 3 items", () => {
    const ctx = buildComponentBaseContext({
      node: {
        configuration: { sql: "SELECT * FROM logs", project: "proj", timeWindow: "1h" },
        metadata: { project: { name: "My Project" } },
      },
    });
    const props = baseMapper.props(ctx);
    expect(props.metadata!.length).toBeLessThanOrEqual(3);
  });
});

// ── onAlertReceivedTriggerRenderer.getTitleAndSubtitle ───────────────

describe("onAlertReceivedTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses alert name as title", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: { alertName: "High Error Rate" } }),
    };
    const { title } = onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(title).toBe("High Error Rate");
  });

  it("falls back to message when alert name is empty", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: { alertName: "", message: "5 matching rows" } }),
    };
    const { title } = onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(title).toBe("5 matching rows");
  });

  it("falls back to default when both alertName and message are missing", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: {} }),
    };
    const { title } = onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(title).toBe("Logfire alert received");
  });

  it("builds subtitle from eventType and severity", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({
        data: { alertName: "Test", eventType: "alert.fired", severity: "critical" },
      }),
    };
    const { subtitle } = onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toContain("alert.fired");
    expect(subtitle).toContain("critical");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: undefined }),
    };
    expect(() => onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── onAlertReceivedTriggerRenderer.getRootEventValues ────────────────

describe("onAlertReceivedTriggerRenderer.getRootEventValues", () => {
  it("extracts alert details from event data", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({
        data: {
          alertName: "High Error Rate",
          severity: "critical",
          message: "10 matching rows found",
          url: "https://logfire.pydantic.dev/alert/123",
        },
      }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Alert Name"]).toBe("High Error Rate");
    expect(values["Severity"]).toBe("critical");
    expect(values["Message"]).toBe("10 matching rows found");
    expect(values["Matching Rows"]).toBe("10");
    expect(values["View in Logfire"]).toBe("https://logfire.pydantic.dev/alert/123");
  });

  it("omits matching rows when message has no row count", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({
        data: { alertName: "Test", message: "Something happened" },
      }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Matching Rows"]).toBeUndefined();
  });

  it("returns empty strings when event data is missing", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: {} }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Alert Name"]).toBe("");
    expect(values["Severity"]).toBe("");
    expect(values["Message"]).toBe("");
  });

  it("includes received at from event createdAt", () => {
    const now = new Date().toISOString();
    const ctx: TriggerEventContext = {
      event: buildEvent({ createdAt: now }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Received At"]).toBeDefined();
    expect(values["Received At"]).not.toBe("");
  });

  it("falls back to event data timestamp when createdAt is missing", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({
        createdAt: "",
        data: { timestamp: "2024-01-01T00:00:00Z" },
      }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Received At"]).toBe("2024-01-01T00:00:00Z");
  });
});

// ── onAlertReceivedTriggerRenderer.getTriggerProps ───────────────────

describe("onAlertReceivedTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Alert Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.title).toBe("My Alert Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Alert Received" }),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.title).toBe("On Alert Received");
  });

  it("includes project and alert metadata", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: {
          project: { id: "p1", name: "My Project" },
          alert: { id: "a1", name: "My Alert" },
        },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata).toBeDefined();
    const projectMeta = props.metadata?.find((m) => String(m.label).includes("Project"));
    const alertMeta = props.metadata?.find((m) => String(m.label).includes("Alert"));
    expect(projectMeta).toBeDefined();
    expect(alertMeta).toBeDefined();
  });

  it("limits metadata to 3 items", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: {
          project: { name: "Proj" },
          alert: { name: "Alert" },
        },
        configuration: { project: "p", alert: "a" },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata!.length).toBeLessThanOrEqual(3);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({
        data: { alertName: "Test Alert", severity: "warning" },
      }),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Test Alert");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits metadata when project and alert are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ──────────────────────────────────────────────

describe("eventStateRegistry.queryLogfire", () => {
  it("returns 'completed' for successful executions", () => {
    const execution = buildExecution();
    expect(eventStateRegistry.queryLogfire.getState(execution)).toBe("completed");
  });

  it("returns running state when execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.queryLogfire.getState(execution)).toBe("running");
  });

  it("returns error state when execution failed", () => {
    const execution = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "query timed out",
    });
    expect(eventStateRegistry.queryLogfire.getState(execution)).toBe("error");
  });
});
