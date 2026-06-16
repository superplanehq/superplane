import { describe, expect, it } from "vitest";

import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";
import { createAlertingPolicyMapper } from "./monitoring";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("createAlertingPolicyMapper.getExecutionDetails (PromQL)", () => {
  it("surfaces the created PromQL policy details", () => {
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
    expect(details["Policy ID"]).toBe("123");
    expect(details["Enabled"]).toBe("Yes");
    expect(details["Severity"]).toBe("WARNING");
    expect(details["Conditions"]).toBe("1");
  });
});
