import { describe, expect, it } from "vitest";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { resolveNodeComponentMetadata } from "./resolveNodeComponentMetadata";

function node(overrides: Partial<ComponentsNode>): ComponentsNode {
  return {
    id: "node-1",
    name: "Node",
    type: "TYPE_ACTION",
    component: "noop",
    ...overrides,
  } as ComponentsNode;
}

describe("resolveNodeComponentMetadata", () => {
  const components = new Map([["noop", { label: "Noop", configuration: [{ name: "field", type: "string" }] }]]);
  const triggers = new Map([["webhook", { label: "Webhook", configuration: [{ name: "path", type: "string" }] }]]);

  it("resolves action metadata by component name", () => {
    expect(
      resolveNodeComponentMetadata(node({ type: "TYPE_ACTION", component: "noop" }), components, triggers),
    ).toEqual({
      label: "Noop",
      configuration: [{ name: "field", type: "string" }],
    });
  });

  it("resolves trigger metadata by component name", () => {
    expect(
      resolveNodeComponentMetadata(node({ type: "TYPE_TRIGGER", component: "webhook" }), components, triggers),
    ).toEqual({
      label: "Webhook",
      configuration: [{ name: "path", type: "string" }],
    });
  });

  it("returns undefined when the component is missing", () => {
    expect(resolveNodeComponentMetadata(node({ component: undefined }), components, triggers)).toBeUndefined();
  });
});
