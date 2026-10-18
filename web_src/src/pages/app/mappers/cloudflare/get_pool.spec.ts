import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { getPoolMapper } from "./get_pool";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.getPool",
  label: "Get Pool",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.pool", data);

const poolOutputData = {
  pool: {
    id: "pool123",
    name: "primary-pool",
    description: "Primary origin pool",
    enabled: true,
    minimum_origins: 1,
    origins: [
      { name: "origin-1", address: "1.2.3.4", enabled: true, weight: 1 },
      { name: "origin-2", address: "5.6.7.8", enabled: true, weight: 1 },
    ],
  },
  accountId: "acc123",
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("getPoolMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getPoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getPoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no pool key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => getPoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts all pool fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    const details = getPoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("pool123");
    expect(details["Name"]).toBe("primary-pool");
    expect(details["Description"]).toBe("Primary origin pool");
    expect(details["Enabled"]).toBe("true");
    expect(details["Minimum Origins"]).toBe("1");
    expect(details["Number of Origins"]).toBe("2");
  });

  it("uses dash placeholders when pool fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ pool: {} })] } },
    });
    const details = getPoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("-");
    expect(details["Name"]).toBe("-");
    expect(details["Enabled"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    expect(getPoolMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("getPoolMapper.props", () => {
  it("prefers pool name from node metadata", () => {
    const props = getPoolMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { pool: "pool-id" },
          metadata: { poolName: "Resolved Name" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "Resolved Name" }]);
  });

  it("falls back to pool id from configuration when metadata is absent", () => {
    const props = getPoolMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { pool: "pool-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "pool-id" }]);
  });

  it("returns empty metadata when both metadata and configuration are empty", () => {
    const props = getPoolMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.getPool", () => {
  it("maps finished success to fetched", () => {
    expect(eventStateRegistry.getPool.getState(buildExecution())).toBe("fetched");
  });
});
