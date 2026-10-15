import { describe, expect, it } from "bun:test";

import type { ComponentDefinition } from "../types";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import { eventStateRegistry } from "./index";
import { buildExecution, buildOutput as buildCloudflareOutput, createComponentFixtures } from "./test_support";

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.deleteLoadBalancer",
  label: "Delete Load Balancer",
  description: "",
  icon: "cloud",
  color: "orange",
};

const { buildNode, buildDetailsCtx, buildPropsContext } = createComponentFixtures(defaultDefinition);
const buildOutput = (data: unknown) => buildCloudflareOutput("cloudflare.loadBalancer", data);

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("deleteLoadBalancerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts load balancer id and deleted status from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ loadBalancerId: "lb123", deleted: true })],
        },
      },
    });
    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Deleted"]).toBe("true");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Deleted"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ loadBalancerId: "lb123", deleted: true })] },
      },
    });
    expect(deleteLoadBalancerMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("deleteLoadBalancerMapper.props", () => {
  it("prefers load balancer name from node metadata", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id" },
          metadata: { loadBalancerName: "My LB" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "My LB" }]);
  });

  it("falls back to load balancer id from configuration when metadata is absent", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id" },
          metadata: {},
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "lb-id" }]);
  });

  it("returns empty metadata when both metadata and configuration are empty", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.deleteLoadBalancer", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteLoadBalancer.getState(buildExecution())).toBe("deleted");
  });
});
