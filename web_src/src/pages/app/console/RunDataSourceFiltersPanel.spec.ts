import { describe, expect, it } from "vitest";

import { nextPersistedTriggerRefs, resolveSelectedTriggerIds } from "./runDataSourceTriggerRefs";

const nodes = [
  { id: "trigger-1", name: "deploy", type: "TYPE_TRIGGER" as const, component: "webhook" },
  { id: "trigger-2", name: "release", type: "TYPE_TRIGGER" as const, component: "schedule" },
  { id: "trigger-3", type: "TYPE_TRIGGER" as const, component: "manual" },
];
const ctx = { nodes };

describe("resolveSelectedTriggerIds", () => {
  it("preserves persisted refs while the trigger catalog is loading", () => {
    expect(resolveSelectedTriggerIds(["trigger-1", "deploy"], { nodes: [] })).toEqual(["trigger-1", "deploy"]);
  });
});

describe("nextPersistedTriggerRefs", () => {
  it("keeps friendly names when unchecking one of several named triggers", () => {
    // Regression: previously rewritten the remaining selection to opaque ids.
    expect(
      nextPersistedTriggerRefs({
        triggers: ["deploy", "release"],
        triggerId: "trigger-1",
        selected: true,
        ctx,
      }),
    ).toEqual(["release"]);
  });

  it("persists the trigger name when checking a new trigger", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: ["deploy"],
        triggerId: "trigger-2",
        selected: false,
        ctx,
      }),
    ).toEqual(["deploy", "release"]);
  });

  it("falls back to the node id when the trigger has no name", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: undefined,
        triggerId: "trigger-3",
        selected: false,
        ctx,
      }),
    ).toEqual(["trigger-3"]);
  });

  it("preserves unresolved stale refs when toggling another trigger off", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: ["deploy", "ghost-trigger"],
        triggerId: "trigger-1",
        selected: true,
        ctx,
      }),
    ).toEqual(["ghost-trigger"]);
  });

  it("returns undefined when the last selection is cleared", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: ["release"],
        triggerId: "trigger-2",
        selected: true,
        ctx,
      }),
    ).toBeUndefined();
  });

  it("removes an id ref while the trigger catalog is loading", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: ["trigger-1"],
        triggerId: "trigger-1",
        selected: true,
        ctx: { nodes: [] },
      }),
    ).toBeUndefined();
  });

  it("does not duplicate an id ref while the trigger catalog is loading", () => {
    expect(
      nextPersistedTriggerRefs({
        triggers: ["trigger-1"],
        triggerId: "trigger-1",
        selected: false,
        ctx: { nodes: [] },
      }),
    ).toEqual(["trigger-1"]);
  });
});
