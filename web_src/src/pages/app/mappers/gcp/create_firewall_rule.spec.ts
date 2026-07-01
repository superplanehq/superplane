import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { createFirewallRuleMapper } from "./create_firewall_rule";

const EXECUTED_AT = "2026-06-23T09:01:00Z";

describe("createFirewallRuleMapper.props", () => {
  it("surfaces name, network, and direction/action", () => {
    const props = createFirewallRuleMapper.props(
      buildComponentCtx(
        { configuration: { name: "allow-http", network: "default", direction: "INGRESS", action: "allow" } },
        "gcp.compute.createFirewallRule",
      ),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "shield", label: "allow-http" }),
        expect.objectContaining({ icon: "network", label: "default" }),
        expect.objectContaining({ icon: "arrow-right-left", label: "INGRESS · ALLOW" }),
      ]),
    );
  });

  it("extracts the network name from a selfLink", () => {
    const props = createFirewallRuleMapper.props(
      buildComponentCtx(
        {
          configuration: {
            name: "allow-http",
            network: "https://www.googleapis.com/compute/v1/projects/my-project/global/networks/prod-vpc",
          },
        },
        "gcp.compute.createFirewallRule",
      ),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "network", label: "prod-vpc" })]),
    );
  });
});

describe("createFirewallRuleMapper.getExecutionDetails", () => {
  it("maps the created firewall rule output with a console link", () => {
    const details = createFirewallRuleMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                name: "allow-http",
                network: "default",
                direction: "INGRESS",
                action: "ALLOW",
                priority: 1000,
                link: "https://console.cloud.google.com/networking/firewalls/details/allow-http?project=my-project",
              }),
            ],
          },
        },
      }),
    );
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBe("allow-http");
    expect(details["Network"]).toBe("default");
    expect(details["Direction"]).toBe("INGRESS");
    expect(details["Action"]).toBe("ALLOW");
    expect(details["Console"]).toContain("console.cloud.google.com");
    // Timestamp first and no more than 6 rows.
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => createFirewallRuleMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(createFirewallRuleMapper.getExecutionDetails(ctx)["Name"]).toBeUndefined();
  });
});

describe("eventStateRegistry.compute.createFirewallRule", () => {
  it("maps success to created", () => {
    expect(eventStateRegistry["compute.createFirewallRule"].getState(buildExecution())).toBe("created");
  });
});
