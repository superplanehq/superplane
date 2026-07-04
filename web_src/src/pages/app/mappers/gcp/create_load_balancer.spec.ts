import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { createLoadBalancerMapper } from "./create_load_balancer";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("createLoadBalancerMapper.props", () => {
  it("surfaces name, region, and protocol/ports", () => {
    const props = createLoadBalancerMapper.props(
      buildComponentCtx(
        { configuration: { name: "web-lb", region: "us-central1", protocol: "TCP", ports: ["80", "443"] } },
        "gcp.compute.createLoadBalancer",
      ),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "globe", label: "web-lb" }),
        expect.objectContaining({ icon: "map-pin", label: "us-central1" }),
        expect.objectContaining({ icon: "ethernet-port", label: "TCP 80, 443" }),
      ]),
    );
  });
});

describe("createLoadBalancerMapper.getExecutionDetails", () => {
  it("maps the created load balancer output", () => {
    const details = createLoadBalancerMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                name: "web-lb",
                region: "us-central1",
                ipAddress: "34.1.2.3",
                protocol: "TCP",
                ports: ["80", "443"],
                forwardingRule: "web-lb-fr",
                backendService: "web-lb-backend",
                healthCheck: "web-lb-hc",
              }),
            ],
          },
        },
      }),
    );
    expect(details["Name"]).toBe("web-lb");
    expect(details["Region"]).toBe("us-central1");
    expect(details["IP Address"]).toBe("34.1.2.3");
    expect(details["Backend Service"]).toBe("web-lb-backend");
    // Trimmed to a max of 5 rows (incl. Executed At).
    expect(details["Protocol"]).toBeUndefined();
    expect(details["Ports"]).toBeUndefined();
    expect(details["Forwarding Rule"]).toBeUndefined();
    expect(details["Health Check"]).toBeUndefined();
    expect(Object.keys(details).length).toBeLessThanOrEqual(5);
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => createLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(createLoadBalancerMapper.getExecutionDetails(ctx)["Name"]).toBeUndefined();
  });
});

describe("eventStateRegistry.compute.createLoadBalancer", () => {
  it("maps success to created", () => {
    expect(eventStateRegistry["compute.createLoadBalancer"].getState(buildExecution())).toBe("created");
  });
});
