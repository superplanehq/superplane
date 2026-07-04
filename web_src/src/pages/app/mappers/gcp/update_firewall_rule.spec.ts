import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { updateFirewallRuleMapper } from "./update_firewall_rule";

const EXECUTED_AT = "2026-06-23T09:01:00Z";

describe("updateFirewallRuleMapper.props", () => {
  it("surfaces the firewall rule name from the node metadata", () => {
    const props = updateFirewallRuleMapper.props(
      buildComponentCtx({ metadata: { firewallName: "allow-http" } }, "gcp.compute.updateFirewallRule"),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "shield", label: "allow-http" })]),
    );
  });
});

describe("updateFirewallRuleMapper.getExecutionDetails", () => {
  it("maps the updated firewall rule output with a console link", () => {
    const details = updateFirewallRuleMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                name: "allow-http",
                network: "default",
                priority: 900,
                disabled: true,
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
    expect(details["Priority"]).toBe("900");
    expect(details["Enabled"]).toBe("No");
    expect(details["Console"]).toContain("console.cloud.google.com");
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => updateFirewallRuleMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(updateFirewallRuleMapper.getExecutionDetails(ctx)["Name"]).toBeUndefined();
  });
});

describe("eventStateRegistry.compute.updateFirewallRule", () => {
  it("maps success to updated", () => {
    expect(eventStateRegistry["compute.updateFirewallRule"].getState(buildExecution())).toBe("updated");
  });
});
