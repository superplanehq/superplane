import { describe, expect, it } from "vitest";
import { createInstanceMapper, getInstanceMapper, deleteInstanceMapper } from "./cloudsql_mapper";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("cloudsql instance mappers getExecutionDetails", () => {
  it("createInstance surfaces the operation result", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "my-instance", operation: "op-1", state: "PENDING_CREATE" })] } },
    });
    const details = createInstanceMapper.getExecutionDetails(ctx);
    expect(details["Instance"]).toBe("my-instance");
    expect(details["State"]).toBe("PENDING_CREATE");
    expect(details["Completed At"]).toBeDefined();
  });

  it("getInstance surfaces the instance details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-instance",
              state: "RUNNABLE",
              region: "us-central1",
              tier: "db-f1-micro",
              connectionName: "p:us-central1:my-instance",
              ipAddress: "34.41.10.20",
            }),
          ],
        },
      },
    });
    const details = getInstanceMapper.getExecutionDetails(ctx);
    expect(details["State"]).toBe("RUNNABLE");
    expect(details["Tier"]).toBe("db-f1-micro");
    expect(details["IP Address"]).toBe("34.41.10.20");
  });

  it("deleteInstance marks the instance as deleting", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "my-instance", operation: "op-2", deleting: true })] } },
    });
    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Instance"]).toBe("my-instance");
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getInstanceMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});

describe("cloudsql instance mappers props metadata", () => {
  function propsCtx(configuration: Record<string, unknown>) {
    return {
      node: {
        id: "n1",
        name: "Create Instance",
        componentName: "gcp.cloudsql.createInstance",
        isCollapsed: false,
        configuration,
        metadata: {},
      },
      nodes: [],
      lastExecutions: [],
      componentDefinition: { name: "gcp.cloudsql.createInstance", label: "Create Instance", icon: "database" },
    } as unknown as Parameters<typeof createInstanceMapper.props>[0];
  }

  it("shows the instance name and version as chips", () => {
    const props = createInstanceMapper.props(propsCtx({ name: "my-instance", databaseVersion: "POSTGRES_16" }));
    expect(props.metadata?.some((m) => m.label === "my-instance")).toBe(true);
    expect(props.metadata?.some((m) => m.label === "POSTGRES_16")).toBe(true);
  });

  it("hides unresolved expression values", () => {
    const props = getInstanceMapper.props(propsCtx({ instance: "{{ $.inputs.instance }}" }));
    expect(props.metadata?.length).toBe(0);
  });
});
