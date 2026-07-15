import { describe, expect, it } from "vitest";

import type { CanvasesCanvasRun } from "@/api-client";

import {
  hasRunStatusTriggerFilters,
  runMatchesStatusTriggerFilters,
  runSelectStatusFilterCanMatch,
  statusesCompatibleWithRunSelect,
  triggerFilterCanMatch,
} from "./runStatusTriggerFilter";

function run(overrides: Partial<CanvasesCanvasRun> = {}): CanvasesCanvasRun {
  return {
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    rootEvent: { nodeId: "trigger-1" },
    ...overrides,
  };
}

describe("runMatchesStatusTriggerFilters", () => {
  it("passes everything when filters are undefined or empty", () => {
    expect(runMatchesStatusTriggerFilters(run(), undefined)).toBe(true);
    expect(runMatchesStatusTriggerFilters(run(), { statuses: [], triggers: [] })).toBe(true);
  });

  it("filters by run status via getRunStatus derivation", () => {
    const passed = run();
    const failed = run({ result: "RESULT_FAILED" });
    const running = run({ state: "STATE_STARTED", result: undefined });
    const cancelled = run({ result: "RESULT_CANCELLED" });

    expect(runMatchesStatusTriggerFilters(passed, { statuses: ["failed", "cancelled"] })).toBe(false);
    expect(runMatchesStatusTriggerFilters(failed, { statuses: ["failed", "cancelled"] })).toBe(true);
    expect(runMatchesStatusTriggerFilters(running, { statuses: ["running"] })).toBe(true);
    expect(runMatchesStatusTriggerFilters(cancelled, { statuses: ["cancelled"] })).toBe(true);
  });

  it("drops runs with unknown status when status filter is set", () => {
    const unknown = run({ state: undefined, result: undefined });
    expect(runMatchesStatusTriggerFilters(unknown, { statuses: ["passed"] })).toBe(false);
  });

  it("filters by trigger node id when references already match", () => {
    expect(runMatchesStatusTriggerFilters(run(), { triggers: ["trigger-1"] })).toBe(true);
    expect(runMatchesStatusTriggerFilters(run(), { triggers: ["other"] })).toBe(false);
  });

  it("resolves trigger references (id-or-name) through the provided resolver", () => {
    const resolver = (reference: string) => (reference === "deploy" ? "trigger-1" : undefined);
    expect(runMatchesStatusTriggerFilters(run(), { triggers: ["deploy"] }, resolver)).toBe(true);
    expect(runMatchesStatusTriggerFilters(run(), { triggers: ["release"] }, resolver)).toBe(false);
  });

  it("rejects every run when a resolver is set but no trigger reference resolves", () => {
    const resolver = () => undefined;
    expect(runMatchesStatusTriggerFilters(run(), { triggers: ["gone"] }, resolver)).toBe(false);
  });

  it("drops runs whose rootEvent lacks a nodeId when trigger filter is set", () => {
    const orphan = run({ rootEvent: {} });
    expect(runMatchesStatusTriggerFilters(orphan, { triggers: ["trigger-1"] })).toBe(false);
  });

  it("requires both dimensions to match when both are set (AND across, OR within)", () => {
    const failedFromTrigger1 = run({ result: "RESULT_FAILED" });
    expect(runMatchesStatusTriggerFilters(failedFromTrigger1, { statuses: ["failed"], triggers: ["trigger-1"] })).toBe(
      true,
    );
    expect(runMatchesStatusTriggerFilters(failedFromTrigger1, { statuses: ["passed"], triggers: ["trigger-1"] })).toBe(
      false,
    );
    expect(runMatchesStatusTriggerFilters(failedFromTrigger1, { statuses: ["failed"], triggers: ["trigger-2"] })).toBe(
      false,
    );
  });
});

describe("hasRunStatusTriggerFilters", () => {
  it("returns false for undefined / empty selections", () => {
    expect(hasRunStatusTriggerFilters(undefined)).toBe(false);
    expect(hasRunStatusTriggerFilters({})).toBe(false);
    expect(hasRunStatusTriggerFilters({ statuses: [], triggers: [] })).toBe(false);
  });

  it("returns true when either dimension has a non-empty selection", () => {
    expect(hasRunStatusTriggerFilters({ statuses: ["failed"] })).toBe(true);
    expect(hasRunStatusTriggerFilters({ triggers: ["deploy"] })).toBe(true);
    expect(hasRunStatusTriggerFilters({ statuses: ["running"], triggers: ["release"] })).toBe(true);
  });
});

