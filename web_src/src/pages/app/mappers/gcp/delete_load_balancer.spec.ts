import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("deleteLoadBalancerMapper.props", () => {
  it("shows the load balancer name from the forwarding rule value", () => {
    const props = deleteLoadBalancerMapper.props(
      buildComponentCtx(
        {
          configuration: {
            loadBalancer:
              "https://www.googleapis.com/compute/v1/projects/my-project/regions/us-central1/forwardingRules/web-lb-fr",
          },
        },
        "gcp.compute.deleteLoadBalancer",
      ),
    );
    expect(props.metadata).toEqual([expect.objectContaining({ icon: "globe", label: "web-lb-fr" })]);
  });

  it("shows nothing for an unresolved expression", () => {
    const props = deleteLoadBalancerMapper.props(
      buildComponentCtx(
        { configuration: { loadBalancer: '{{ $["Create Load Balancer"].data.forwardingRule }}' } },
        "gcp.compute.deleteLoadBalancer",
      ),
    );
    expect(props.metadata).toEqual([]);
  });
});

describe("deleteLoadBalancerMapper.getExecutionDetails", () => {
  it("maps what was deleted", () => {
    const details = deleteLoadBalancerMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                forwardingRule: "web-lb-fr",
                backendService: "web-lb-backend",
                healthCheck: "web-lb-hc",
                region: "us-central1",
              }),
            ],
          },
        },
      }),
    );
    expect(details["Forwarding Rule"]).toBe("web-lb-fr");
    expect(details["Backend Service"]).toBe("web-lb-backend");
    expect(details["Health Check"]).toBe("web-lb-hc");
    expect(details["Region"]).toBe("us-central1");
  });
});

describe("eventStateRegistry.compute.deleteLoadBalancer", () => {
  it("maps success to deleted", () => {
    expect(eventStateRegistry["compute.deleteLoadBalancer"].getState(buildExecution())).toBe("deleted");
  });
});
