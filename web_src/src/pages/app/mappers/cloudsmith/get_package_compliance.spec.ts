import { describe, expect, it } from "vitest";
import { getPackageComplianceMapper } from "./get_package_compliance";
import { buildDetailsCtx, buildOutput, buildPackageComplianceData } from "./test_helpers";

describe("getPackageComplianceMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getPackageComplianceMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without compliance fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(undefined)] } } });
    const details = getPackageComplianceMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["License"]).toBeUndefined();
  });

  it("extracts the compliance fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildPackageComplianceData())] } },
    });
    const details = getPackageComplianceMapper.getExecutionDetails(ctx);
    expect(details["License"]).toBe("GPL-3.0-only");
    expect(details["OSI Approved"]).toBe("Yes");
    expect(details["Quarantined"]).toBe("Yes");
    expect(details["Policy Violated"]).toBe("No");
    expect(details["Status"]).toBe("Quarantined");
    expect(details["URL"]).toContain("cloudsmith.io");
  });

  it("renders booleans as No when false/undefined", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildPackageComplianceData({ osi_approved: false, is_quarantined: false }))],
        },
      },
    });
    const details = getPackageComplianceMapper.getExecutionDetails(ctx);
    expect(details["OSI Approved"]).toBe("No");
    expect(details["Quarantined"]).toBe("No");
  });
});