describe("triggerFilterCanMatch", () => {
  it("returns true when there is no trigger filter", () => {
    expect(triggerFilterCanMatch(undefined)).toBe(true);
    expect(triggerFilterCanMatch([])).toBe(true);
  });

  it("returns true without a resolver (compare refs as-is)", () => {
    expect(triggerFilterCanMatch(["deploy"])).toBe(true);
  });

  it("returns false when every reference fails to resolve", () => {
    expect(triggerFilterCanMatch(["gone"], () => undefined, { nodeCatalogLoading: false })).toBe(false);
  });

  it("returns true when at least one reference resolves", () => {
    const resolver = (reference: string) => (reference === "deploy" ? "id-1" : undefined);
    expect(triggerFilterCanMatch(["gone", "deploy"], resolver)).toBe(true);
  });

  it("stays optimistic only while the node catalog is loading", () => {
    const resolver = () => undefined;
    expect(triggerFilterCanMatch(["deploy"], resolver, { nodeCatalogLoading: true })).toBe(true);
    expect(triggerFilterCanMatch(["deploy"], resolver, { nodeCatalogLoading: false })).toBe(false);
  });
});

describe("runMatchesStatusTriggerFilters with an unresolved node catalog", () => {
  it("falls back to raw id comparison while the catalog is loading", () => {
    const resolver = () => undefined;
    const matched = run({ rootEvent: { nodeId: "trigger-1" } });
    expect(
      runMatchesStatusTriggerFilters(matched, { triggers: ["trigger-1"] }, resolver, { nodeCatalogLoading: true }),
    ).toBe(true);
    expect(
      runMatchesStatusTriggerFilters(matched, { triggers: ["deploy"] }, resolver, { nodeCatalogLoading: true }),
    ).toBe(false);
  });

  it("treats fully unresolved refs as stale once loading settles, including an empty canvas", () => {
    const resolver = () => undefined;
    expect(
      runMatchesStatusTriggerFilters(run(), { triggers: ["trigger-1"] }, resolver, { nodeCatalogLoading: false }),
    ).toBe(false);
  });
});

describe("runSelectStatusFilterCanMatch", () => {
  it("allows any status filter on the unfiltered latest bucket", () => {
    expect(runSelectStatusFilterCanMatch("latest", ["failed"])).toBe(true);
    expect(runSelectStatusFilterCanMatch("latest", undefined)).toBe(true);
  });

  it("rejects status filters that can never appear in a passed/failed bucket", () => {
    expect(runSelectStatusFilterCanMatch("latest_passed", ["failed"])).toBe(false);
    expect(runSelectStatusFilterCanMatch("latest_passed", ["running", "cancelled"])).toBe(false);
    expect(runSelectStatusFilterCanMatch("latest_failed", ["passed"])).toBe(false);
  });

  it("allows a passed/failed filter when it includes the bucket status", () => {
    expect(runSelectStatusFilterCanMatch("latest_passed", ["passed"])).toBe(true);
    expect(runSelectStatusFilterCanMatch("latest_passed", ["passed", "failed"])).toBe(true);
    expect(runSelectStatusFilterCanMatch("latest_failed", ["failed", "cancelled"])).toBe(true);
  });
});

describe("statusesCompatibleWithRunSelect", () => {
  it("keeps only the status achievable in the selected bucket", () => {
    expect(statusesCompatibleWithRunSelect("latest_passed", ["passed", "failed"])).toEqual(["passed"]);
    expect(statusesCompatibleWithRunSelect("latest_failed", ["failed"])).toEqual(["failed"]);
    expect(statusesCompatibleWithRunSelect("latest_passed", ["failed"])).toBeUndefined();
  });

  it("leaves latest selections unchanged", () => {
    expect(statusesCompatibleWithRunSelect("latest", ["running", "failed"])).toEqual(["running", "failed"]);
  });
});
