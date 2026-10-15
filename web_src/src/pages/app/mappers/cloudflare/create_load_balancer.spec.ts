import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.createLoadBalancer",
  label: "Create Load Balancer",
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

describe("createLoadBalancerMapper.getExecutionDetails", () => {
  it("extracts load balancer fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    const details = createLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-lb");
    expect(details["Description"]).toBe("Primary load balancer");
    expect(details["Enabled"]).toBe("true");
    expect(details["Proxied"]).toBe("true");
    expect(details["Default Pools"]).toBe("2");
  });

  it("omits description when it is absent", () => {
    const data = { loadBalancer: { id: "lb1", name: "my-lb", enabled: true, proxied: false, default_pools: [] } };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(data)] } },
    });
    expect(createLoadBalancerMapper.getExecutionDetails(ctx)["Description"]).toBeUndefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createLoadBalancerMapper.props", () => {
  it("includes lb name from configuration in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", enabled: true } }) }),
    );
    expect(props.metadata).toEqual([
      { icon: "network", label: "my-lb" },
      { icon: "check-circle", label: "Enabled" },
    ]);
  });

  it("includes default pools count in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1", "pool2", "pool3"] } }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "3 pools" });
  });

  it("shows singular pool label when only one default pool", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1"] } }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "1 pool" });
  });

  it("omits pools item when defaultPools is empty", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", defaultPools: [] } }) }),
    );
    expect(props.metadata?.find((m) => m.icon === "layers")).toBeUndefined();
  });

  it("shows disabled icon when enabled is false", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", enabled: false } }) }),
    );
    expect(props.metadata).toContainEqual({ icon: "circle", label: "Disabled" });
  });

  it("includes both pools and enabled state in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1", "pool2"], enabled: true } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "network", label: "my-lb" },
      { icon: "layers", label: "2 pools" },
      { icon: "check-circle", label: "Enabled" },
    ]);
  });

  it("omits enabled item when enabled is not set", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "my-lb" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = createLoadBalancerMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.createLoadBalancer", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry.createLoadBalancer.getState(buildExecution())).toBe("created");
  });
});
