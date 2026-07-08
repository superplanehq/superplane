import { describe, expect, it } from "vitest";

import type { SuperplaneComponentsNode, TriggersTrigger } from "@/api-client";

import { isManualRunNode, manualRunTriggersFromCatalog } from "./ConsoleContext";

const TRIGGERS: TriggersTrigger[] = [
  { name: "start", manualRunnable: true },
  { name: "schedule", manualRunnable: true },
  { name: "github.pullRequest", manualRunnable: false },
  { name: "webhook" },
];

const EVENT_TRIGGER_NODE: SuperplaneComponentsNode = {
  id: "pr-id",
  name: "on-pr",
  type: "TYPE_TRIGGER",
  component: "github.pullRequest",
};

describe("manualRunTriggersFromCatalog", () => {
  it("returns undefined while the catalog is still loading", () => {
    expect(manualRunTriggersFromCatalog(undefined, false)).toBeUndefined();
  });

  it("collects only the manual-runnable trigger names from a loaded catalog", () => {
    const catalog = manualRunTriggersFromCatalog(TRIGGERS, false);
    expect(catalog).toEqual(new Set(["start", "schedule"]));
  });

  it("fails closed with an empty set when the fetch errored without data", () => {
    const catalog = manualRunTriggersFromCatalog(undefined, true);
    expect(catalog).toEqual(new Set());
  });

  it("prefers stale data over failing closed when a refetch errors", () => {
    const catalog = manualRunTriggersFromCatalog(TRIGGERS, true);
    expect(catalog).toEqual(new Set(["start", "schedule"]));
  });
});

describe("isManualRunNode after a failed triggers fetch", () => {
  it("hides Run controls for event-only triggers instead of treating every TYPE_TRIGGER as runnable", () => {
    const manualRunTriggers = manualRunTriggersFromCatalog(undefined, true);
    expect(isManualRunNode({ manualRunTriggers }, EVENT_TRIGGER_NODE)).toBe(false);
  });

  it("keeps the permissive TYPE_TRIGGER fallback only while the catalog is loading", () => {
    const manualRunTriggers = manualRunTriggersFromCatalog(undefined, false);
    expect(isManualRunNode({ manualRunTriggers }, EVENT_TRIGGER_NODE)).toBe(true);
  });
});
