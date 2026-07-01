import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import { eventStateRegistry } from "./index";
import type {
  ComponentBaseContext,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
  SubtitleContext,
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
  return { type: "logfire.result", timestamp: new Date().toISOString(), data };
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
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

// ── baseMapper.props ────────────────────────────────────────────────

describe("baseMapper.props", () => {
  it("returns props with correct title from node name", () => {
    const ctx = buildComponentBaseContext({ node: { name: "My Query" } });
    expect(baseMapper.props(ctx).title).toBe("My Query");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx = buildComponentBaseContext({ node: { name: "" } });
    expect(baseMapper.props(ctx).title).toBe("Query Logfire");
  });

  it("includes metadata for project, sql, and time window", () => {
    const ctx = buildComponentBaseContext({
      node: {
        configuration: { sql: "SELECT * FROM logs", project: "my-proj", timeWindow: "1h" },
        metadata: { project: { id: "p1", name: "My Project" } },
      },
    });
    const props = baseMapper.props(ctx);
    expect(props.metadata!.length).toBeGreaterThan(0);
    expect(props.metadata!.length).toBeLessThanOrEqual(3);
  });

  it("includes empty state when there are no executions", () => {
    const props = baseMapper.props(buildComponentBaseContext({ lastExecutions: [] }));
    expect(props.includeEmptyState).toBe(true);
    expect(props.eventSections).toBeUndefined();
  });

  it("does not include empty state when executions exist", () => {
    const ctx = buildComponentBaseContext({ lastExecutions: [buildExecution()] });
    expect(baseMapper.props(ctx).includeEmptyState).toBe(false);
  });
});

// ── baseMapper.getExecutionDetails ──────────────────────────────────

describe("baseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    expect(() => baseMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: undefined } }))).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    expect(() =>
      baseMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [] } } })),
    ).not.toThrow();
  });

  it("does not throw when node configuration and metadata are undefined", () => {
    expect(() =>
      baseMapper.getExecutionDetails(buildDetailsCtx({ node: { configuration: undefined, metadata: undefined } })),
    ).not.toThrow();
  });

  it("extracts project name from node metadata", () => {
    const ctx = buildDetailsCtx({ node: { metadata: { project: { id: "p1", name: "My Project" } } } });
    expect(baseMapper.getExecutionDetails(ctx)["Project"]).toBe("My Project");
  });

  it("extracts SQL from configuration", () => {
    const ctx = buildDetailsCtx({ execution: { configuration: { sql: "SELECT * FROM logs WHERE level = 'error'" } } });
    expect(baseMapper.getExecutionDetails(ctx)["SQL"]).toBe("SELECT * FROM logs WHERE level = 'error'");
  });

  it("truncates long SQL queries", () => {
    const ctx = buildDetailsCtx({ execution: { configuration: { sql: "SELECT " + "a".repeat(200) + " FROM logs" } } });
    const sql = baseMapper.getExecutionDetails(ctx)["SQL"];
    expect(sql.length).toBeLessThanOrEqual(123);
    expect(sql).toContain("...");
  });

  it("counts rows from array-based data", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ rows: [{ a: 1 }, { a: 2 }, { a: 3 }] })] } },
    });
    expect(baseMapper.getExecutionDetails(ctx)["Rows Returned"]).toBe("3");
  });

  it("counts rows from columnar data", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ col1: [1, 2, 3, 4, 5] })] } },
    });
    expect(baseMapper.getExecutionDetails(ctx)["Rows Returned"]).toBe("5");
  });

  it("returns 0 rows when data is empty or null", () => {
    expect(
      baseMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } }))[
        "Rows Returned"
      ],
    ).toBe("0");
    expect(
      baseMapper.getExecutionDetails(buildDetailsCtx({ execution: { outputs: { default: [buildOutput(null)] } } }))[
        "Rows Returned"
      ],
    ).toBe("0");
  });

  it("includes Executed At from updatedAt", () => {
    const ctx = buildDetailsCtx({ execution: { updatedAt: new Date().toISOString() } });
    expect(baseMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── baseMapper.subtitle ─────────────────────────────────────────────

describe("baseMapper.subtitle", () => {
  it("returns a non-empty value when execution has updatedAt", () => {
    const ctx: SubtitleContext = { node: buildNode(), execution: buildExecution() };
    expect(baseMapper.subtitle(ctx)).not.toBe("");
  });

  it("returns empty string when no timestamp is available", () => {
    const ctx: SubtitleContext = { node: buildNode(), execution: buildExecution({ updatedAt: "", createdAt: "" }) };
    expect(baseMapper.subtitle(ctx)).toBe("");
  });
});

// ── baseMapper metadata helpers ─────────────────────────────────────

describe("baseMapper.props metadata", () => {
  it("shows project metadata from node metadata name", () => {
    const ctx = buildComponentBaseContext({
      node: { metadata: { project: { id: "p1", name: "My Project" } }, configuration: {} },
    });
    const projectMeta = baseMapper.props(ctx).metadata?.find((m) => String(m.label).includes("Project"));
    expect(projectMeta).toBeDefined();
    expect(projectMeta!.label).toContain("My Project");
  });

  it("falls back to project from configuration when metadata is absent", () => {
    const ctx = buildComponentBaseContext({ node: { metadata: {}, configuration: { project: "proj-from-config" } } });
    const projectMeta = baseMapper.props(ctx).metadata?.find((m) => String(m.label).includes("Project"));
    expect(projectMeta!.label).toContain("proj-from-config");
  });

  it("shows SQL metadata", () => {
    const ctx = buildComponentBaseContext({ node: { configuration: { sql: "SELECT count(*) FROM traces" } } });
    expect(baseMapper.props(ctx).metadata?.find((m) => String(m.label).includes("SQL"))).toBeDefined();
  });

  it("shows time window metadata for preset windows", () => {
    const ctx = buildComponentBaseContext({ node: { configuration: { timeWindow: "1h" } } });
    expect(baseMapper.props(ctx).metadata?.find((m) => String(m.label).includes("Last 1 hour"))).toBeDefined();
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
    const twMeta = baseMapper.props(ctx).metadata?.find((m) => String(m.label).includes("Window"));
    expect(twMeta).toBeDefined();
    expect(twMeta!.label).toContain("from");
    expect(twMeta!.label).toContain("to");
  });

  it("omits time window when set to none", () => {
    const ctx = buildComponentBaseContext({ node: { configuration: { timeWindow: "none" } } });
    const props = baseMapper.props(ctx);
    expect(
      props.metadata?.find((m) => String(m.label).includes("Window") || String(m.label).includes("Last")),
    ).toBeUndefined();
  });

  it("limits metadata to 3 items", () => {
    const ctx = buildComponentBaseContext({
      node: {
        configuration: { sql: "SELECT * FROM logs", project: "proj", timeWindow: "1h" },
        metadata: { project: { name: "My Project" } },
      },
    });
    expect(baseMapper.props(ctx).metadata!.length).toBeLessThanOrEqual(3);
  });
});

// ── eventStateRegistry ──────────────────────────────────────────────

describe("eventStateRegistry.queryLogfire", () => {
  it("returns 'completed' for successful executions", () => {
    expect(eventStateRegistry.queryLogfire.getState(buildExecution())).toBe("completed");
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
