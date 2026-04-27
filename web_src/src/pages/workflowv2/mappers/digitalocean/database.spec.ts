import { describe, expect, it } from "vitest";

import { createDatabaseMapper } from "./create_database";
import { deleteDatabaseMapper } from "./delete_database";
import { eventStateRegistry } from "./index";
import { getClusterConfigurationMapper } from "./get_cluster_configuration";
import { getDatabaseMapper } from "./get_database";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "digitalocean.getDatabase",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "digitalocean.result",
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
    actions: { invokeNodeExecutionHook: async () => {} },
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

// ── getDatabaseMapper ─────────────────────────────────────────────────

describe("getDatabaseMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data fields are all missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => getDatabaseMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts database and cluster fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "app_db",
              databaseClusterName: "superplane-db",
              engine: "pg",
              version: "17",
              region: "nyc1",
              status: "online",
            }),
          ],
        },
      },
    });
    const details = getDatabaseMapper.getExecutionDetails(ctx);
    expect(details["Database"]).toBe("app_db");
    expect(details["Cluster"]).toBe("superplane-db");
    expect(details["Engine"]).toBe("pg");
    expect(details["Version"]).toBe("17");
    expect(details["Region"]).toBe("nyc1");
    expect(details["Status"]).toBe("online");
  });

  it("includes host when connection.host is present", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "app_db",
              connection: { host: "db.example.com", port: 25060 },
            }),
          ],
        },
      },
    });
    expect(getDatabaseMapper.getExecutionDetails(ctx)["Host"]).toBe("db.example.com");
  });

  it("omits host when connection is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "app_db" })] } },
    });
    expect(getDatabaseMapper.getExecutionDetails(ctx)["Host"]).toBeUndefined();
  });
});

// ── getClusterConfigurationMapper ────────────────────────────────────

describe("getClusterConfigurationMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getClusterConfigurationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getClusterConfigurationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("includes cluster name when payload has no config", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              databaseClusterId: "cluster-1",
              databaseClusterName: "my-cluster",
            }),
          ],
        },
      },
    });
    const details = getClusterConfigurationMapper.getExecutionDetails(ctx);
    expect(details["Cluster"]).toBe("my-cluster");
    expect(Object.keys(details).filter((k) => k !== "Executed At" && k !== "Cluster")).toHaveLength(0);
  });

  it("maps known config keys to friendly labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              databaseClusterName: "my-cluster",
              config: {
                autovacuum_naptime: 60,
                max_parallel_workers: 8,
              },
            }),
          ],
        },
      },
    });
    const details = getClusterConfigurationMapper.getExecutionDetails(ctx);
    expect(details["Cluster"]).toBe("my-cluster");
    expect(details["Autovacuum Interval"]).toBe("60");
    expect(details["Parallel Workers"]).toBe("8");
  });

  it("skips null and object config values", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              databaseClusterName: "c",
              config: {
                jit: true,
                nested: { a: 1 },
                empty: null,
              },
            }),
          ],
        },
      },
    });
    const details = getClusterConfigurationMapper.getExecutionDetails(ctx);
    expect(details["Jit"]).toBe("true");
    expect(details["Nested"]).toBeUndefined();
    expect(details["Empty"]).toBeUndefined();
  });

  it("limits total detail rows to six including executed-at and cluster", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              databaseClusterName: "c",
              config: {
                a: 1,
                b: 2,
                c: 3,
                d: 4,
                e: 5,
                f: 6,
              },
            }),
          ],
        },
      },
    });
    const details = getClusterConfigurationMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)).toHaveLength(6);
    expect(details["A"]).toBe("1");
    expect(details["B"]).toBe("2");
    expect(details["C"]).toBe("3");
    expect(details["D"]).toBe("4");
    expect(details["E"]).toBeUndefined();
  });
});

describe("eventStateRegistry (getDatabase, getClusterConfiguration)", () => {
  it("maps finished success to fetched for getDatabase", () => {
    const execution = buildExecution();
    expect(eventStateRegistry.getDatabase.getState(execution)).toBe("fetched");
  });

  it("maps finished success to fetched for getClusterConfiguration", () => {
    const execution = buildExecution();
    expect(eventStateRegistry.getClusterConfiguration.getState(execution)).toBe("fetched");
  });

  it("returns running when execution is in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.getDatabase.getState(execution)).toBe("running");
  });
});
