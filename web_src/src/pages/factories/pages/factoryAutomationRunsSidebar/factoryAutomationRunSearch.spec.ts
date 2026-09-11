import { describe, expect, it } from "vitest";

import { factoryAutomationRunMatchesQuery, factoryAutomationRunTitle } from "./factoryAutomationRunSearch";

describe("factoryAutomationRunTitle", () => {
  it("uses the trigger custom name when present", () => {
    expect(factoryAutomationRunTitle({ rootEvent: { customName: "Opened issue" } })).toBe("Opened issue");
  });

  it("falls back to a short run id", () => {
    expect(factoryAutomationRunTitle({ id: "run-abcdef12" })).toBe("Run run-abcd");
  });
});

describe("factoryAutomationRunMatchesQuery", () => {
  it("matches the run title or the linked task title", () => {
    const run = { id: "run-1", rootEvent: { customName: "Event received" } };

    expect(factoryAutomationRunMatchesQuery("refund", run, { title: "Ship refund retries" })).toBe(true);
    expect(factoryAutomationRunMatchesQuery("event", run)).toBe(true);
    expect(factoryAutomationRunMatchesQuery("missing", run, { title: "Ship refund retries" })).toBe(false);
  });
});
