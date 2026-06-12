import { describe, expect, it } from "vitest";
import { createInstanceMapper, getInstanceMapper, deleteInstanceMapper } from "./cloudsql_mapper";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("cloudsql instance mappers getExecutionDetails", () => {
  it("createInstance surfaces the ready instance with the timestamp first", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-instance",
              state: "RUNNABLE",
              databaseVersion: "POSTGRES_16",
              connectionName: "p:us-central1:my-instance",
              ipAddress: "34.41.10.20",
            }),
          ],
        },
      },
    });
    const details = createInstanceMapper.getExecutionDetails(ctx);
    // Timestamp first, then at most five fields total.
    expect(Object.keys(details)[0]).toBe("Completed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(5);
    expect(details["State"]).toBe("RUNNABLE");
    expect(details["Version"]).toBe("POSTGRES_16");
    expect(details["IP Address"]).toBe("34.41.10.20");
  });

  it("getInstance surfaces the instance details", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-instance",
              state: "RUNNABLE",
              databaseVersion: "POSTGRES_16",
              connectionName: "p:us-central1:my-instance",
              ipAddress: "34.41.10.20",
            }),
          ],
        },
      },
    });
    const details = getInstanceMapper.getExecutionDetails(ctx);
    expect(Object.keys(details).length).toBeLessThanOrEqual(5);
    expect(details["State"]).toBe("RUNNABLE");
    expect(details["Connection"]).toBe("p:us-central1:my-instance");
    expect(details["IP Address"]).toBe("34.41.10.20");
  });

  it("deleteInstance confirms the deletion", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ name: "my-instance", deleted: true })] } },
    });
    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Instance"]).toBe("my-instance");
    expect(details["Deleted"]).toBe("true");
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
