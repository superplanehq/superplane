import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { createPoolMapper } from "./create_pool";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.createPool",
  label: "Create Pool",
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

describe("createPoolMapper.getExecutionDetails", () => {
  it("extracts pool fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    const details = createPoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("pool123");
    expect(details["Name"]).toBe("primary-pool");
    expect(details["Description"]).toBe("Primary origin pool");
    expect(details["Enabled"]).toBe("true");
    expect(details["Minimum Origins"]).toBe("1");
    expect(details["Number of Origins"]).toBe("2");
  });

  it("omits description when it is absent", () => {
    const data = { pool: { id: "p1", name: "my-pool", enabled: true, minimum_origins: 1, origins: [] } };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(data)] } },
    });
    expect(createPoolMapper.getExecutionDetails(ctx)["Description"]).toBeUndefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createPoolMapper.props", () => {
  it("includes pool name from configuration in metadata", () => {
    const props = createPoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-pool", enabled: true } }) }),
    );
    expect(props.metadata).toEqual([
      { icon: "network", label: "my-pool" },
      { icon: "check-circle", label: "Enabled" },
    ]);
  });

  it("shows disabled icon when enabled is false", () => {
    const props = createPoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-pool", enabled: false } }) }),
    );
    expect(props.metadata).toContainEqual({ icon: "circle", label: "Disabled" });
  });

  it("omits enabled item when enabled is not set", () => {
    const props = createPoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-pool" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "my-pool" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = createPoolMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.createPool", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry.createPool.getState(buildExecution())).toBe("created");
  });
});
