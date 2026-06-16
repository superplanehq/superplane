import { describe, expect, it } from "vitest";

import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";
import { createAlertingPolicyMapper, updateAlertingPolicyMapper } from "./monitoring";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("createAlertingPolicyMapper.getExecutionDetails (PromQL)", () => {
  it("surfaces the created PromQL policy details without the policy ID", () => {
    const details = createAlertingPolicyMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                displayName: "Demo PromQL Policy",
                id: "123",
                enabled: true,
                severity: "WARNING",
                conditionsCount: 1,
              }),
            ],
          },
        },
      }),
    );
    expect(details["Display Name"]).toBe("Demo PromQL Policy");
    expect(details["Policy ID"]).toBeUndefined();
    expect(details["Enabled"]).toBe("Yes");
    expect(details["Severity"]).toBe("WARNING");
    expect(details["Conditions"]).toBe("1");
  });
});

describe("updateAlertingPolicyMapper.getExecutionDetails", () => {
  it("omits the policy ID and first condition", () => {
    const details = updateAlertingPolicyMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({
                displayName: "Updated Policy",
                id: "123",
                severity: "CRITICAL",
                comparison: "COMPARISON_GT",
                thresholdValue: 0.8,
                conditionsCount: 1,
              }),
            ],
          },
        },
      }),
    );
    expect(details["Display Name"]).toBe("Updated Policy");
    expect(details["Severity"]).toBe("CRITICAL");
    expect(details["Policy ID"]).toBeUndefined();
    expect(details["First Condition"]).toBeUndefined();
  });
});
