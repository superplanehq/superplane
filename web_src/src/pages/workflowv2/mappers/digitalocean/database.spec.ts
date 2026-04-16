import { describe, expect, it } from "vitest";

import { createDatabaseMapper } from "./create_database";
import { deleteDatabaseMapper } from "./delete_database";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "digitalocean.createDatabase",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "digitalocean.database.created",
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

function buildComponentContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastExecutions?: ExecutionInfo[];
  componentDefinition?: Partial<ComponentDefinition>;
}): ComponentBaseContext {
  const node = buildNode(overrides?.node);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "digitalocean.createDatabase",
      label: "Create Database",
      description: "",
      icon: "database",
      color: "blue",
      ...overrides?.componentDefinition,
    },
    lastExecutions: overrides?.lastExecutions ?? [],
    currentUser: undefined,
    actions: { invokeNodeExecutionAction: async () => {} },
  };
}

// ── createDatabaseMapper ─────────────────────────────────────────────

describe("createDatabaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns executed at without database fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    const details = createDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Database Name"]).toBe("-");
    expect(details["Database Cluster"]).toBe("-");
  });

  it("extracts database name and cluster from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "app_db",
              databaseClusterId: "cluster-uuid",
              databaseClusterName: "primary-postgres",
            }),
          ],
        },
      },
    });
    const details = createDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database Name"]).toBe("app_db");
    expect(details["Database Cluster"]).toBe("primary-postgres");
  });

  it("falls back to databaseClusterId when cluster name is absent", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ name: "db1", databaseClusterId: "9cc10173-e9ea-4176-9dbc-a4cee4c4ff30" })],
        },
      },
    });
    expect(createDatabaseMapper.getExecutionDetails(ctx)["Database Cluster"]).toBe(
      "9cc10173-e9ea-4176-9dbc-a4cee4c4ff30",
    );
  });

  it("does not throw when node configuration and metadata are undefined", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: undefined, metadata: undefined },
      execution: {
        outputs: {
          default: [buildOutput({ name: "app_db", databaseClusterName: "primary" })],
        },
      },
    });
    expect(() => createDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("createDatabaseMapper.props", () => {
  it("includes database name from configuration in metadata", () => {
    const props = createDatabaseMapper.props(
      buildComponentContext({
        node: {
          componentName: "digitalocean.createDatabase",
          configuration: { databaseCluster: "c1", name: "app_db" },
          metadata: { databaseClusterName: "primary-postgres" },
        },
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "app_db" },
      { icon: "server", label: "primary-postgres" },
    ]);
  });

  it("falls back to metadata databaseName when configuration name is absent", () => {
    const props = createDatabaseMapper.props(
      buildComponentContext({
        node: {
          componentName: "digitalocean.createDatabase",
          configuration: { databaseCluster: "c1" },
          metadata: { databaseName: "from_meta", databaseClusterName: "cluster-a" },
        },
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "from_meta" },
      { icon: "server", label: "cluster-a" },
    ]);
  });
});

// ── deleteDatabaseMapper ─────────────────────────────────────────────

describe("deleteDatabaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({
      node: { componentName: "digitalocean.deleteDatabase" },
      execution: { outputs: undefined },
    });
    expect(() => deleteDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({
      node: { componentName: "digitalocean.deleteDatabase" },
      execution: { outputs: { default: [] } },
    });
    expect(() => deleteDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns executed at with dash status when output has no database payload", () => {
    const ctx = buildDetailsCtx({
      node: { componentName: "digitalocean.deleteDatabase" },
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = deleteDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Database Name"]).toBe("-");
    expect(details["Database Cluster"]).toBe("-");
    expect(details["Status"]).toBe("-");
  });

  it("extracts name, cluster, and deleted status from output", () => {
    const ctx = buildDetailsCtx({
      node: { componentName: "digitalocean.deleteDatabase" },
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "app_db",
              databaseClusterId: "cluster-uuid",
              databaseClusterName: "primary-postgres",
              deleted: true,
            }),
          ],
        },
      },
    });
    const details = deleteDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database Name"]).toBe("app_db");
    expect(details["Database Cluster"]).toBe("primary-postgres");
    expect(details["Status"]).toBe("Deleted");
  });

  it("shows dash status when deleted is false", () => {
    const ctx = buildDetailsCtx({
      node: { componentName: "digitalocean.deleteDatabase" },
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "app_db",
              databaseClusterName: "primary",
              deleted: false,
            }),
          ],
        },
      },
    });
    expect(deleteDatabaseMapper.getExecutionDetails(ctx)["Status"]).toBe("-");
  });

  it("does not throw when node metadata and configuration are undefined", () => {
    const ctx = buildDetailsCtx({
      node: {
        componentName: "digitalocean.deleteDatabase",
        configuration: undefined,
        metadata: undefined,
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "x",
              databaseClusterId: "id-1",
              deleted: true,
            }),
          ],
        },
      },
    });
    expect(() => deleteDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("deleteDatabaseMapper.props", () => {
  it("prefers metadata database name over configuration database", () => {
    const props = deleteDatabaseMapper.props({
      ...buildComponentContext({
        node: {
          componentName: "digitalocean.deleteDatabase",
          configuration: { databaseCluster: "c1", database: "from_config" },
          metadata: { databaseName: "from_meta", databaseClusterName: "primary" },
        },
        componentDefinition: {
          name: "digitalocean.deleteDatabase",
          label: "Delete Database",
          color: "red",
        },
      }),
    });
    expect(props.metadata).toEqual([
      { icon: "trash-2", label: "from_meta" },
      { icon: "server", label: "primary" },
    ]);
  });

  it("uses configuration database when metadata databaseName is absent", () => {
    const props = deleteDatabaseMapper.props({
      ...buildComponentContext({
        node: {
          componentName: "digitalocean.deleteDatabase",
          configuration: { databaseCluster: "c1", database: "legacy_db" },
          metadata: { databaseClusterName: "cache" },
        },
        componentDefinition: {
          name: "digitalocean.deleteDatabase",
          label: "Delete Database",
          color: "red",
        },
      }),
    });
    expect(props.metadata).toEqual([
      { icon: "trash-2", label: "legacy_db" },
      { icon: "server", label: "cache" },
    ]);
  });
});
