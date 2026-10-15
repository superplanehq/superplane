import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { eventStateRegistry } from "./index";
import { updateLoadBalancerMapper } from "./update_load_balancer";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.updateLoadBalancer",
  label: "Update Load Balancer",
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

describe("updateLoadBalancerMapper.getExecutionDetails", () => {
  it("extracts load balancer fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    const details = updateLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-lb");
    expect(details["Description"]).toBe("Primary load balancer");
    expect(details["Enabled"]).toBe("true");
    expect(details["Proxied"]).toBe("true");
    expect(details["Default Pools"]).toBe("2");
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("updateLoadBalancerMapper.props", () => {
  it("prefers load balancer name from node metadata", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id", steeringPolicy: "random" },
          metadata: { loadBalancerName: "Resolved Name" },
        }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "network", label: "Resolved Name" });
  });

  it("falls back to load balancer id from configuration when metadata is absent", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "lb-id" }]);
  });

  it("shows description in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", description: "My LB" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "text", label: "My LB" });
  });

  it("omits description when not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "text")).toBeUndefined();
  });

  it("shows steering policy in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", steeringPolicy: "geo" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "git-branch", label: "geo" });
  });

  it("omits steering policy when not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).not.toContain("geo");
    expect(props.metadata).toHaveLength(1);
  });

  it("shows default pools count in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: ["pool1", "pool2"] }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "2 pools" });
  });

  it("shows singular pool label when only one default pool", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: ["pool1"] }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "1 pool" });
  });

  it("omits pools item when defaultPools is empty", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: [] }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "layers")).toBeUndefined();
  });

  it("shows enabled state in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", enabled: true }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "check-circle", label: "Enabled" });
  });

  it("shows disabled state in metadata when enabled is false", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", enabled: false }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "circle", label: "Disabled" });
  });

  it("omits enabled item when enabled is not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "check-circle" || m.icon === "circle")).toBeUndefined();
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.updateLoadBalancer", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry.updateLoadBalancer.getState(buildExecution())).toBe("updated");
  });
});
