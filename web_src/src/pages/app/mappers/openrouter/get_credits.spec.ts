import { describe, expect, it } from "vitest";

import { getCreditsMapper } from "./get_credits";
import { buildDetailsCtx, buildOutput } from "./test_helpers";

const PAYLOAD_TYPE = "openrouter.getCredits.result";

function detailsFor(data: unknown) {
  return getCreditsMapper.getExecutionDetails!(buildDetailsCtx(buildOutput(PAYLOAD_TYPE, data)));
}

describe("openrouter getCreditsMapper execution details", () => {
  it("surfaces the balance, account totals and key usage", () => {
    const details = detailsFor({
      totalCredits: 100.5,
      totalUsage: 25.75,
      balance: 74.75,
      key: { label: "sk-or-v1-...c0de", limit: 50, limitRemaining: 24.25, usageMonthly: 25.75 },
    });

    expect(Object.keys(details)).toEqual([
      "Fetched At",
      "Balance",
      "Total Credits",
      "Total Usage",
      "Key Usage (Month)",
      "Key Limit Remaining",
    ]);
    expect(details["Balance"]).toBe("$74.75");
    expect(details["Total Credits"]).toBe("$100.50");
    expect(details["Key Limit Remaining"]).toBe("$24.25");
  });

  it("keeps the timestamp first and stays within six details", () => {
    const details = detailsFor({
      totalCredits: 1,
      totalUsage: 1,
      balance: 0,
      key: { usageMonthly: 1, limitRemaining: 1 },
    });

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(Object.keys(details)[0]).toBe("Fetched At");
  });

  it("shows a zero balance rather than omitting it", () => {
    expect(detailsFor({ balance: 0 })["Balance"]).toBe("$0.00");
  });

  it("omits the key limit when the key has none", () => {
    const details = detailsFor({ balance: 10, key: { limit: null, limitRemaining: null } });
    expect(details["Key Limit Remaining"]).toBeUndefined();
  });

  it("returns nothing when the execution has no output", () => {
    expect(getCreditsMapper.getExecutionDetails!(buildDetailsCtx())).toEqual({});
  });
});
