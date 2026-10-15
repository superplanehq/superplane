import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { eventStateRegistry } from "./index";
import { updatePoolMapper } from "./update_pool";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.updatePool",
  label: "Update Pool",
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

describe("updatePoolMapper.getExecutionDetails", () => {
  it("extracts pool fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    const details = updatePoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("pool123");
    expect(details["Name"]).toBe("primary-pool");
    expect(details["Description"]).toBe("Primary origin pool");
    expect(details["Enabled"]).toBe("true");
    expect(details["Minimum Origins"]).toBe("1");
    expect(details["Number of Origins"]).toBe("2");
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("updatePoolMapper.props", () => {
  it("shows pool name from configuration when set", () => {
    const props = updatePoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-pool", pool: "pool-id" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "my-pool" }]);
  });

  it("falls back to pool id when name is absent", () => {
    const props = updatePoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { pool: "pool-id" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "pool-id" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = updatePoolMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.updatePool", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry.updatePool.getState(buildExecution())).toBe("updated");
  });
});
