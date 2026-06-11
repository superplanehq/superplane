import { describe, expect, it } from "vitest";
import { queryMapper, queryRangeMapper } from "./prometheus";
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
});

describe("queryRangeMapper.getExecutionDetails", () => {
  it("surfaces the matrix result type and series count", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ resultType: "matrix", seriesCount: 2 })] } },
    });
    const details = queryRangeMapper.getExecutionDetails(ctx);
    expect(details["Result Type"]).toBe("matrix");
    expect(details["Series"]).toBe("2");
  });
});
