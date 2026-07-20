import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { deleteFirewallRuleMapper } from "./delete_firewall_rule";

const EXECUTED_AT = "2026-06-23T09:01:00Z";

describe("deleteFirewallRuleMapper.props", () => {
  it("surfaces the firewall rule name from the node metadata", () => {
    const props = deleteFirewallRuleMapper.props(
      buildComponentCtx({ metadata: { firewallName: "allow-http" } }, "gcp.compute.deleteFirewallRule"),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "shield", label: "allow-http" })]),
    );
  });

  it("falls back to the configured firewall selfLink", () => {
    const props = deleteFirewallRuleMapper.props(
      buildComponentCtx(
        {
          configuration: {
            firewall: "https://www.googleapis.com/compute/v1/projects/my-project/global/firewalls/allow-http",
          },
        },
        "gcp.compute.deleteFirewallRule",
      ),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "shield", label: "allow-http" })]),
    );
  });
});

describe("deleteFirewallRuleMapper.getExecutionDetails", () => {
  it("maps the deleted firewall rule output", () => {
    const details = deleteFirewallRuleMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: { default: [buildOutput({ name: "allow-http" })] },
        },
      }),
    );
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBe("allow-http");
    expect(Object.keys(details)[0]).toBe("Executed At");
    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
  });

  it("does not throw when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => deleteFirewallRuleMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(deleteFirewallRuleMapper.getExecutionDetails(ctx)["Name"]).toBeUndefined();
  });
});

describe("eventStateRegistry.compute.deleteFirewallRule", () => {
  it("maps success to deleted", () => {
    expect(eventStateRegistry["compute.deleteFirewallRule"].getState(buildExecution())).toBe("deleted");
  });
});
