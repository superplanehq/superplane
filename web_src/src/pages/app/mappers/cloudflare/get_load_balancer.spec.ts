import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { getLoadBalancerMapper } from "./get_load_balancer";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.getLoadBalancer",
  label: "Get Load Balancer",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.loadBalancer", data);

const lbOutputData = {
  loadBalancer: {
    id: "lb123",
    name: "my-lb",
    description: "Primary load balancer",
    enabled: true,
    proxied: true,
    steering_policy: "random",
    session_affinity: "cookie",
    default_pools: ["pool1", "pool2"],
  },
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("getLoadBalancerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no loadBalancer key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => getLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts all load balancer fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    const details = getLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-lb");
    expect(details["Description"]).toBe("Primary load balancer");
    expect(details["Enabled"]).toBe("true");
    expect(details["Proxied"]).toBe("true");
    expect(details["Default Pools"]).toBe("2");
  });

  it("uses dash placeholders when lb fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ loadBalancer: {} })] } },
    });
    const details = getLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("-");
    expect(details["Enabled"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    expect(getLoadBalancerMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("getLoadBalancerMapper.props", () => {
  it("prefers load balancer name from node metadata", () => {
    const props = getLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id" },
          metadata: { loadBalancerName: "Resolved Name" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "Resolved Name" }]);
  });

  it("falls back to load balancer id from configuration when metadata is absent", () => {
    const props = getLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "lb-id" }]);
  });

  it("returns empty metadata when both metadata and configuration are empty", () => {
    const props = getLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.getLoadBalancer", () => {
  it("maps finished success to fetched", () => {
    expect(eventStateRegistry.getLoadBalancer.getState(buildExecution())).toBe("fetched");
  });
});
