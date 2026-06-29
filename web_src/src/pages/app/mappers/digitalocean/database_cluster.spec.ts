import { describe, expect, it } from "vitest";

import { createDatabaseClusterMapper } from "./create_database_cluster";
import { getDatabaseClusterMapper } from "./get_database_cluster";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";

const defaultDefinition: ComponentDefinition = {
  name: "digitalocean.createDatabaseCluster",
  label: "Create Database Cluster",
  description: "",
  icon: "database",
  color: "blue",
};

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "digitalocean.createDatabaseCluster",
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

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: {
      id: "user-1",
      name: "Test User",
      email: "test@example.com",
      roles: [],
      groups: [],
    },
    actions: {
      invokeNodeExecutionHook: async () => {},
    },
    ...overrides,
  };
}

const clusterMappers = [
  { label: "createDatabaseClusterMapper", mapper: createDatabaseClusterMapper },
  { label: "getDatabaseClusterMapper", mapper: getDatabaseClusterMapper },
] as const;

describe.each(clusterMappers)("$label.getExecutionDetails", ({ mapper }) => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => mapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => mapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data fields are all missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => mapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts cluster fields and uses dash placeholders", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "superplane-db",
              engine: "pg",
              version: "18.0",
              region: "nyc1",
              size: "db-s-1vcpu-1gb",
              num_nodes: 1,
              status: "online",
            }),
          ],
        },
      },
    });
    const details = mapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("superplane-db");
    expect(details["Engine"]).toBe("pg");
    expect(details["Version"]).toBe("18.0");
    expect(details["Region"]).toBe("nyc1");
    expect(details["Size"]).toBe("db-s-1vcpu-1gb");
    expect(details["Node Count"]).toBe("1");
    expect(details["Status"]).toBe("online");
  });

  it("includes Host when connection.host is present", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "db",
              connection: { host: "db.example.com", port: 25060 },
            }),
          ],
        },
      },
    });
    expect(mapper.getExecutionDetails(ctx)["Host"]).toBe("db.example.com");
  });

  it("omits Host when connection is missing or has no host", () => {
    const withoutConnection = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "db" })] } },
    });
    expect(mapper.getExecutionDetails(withoutConnection)["Host"]).toBeUndefined();

    const emptyHost = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ connection: { port: 123 } })] } },
    });
    expect(mapper.getExecutionDetails(emptyHost)["Host"]).toBeUndefined();
  });
});

describe("createDatabaseClusterMapper.props", () => {
  it("includes name, engine, and region in metadata when configured", () => {
    const props = createDatabaseClusterMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {
            name: "my-cluster",
            engine: "pg",
            region: "nyc1",
          },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "my-cluster" },
      { icon: "cpu", label: "pg" },
      { icon: "map-pinned", label: "nyc1" },
    ]);
  });

  it("omits metadata entries when configuration fields are absent", () => {
    const props = createDatabaseClusterMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "only-name" } }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "database", label: "only-name" }]);
  });
});

describe("getDatabaseClusterMapper.props", () => {
  it("prefers database cluster name from node metadata", () => {
    const props = getDatabaseClusterMapper.props(
      buildPropsContext({
        node: buildNode({
          componentName: "digitalocean.getDatabaseCluster",
          metadata: { databaseClusterName: "Resolved Name" },
          configuration: { databaseCluster: "cluster-uuid" },
        }),
        componentDefinition: {
          ...defaultDefinition,
          name: "digitalocean.getDatabaseCluster",
          label: "Get Database Cluster",
        },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "database", label: "Resolved Name" }]);
  });

  it("falls back to cluster id label when name metadata is absent", () => {
    const props = getDatabaseClusterMapper.props(
      buildPropsContext({
        node: buildNode({
          componentName: "digitalocean.getDatabaseCluster",
          metadata: {},
          configuration: { databaseCluster: "65b497a5-1674-4b1a-a122-01aebe761ef7" },
        }),
        componentDefinition: {
          ...defaultDefinition,
          name: "digitalocean.getDatabaseCluster",
          label: "Get Database Cluster",
        },
      }),
    );
    expect(props.metadata).toEqual([{ icon: "info", label: "Cluster ID: 65b497a5-1674-4b1a-a122-01aebe761ef7" }]);
  });
});
