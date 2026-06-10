import { describe, expect, it } from "vitest";
import { queryMapper } from "./prometheus";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("queryMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => queryMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = queryMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Result Type"]).toBeUndefined();
  });

  it("surfaces the result type and series count", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ resultType: "vector", seriesCount: 3 })] } },
    });
    const details = queryMapper.getExecutionDetails(ctx);
    expect(details["Result Type"]).toBe("vector");
    expect(details["Series"]).toBe("3");
  });

  it("renders a zero series count rather than omitting it", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ resultType: "vector", seriesCount: 0 })] } },
    });
    const details = queryMapper.getExecutionDetails(ctx);
    expect(details["Series"]).toBe("0");
  });

  it("surfaces a matrix result type for a range (lookback) query", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ resultType: "matrix", seriesCount: 2 })] } },
    });
    const details = queryMapper.getExecutionDetails(ctx);
    expect(details["Result Type"]).toBe("matrix");
    expect(details["Series"]).toBe("2");
  });
});

describe("queryMapper.props lookback metadata", () => {
  function node(configuration: Record<string, unknown>) {
    return {
      id: "n1",
      name: "Query",
      componentName: "gcp.prometheus.query",
      isCollapsed: false,
      configuration,
      metadata: {},
    };
  }
  const ctx = (configuration: Record<string, unknown>) =>
    ({
      node: node(configuration),
      nodes: [],
      lastExecutions: [],
      componentDefinition: { name: "gcp.prometheus.query", label: "Query", icon: "chart-line" },
    }) as unknown as Parameters<typeof queryMapper.props>[0];

  it("shows a lookback chip when a window is set", () => {
    const props = queryMapper.props(ctx({ query: "up", lookbackPeriod: "1h" }));
    expect(props.metadata?.some((m) => m.label === "last 1 hour")).toBe(true);
  });

  it("omits the lookback chip for an instant query", () => {
    const props = queryMapper.props(ctx({ query: "up", lookbackPeriod: "instant" }));
    expect(props.metadata?.some((m) => m.icon === "clock")).toBe(false);
  });
});
