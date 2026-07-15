import { describe, expect, it } from "vitest";

import { resolveConsoleNode, resolveConsoleTrigger } from "./ConsoleContext";

const nodes = [
  { id: "action-1", name: "deploy", type: "TYPE_ACTION" as const, component: "http" },
  { id: "trigger-1", name: "deploy", type: "TYPE_TRIGGER" as const, component: "webhook" },
  { id: "trigger-2", name: "release", type: "TYPE_TRIGGER" as const, component: "schedule" },
];

describe("resolveConsoleTrigger", () => {
  it("resolves triggers by id", () => {
    expect(resolveConsoleTrigger({ nodes }, "trigger-2")?.node.id).toBe("trigger-2");
  });

  it("resolves triggers by name", () => {
    expect(resolveConsoleTrigger({ nodes }, "release")?.node.id).toBe("trigger-2");
  });

  it("ignores non-trigger nodes that share a name", () => {
    // resolveConsoleNode would bind "deploy" to the action (first match);
    // trigger resolution must prefer the TYPE_TRIGGER namesake.
    expect(resolveConsoleNode({ nodes }, "deploy")?.node.id).toBe("action-1");
    expect(resolveConsoleTrigger({ nodes }, "deploy")?.node.id).toBe("trigger-1");
  });

  it("returns undefined for action-only references", () => {
    expect(resolveConsoleTrigger({ nodes }, "action-1")).toBeUndefined();
  });

  it("returns undefined for blank or missing context", () => {
    expect(resolveConsoleTrigger({ nodes }, "  ")).toBeUndefined();
    expect(resolveConsoleTrigger(undefined, "deploy")).toBeUndefined();
  });
});
